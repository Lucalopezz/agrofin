package pagamentos

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
)

type harvestSummarizer interface {
	Summarize(ctx context.Context, userID string, period colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error)
}

type dailyWorkSummarizer interface {
	Summarize(ctx context.Context, userID string, period diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error)
}

// Service combina projeções já calculadas nos domínios de colheita e diária.
// Ele não carrega histórico dentro de Employee nem altera os lançamentos de origem.
type Service struct {
	harvest harvestSummarizer
	daily   dailyWorkSummarizer
}

func NewService(harvest harvestSummarizer, daily dailyWorkSummarizer) *Service {
	return &Service{harvest: harvest, daily: daily}
}

// Summarize calcula o valor devido antes de pagamentos para cada funcionário
// com ao menos uma colheita ou diária no intervalo inclusivo.
func (s *Service) Summarize(ctx context.Context, userID string, period Period) (PeriodSummary, error) {
	// Não iniciamos consultas se o cliente já cancelou ou perdeu o prazo.
	if err := ctx.Err(); err != nil {
		return PeriodSummary{}, err
	}
	// O dono vem da sessão do handler; validar aqui também protege chamadas
	// internas que usem Service sem passar pelo HTTP.
	if userID == "" || strings.TrimSpace(userID) != userID {
		return PeriodSummary{}, ErrInvalidPayrollOwner
	}
	// Os dois domínios recebem o mesmo intervalo de dias de calendário, com
	// limites inclusivos e horários removidos.
	period, err := normalizePeriod(period)
	if err != nil {
		return PeriodSummary{}, err
	}
	// Falhar antes de qualquer leitura deixa uma configuração parcial visível
	// como erro, em vez de produzir um resumo que omita uma das fontes.
	if s == nil || s.harvest == nil || s.daily == nil {
		return PeriodSummary{}, ErrInvalidPayrollConfig
	}

	// Cada serviço filtra por gerente/período e autentica seus próprios envelopes.
	// Propagamos a falha sem devolver os totais parciais da outra fonte.
	harvests, err := s.harvest.Summarize(ctx, userID, period.harvestPeriod())
	if err != nil {
		return PeriodSummary{}, err
	}
	// A primeira consulta pode ter demorado; evitar começar a segunda quando o
	// chamador já abandonou o resumo economiza leitura e descriptografia.
	if err := ctx.Err(); err != nil {
		return PeriodSummary{}, err
	}
	dailyWorks, err := s.daily.Summarize(ctx, userID, diarias.DailyWorkPeriod{
		StartDate: period.StartDate, EndDate: period.EndDate,
	})
	if err != nil {
		return PeriodSummary{}, err
	}
	if err := ctx.Err(); err != nil {
		return PeriodSummary{}, err
	}

	// Cada domínio mantém os próprios dados e a própria cifra. Aqui apenas juntamos
	// os totais por funcionário para produzir uma visão de leitura da folha.
	byEmployee := make(map[string]EmployeeSummary, len(harvests)+len(dailyWorks))
	for _, harvest := range harvests {
		// O serviço de colheitas já verifica cada multiplicação e soma; este
		// limite protege a fronteira desta projeção caso outra fonte seja adicionada.
		if harvest.HarvestAmountCents < 0 {
			return PeriodSummary{}, ErrInvalidPayrollAmount
		}
		byEmployee[harvest.EmployeeID] = EmployeeSummary{
			EmployeeID: harvest.EmployeeID, EmployeeName: harvest.EmployeeName,
			NormalBoxes: harvest.NormalBoxes, MillingBoxes: harvest.MillingBoxes,
			NormalAmountCents:  harvest.NormalAmountCents,
			MillingAmountCents: harvest.MillingAmountCents,
			HarvestAmountCents: harvest.HarvestAmountCents,
			DueAmountCents:     harvest.HarvestAmountCents,
		}
	}
	for _, daily := range dailyWorks {
		if daily.DailyWorkAmountCents < 0 {
			return PeriodSummary{}, ErrInvalidPayrollAmount
		}
		// O ID é a chave, não o nome: funcionários homônimos continuam separados,
		// e um funcionário que só recebeu diárias também aparece no resumo.
		summary, exists := byEmployee[daily.EmployeeID]
		if !exists {
			summary = EmployeeSummary{EmployeeID: daily.EmployeeID, EmployeeName: daily.EmployeeName}
		}
		// Valor devido é o gerado pelas caixas mais as diárias; pagamentos ainda
		// não entram nesta feature e serão subtraídos por um fluxo posterior.
		due, err := addAmount(summary.DueAmountCents, daily.DailyWorkAmountCents)
		if err != nil {
			return PeriodSummary{}, err
		}
		summary.DailyWorkAmountCents = daily.DailyWorkAmountCents
		summary.DueAmountCents = due
		byEmployee[daily.EmployeeID] = summary
	}

	employees := make([]EmployeeSummary, 0, len(byEmployee))
	for _, summary := range byEmployee {
		employees = append(employees, summary)
	}
	// A ordem dos mapas varia; nome e ID mantêm a lista previsível e distinguem homônimos.
	sort.Slice(employees, func(i, j int) bool {
		if employees[i].EmployeeName != employees[j].EmployeeName {
			return employees[i].EmployeeName < employees[j].EmployeeName
		}
		return employees[i].EmployeeID < employees[j].EmployeeID
	})
	return PeriodSummary{Period: period, Employees: employees}, nil
}

func normalizePeriod(period Period) (Period, error) {
	// DATE não representa um instante. Normalizar os campos para meia-noite UTC
	// faz colheitas e diárias consultarem exatamente os mesmos dias inclusivos.
	period.StartDate = dateOnly(period.StartDate)
	period.EndDate = dateOnly(period.EndDate)
	if period.StartDate.IsZero() || period.EndDate.IsZero() || period.EndDate.Before(period.StartDate) {
		return Period{}, ErrInvalidPayrollPeriod
	}
	return period, nil
}

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func addAmount(current, addition int64) (int64, error) {
	if current < 0 || addition < 0 {
		return 0, ErrInvalidPayrollAmount
	}
	// Verificar antes da soma evita wraparound de int64 e um total artificialmente baixo.
	if current > math.MaxInt64-addition {
		return 0, ErrPayrollSummaryOverflow
	}
	return current + addition, nil
}
