package diarias

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type dailyWorkRepository interface {
	CheckActiveEmployee(ctx context.Context, userID, employeeID string) error
	CreateDailyWork(ctx context.Context, work ProtectedDailyWork) error
	ListDailyWorks(ctx context.Context, userID string, period DailyWorkPeriod) ([]ProtectedDailyWorkRecord, error)
}

type dailyWorkProtector interface {
	EncryptDailyWorkAmount(userID, dailyWorkID, employeeID string, workDate time.Time, amountCents int64) (ProtectedAmount, error)
	DecryptDailyWorkAmount(userID, dailyWorkID, employeeID string, workDate time.Time, amount ProtectedAmount) (int64, error)
}

// Service valida a entrada, protege o valor e envia ao Repository uma entidade
// pronta para gravação, sem depender de HTTP ou de detalhes do PostgreSQL.
type Service struct {
	repo      dailyWorkRepository
	protector dailyWorkProtector
}

func NewService(repo dailyWorkRepository, protector dailyWorkProtector) *Service {
	return &Service{repo: repo, protector: protector}
}

func (s *Service) Create(ctx context.Context, userID string, input CreateDailyWorkInput) (DailyWork, error) {
	if err := ctx.Err(); err != nil {
		return DailyWork{}, err
	}
	if strings.TrimSpace(userID) != userID || userID == "" {
		return DailyWork{}, ErrInvalidDailyWorkOwner
	}

	work, err := normalizeDailyWork(DailyWork{
		UserID: userID, EmployeeID: input.EmployeeID,
		WorkDate: input.WorkDate, AmountCents: input.AmountCents,
		Notes: input.Notes,
	})
	if err != nil {
		return DailyWork{}, err
	}
	if s == nil || s.repo == nil || s.protector == nil {
		return DailyWork{}, ErrInvalidDailyWorkConfig
	}
	if err := s.repo.CheckActiveEmployee(ctx, work.UserID, work.EmployeeID); err != nil {
		return DailyWork{}, err
	}

	// O identificador é necessário no contexto da cifra, mas não depende do banco;
	// gerá-lo aqui remove uma consulta isolada antes da transação de criação.
	work.ID, err = uuid.New()
	if err != nil {
		return DailyWork{}, err
	}
	if err := ctx.Err(); err != nil {
		return DailyWork{}, err
	}
	amount, err := s.protector.EncryptDailyWorkAmount(
		work.UserID, work.ID, work.EmployeeID, work.WorkDate, work.AmountCents,
	)
	if err != nil {
		return DailyWork{}, err
	}
	if err := ctx.Err(); err != nil {
		return DailyWork{}, err
	}
	protected := ProtectedDailyWork{
		ID: work.ID, UserID: work.UserID, EmployeeID: work.EmployeeID,
		WorkDate: work.WorkDate, Amount: amount, Notes: work.Notes,
	}
	if err := s.repo.CreateDailyWork(ctx, protected); err != nil {
		return DailyWork{}, err
	}
	return work, nil
}

// Summarize autentica os valores das diárias antes de agregá-los por funcionário.
// O repositório restringe a leitura ao gerente e ao intervalo recebido.
func (s *Service) Summarize(ctx context.Context, userID string, period DailyWorkPeriod) ([]DailyWorkSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID == "" || strings.TrimSpace(userID) != userID {
		return nil, ErrInvalidDailyWorkOwner
	}
	period, err := normalizeDailyWorkPeriod(period)
	if err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil || s.protector == nil {
		return nil, ErrInvalidDailyWorkConfig
	}

	records, err := s.repo.ListDailyWorks(ctx, userID, period)
	if err != nil {
		return nil, err
	}
	byEmployee := make(map[string]DailyWorkSummary)
	for _, record := range records {
		// Uma consulta cancelada não pode parecer um resumo completo com apenas
		// parte dos registros processados.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		work := record.Work
		// EmployeeID e WorkDate entram no AAD. Validá-los e normalizá-los antes
		// da descriptografia garante que a leitura reconstrua o mesmo contexto usado na gravação.
		employeeID, err := normalizeDailyWorkEmployeeID(work.EmployeeID)
		if err != nil {
			return nil, err
		}
		workDate := calendarDate.Normalize(work.WorkDate)
		if workDate.IsZero() {
			return nil, ErrInvalidDailyWorkDate
		}
		// A conta autenticada, não um valor vindo da linha, compõe o AAD usado
		// para abrir o envelope; SQL já limitou a busca ao mesmo user_id.
		amount, err := s.protector.DecryptDailyWorkAmount(
			userID, work.ID, employeeID, workDate, work.Amount,
		)
		if err != nil {
			return nil, err
		}
		// Valores não positivos violam a regra do domínio e não devem entrar
		// silenciosamente no total mesmo se um envelope antigo tiver formato válido.
		if amount <= 0 {
			return nil, ErrInvalidDailyWorkAmount
		}

		// O UUID agrupa os lançamentos da mesma pessoa mesmo quando existem
		// homônimos; o nome vem do cadastro atual e serve apenas para exibição.
		summary, exists := byEmployee[employeeID]
		if !exists {
			summary = DailyWorkSummary{EmployeeID: employeeID, EmployeeName: record.EmployeeName}
		}
		// Impedir a soma que ultrapassaria int64 evita que um valor devido grande
		// transborde e apareça como um total menor.
		if summary.DailyWorkAmountCents > math.MaxInt64-amount {
			return nil, ErrDailyWorkSummaryOverflow
		}
		summary.DailyWorkAmountCents += amount
		byEmployee[employeeID] = summary
	}
	// O contexto pode ter sido cancelado após o último registro; nesse caso não
	// entregamos o resultado de uma leitura que o chamador já abandonou.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	summaries := make([]DailyWorkSummary, 0, len(byEmployee))
	for _, summary := range byEmployee {
		summaries = append(summaries, summary)
	}
	// Mapas não preservam ordem. Ordenar por nome e depois pelo ID deixa a resposta
	// estável entre consultas, mantendo funcionários com o mesmo nome distintos.
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].EmployeeName != summaries[j].EmployeeName {
			return summaries[i].EmployeeName < summaries[j].EmployeeName
		}
		return summaries[i].EmployeeID < summaries[j].EmployeeID
	})
	return summaries, nil
}
