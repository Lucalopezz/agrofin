// Coordena cálculo do devido e projeção de valores pagos e restantes.
package pagamentos

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

type harvestSummarizer interface {
	Summarize(ctx context.Context, userID string, period colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error)
}

type dailyWorkSummarizer interface {
	Summarize(ctx context.Context, userID string, period diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error)
}

// PayrollService cria pagamentos e monta a projeção de devido, pago e restante.
type PayrollService struct {
	harvest   harvestSummarizer
	daily     dailyWorkSummarizer
	repo      paymentCreatorRepository
	protector paymentAmountProtector
}

func NewPayrollService(
	harvest harvestSummarizer,
	daily dailyWorkSummarizer,
	repo paymentCreatorRepository,
	protector paymentAmountProtector,
) *PayrollService {
	return &PayrollService{
		harvest: harvest, daily: daily, repo: repo, protector: protector,
	}
}

// Summarize combina primeiro pagamentos e depois as fontes de devido para evitar saldo acima do devido
// quando um pagamento concorre com a consulta. As fontes atuais só recebem novos lançamentos.
func (s *PayrollService) Summarize(ctx context.Context, userID string, period Period) (PeriodSummary, error) {
	if err := ctx.Err(); err != nil {
		return PeriodSummary{}, err
	}
	if userID == "" || strings.TrimSpace(userID) != userID {
		return PeriodSummary{}, ErrInvalidPayrollOwner
	}
	period, err := normalizePeriod(period)
	if err != nil {
		return PeriodSummary{}, err
	}
	if s == nil || s.harvest == nil || s.daily == nil {
		return PeriodSummary{}, ErrInvalidPayrollConfig
	}
	if s.repo == nil || s.protector == nil {
		return PeriodSummary{}, ErrInvalidPaymentConfig
	}

	paidByEmployee, err := s.summarizePayments(ctx, userID, period)
	if err != nil {
		return PeriodSummary{}, err
	}
	if err := ctx.Err(); err != nil {
		return PeriodSummary{}, err
	}
	dueSummary, err := s.summarizeDue(ctx, userID, period)
	if err != nil {
		return PeriodSummary{}, err
	}

	for index := range dueSummary.Employees {
		employee := &dueSummary.Employees[index]
		paid := paidByEmployee[employee.EmployeeID]
		if paid < 0 || paid > employee.DueAmountCents {
			// Não limitamos o saldo a zero: pagar acima do devido indica histórico inconsistente.
			return PeriodSummary{}, ErrInvalidPaymentBalance
		}
		employee.PaidAmountCents = paid
		employee.RemainingAmountCents = employee.DueAmountCents - paid
		delete(paidByEmployee, employee.EmployeeID)
	}
	if len(paidByEmployee) > 0 {
		// Todo pagamento precisa corresponder a um funcionário com devido no intervalo exato.
		return PeriodSummary{}, ErrInvalidPaymentBalance
	}
	return dueSummary, nil
}

// CheckHarvestUpdate protege todos os intervalos personalizados atingidos pela data do lançamento.
func (s *PayrollService) CheckHarvestUpdate(ctx context.Context, current, updated colheitas.Harvest) error {
	if s == nil || s.repo == nil || s.protector == nil || s.harvest == nil || s.daily == nil {
		return ErrInvalidPaymentConfig
	}
	if current.EmployeeID != updated.EmployeeID || !current.HarvestDate.Equal(updated.HarvestDate) {
		return colheitas.ErrInvalidHarvestItems
	}
	periodPayments, err := s.repo.ListPaymentsCoveringDate(ctx, current.UserID, current.EmployeeID, current.HarvestDate)
	if err != nil {
		return err
	}
	if len(periodPayments) == 0 {
		return nil
	}
	oldAmount, err := harvestValue(current)
	if err != nil {
		return err
	}
	newAmount, err := harvestValue(updated)
	if err != nil {
		return err
	}
	periods := make(map[[2]int64]Period)
	for _, protected := range periodPayments {
		payment, err := s.openProtectedPayment(protected, current.UserID)
		if err != nil {
			return err
		}
		key := [2]int64{payment.PeriodStart.Unix(), payment.PeriodEnd.Unix()}
		periods[key] = Period{StartDate: payment.PeriodStart, EndDate: payment.PeriodEnd.AddDate(0, 0, -1)}
	}
	for _, period := range periods {
		if err := ctx.Err(); err != nil {
			return err
		}
		paidByEmployee, err := s.summarizePayments(ctx, current.UserID, period)
		if err != nil {
			return err
		}
		paid := paidByEmployee[current.EmployeeID]
		dueSummary, err := s.summarizeDue(ctx, current.UserID, period)
		if err != nil {
			return err
		}
		due, err := dueAmountForEmployee(dueSummary, current.EmployeeID)
		if err != nil {
			return err
		}
		if paid > due {
			return ErrInvalidPaymentBalance
		}
		if paid == due {
			return colheitas.ErrHarvestPaidPeriod
		}
		updatedDue := due - oldAmount
		updatedDue, err = addAmount(updatedDue, newAmount)
		if err != nil {
			return err
		}
		if updatedDue < paid {
			return colheitas.ErrHarvestBelowPaidAmount
		}
	}
	return nil
}

func harvestValue(harvest colheitas.Harvest) (int64, error) {
	var total int64
	for _, item := range harvest.Items {
		if item.Quantity <= 0 || item.AppliedRateCents <= 0 {
			return 0, ErrInvalidPayrollAmount
		}
		quantity := int64(item.Quantity)
		if item.AppliedRateCents > math.MaxInt64/quantity {
			return 0, ErrPayrollSummaryOverflow
		}
		amount, err := addAmount(total, item.AppliedRateCents*quantity)
		if err != nil {
			return 0, err
		}
		total = amount
	}
	return total, nil
}

// summarizeDue calcula somente caixas e diárias para um período já normalizado.
// Create usa esse resultado antes de abrir a transação.
func (s *PayrollService) summarizeDue(ctx context.Context, userID string, period Period) (PeriodSummary, error) {
	if err := ctx.Err(); err != nil {
		return PeriodSummary{}, err
	}

	harvests, err := s.harvest.Summarize(ctx, userID, period.harvestPeriod())
	if err != nil {
		return PeriodSummary{}, err
	}
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

	byEmployee := make(map[string]EmployeeSummary, len(harvests)+len(dailyWorks))
	for _, harvest := range harvests {
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
		summary, exists := byEmployee[daily.EmployeeID]
		if !exists {
			summary = EmployeeSummary{EmployeeID: daily.EmployeeID, EmployeeName: daily.EmployeeName}
		}
		// O ID agrupa lançamentos de funcionários homônimos sem misturar seus totais.
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
	sort.Slice(employees, func(i, j int) bool {
		if employees[i].EmployeeName != employees[j].EmployeeName {
			return employees[i].EmployeeName < employees[j].EmployeeName
		}
		return employees[i].EmployeeID < employees[j].EmployeeID
	})
	return PeriodSummary{Period: period, Employees: employees}, nil
}

func normalizePeriod(period Period) (Period, error) {
	// DATE não representa um instante. Normalizar ambos os limites preserva os mesmos dias civis.
	start, end, err := calendarDate.NormalizeInclusivePeriod(period.StartDate, period.EndDate)
	if err != nil {
		return Period{}, ErrInvalidPayrollPeriod
	}
	period.StartDate, period.EndDate = start, end
	return period, nil
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
