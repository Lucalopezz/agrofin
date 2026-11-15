// Valida despesas e protege valores antes da persistência.
package despesas

import (
	"context"
	"strings"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type repository interface {
	Create(context.Context, protectedExpense) error
	List(context.Context, string, time.Time, time.Time) ([]protectedExpense, error)
	Update(context.Context, protectedExpense) (protectedExpense, error)
	Delete(context.Context, string, string) error
}

// expenseProtector mantém a cifra independente da coordenação dos casos de uso.
type expenseProtector interface {
	protect(Expense) (protectedExpense, error)
	open(protectedExpense) (Expense, error)
}

type Service struct {
	repo      repository
	protector expenseProtector
}

func NewService(repo repository, protector expenseProtector) *Service {
	return &Service{repo: repo, protector: protector}
}

func normalize(owner string, in Input) (Expense, error) {
	id, err := uuid.Normalize(owner)
	if err != nil {
		return Expense{}, ErrInvalid
	}
	date := calendarDate.Normalize(in.PaidAt)
	category := strings.TrimSpace(in.Category)
	description := strings.TrimSpace(in.Description)
	if date.IsZero() || category == "" || in.AmountCents <= 0 {
		return Expense{}, ErrInvalid
	}
	return Expense{UserID: id, PaidAt: date, Category: category, Description: description, AmountCents: in.AmountCents}, nil
}

func (s *Service) Create(ctx context.Context, owner string, in Input) (Expense, error) {
	if err := ctx.Err(); err != nil {
		return Expense{}, err
	}
	e, err := normalize(owner, in)
	if err != nil {
		return Expense{}, err
	}
	// O ID já existe antes da cifra para vincular o envelope à despesa.
	e.ID, err = uuid.New()
	if err != nil {
		return Expense{}, err
	}
	p, err := s.protector.protect(e)
	if err != nil {
		return Expense{}, err
	}
	if err = s.repo.Create(ctx, p); err != nil {
		return Expense{}, err
	}
	return e, nil
}

func (s *Service) List(ctx context.Context, owner string, start, end time.Time) ([]Expense, error) {
	owner, err := uuid.Normalize(owner)
	if err != nil {
		return nil, ErrInvalid
	}
	start, end, err = calendarDate.NormalizeInclusivePeriod(start, end)
	if err != nil {
		return nil, ErrInvalid
	}
	rows, err := s.repo.List(ctx, owner, start, end)
	if err != nil {
		return nil, err
	}
	out := make([]Expense, 0, len(rows))
	for _, p := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		e, err := s.protector.open(p)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, owner, id string, in Input) (Expense, error) {
	e, err := normalize(owner, in)
	if err != nil {
		return Expense{}, err
	}
	e.ID, err = uuid.Normalize(id)
	if err != nil {
		return Expense{}, ErrInvalid
	}
	p, err := s.protector.protect(e)
	if err != nil {
		return Expense{}, err
	}
	p, err = s.repo.Update(ctx, p)
	if err != nil {
		return Expense{}, err
	}
	return s.protector.open(p)
}

func (s *Service) Delete(ctx context.Context, owner, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	owner, err := uuid.Normalize(owner)
	if err != nil {
		return ErrInvalid
	}
	id, err = uuid.Normalize(id)
	if err != nil {
		return ErrInvalid
	}
	return s.repo.Delete(ctx, owner, id)
}
