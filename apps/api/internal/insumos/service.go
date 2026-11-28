// Prepara o cadastro e consulta os insumos da conta.
package insumos

import "context"

type productRepository interface {
	Create(ctx context.Context, product Product) (Product, error)
	List(ctx context.Context, userID string) ([]Product, error)
}

type Service struct {
	repo productRepository
}

func NewService(repo productRepository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	Name        string
	Category    Category
	DefaultUnit string
	Notes       string
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (Product, error) {
	product := Product{
		UserID:      userID,
		Name:        input.Name,
		Category:    input.Category,
		DefaultUnit: input.DefaultUnit,
		Notes:       input.Notes,
		// Sem saldo inicial conferido, o cadastro começa pendente.
		StockControlState: stockPending,
	}
	return s.repo.Create(ctx, product)
}

func (s *Service) List(ctx context.Context, userID string) ([]Product, error) {
	return s.repo.List(ctx, userID)
}
