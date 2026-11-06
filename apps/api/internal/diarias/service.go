package diarias

import (
	"context"
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type dailyWorkRepository interface {
	CheckActiveEmployee(ctx context.Context, userID, employeeID string) error
	CreateDailyWork(ctx context.Context, work ProtectedDailyWork) error
}

type dailyWorkProtector interface {
	EncryptDailyWorkAmount(userID, dailyWorkID, employeeID string, workDate time.Time, amountCents int64) (ProtectedAmount, error)
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
