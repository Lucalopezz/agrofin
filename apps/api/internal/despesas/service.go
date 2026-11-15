// Coordena despesas e protege valores antes da persistência.
package despesas

import (
	"context"
	"time"

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

func (s *Service) Create(ctx context.Context, owner string, in Input) (Expense, error) {
	if err := ctx.Err(); err != nil {
		return Expense{}, err
	}
	// O request entrega os campos já validados e normalizados.
	e := Expense{UserID: owner, PaidAt: in.PaidAt, Category: in.Category, Description: in.Description, AmountCents: in.AmountCents}
	var err error
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
	e := Expense{ID: id, UserID: owner, PaidAt: in.PaidAt, Category: in.Category, Description: in.Description, AmountCents: in.AmountCents}
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
	return s.repo.Delete(ctx, owner, id)
}
