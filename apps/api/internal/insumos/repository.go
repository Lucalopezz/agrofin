// Persiste e consulta insumos no PostgreSQL sob a conta proprietária.
package insumos

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, product Product) (Product, error) {
	const query = `
		INSERT INTO input_products (user_id, name, category, default_unit, notes, stock_control_state)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
		RETURNING id::text
	`
	// O banco gera o ID e o devolve na própria inserção, sem consulta adicional.
	err := r.pool.QueryRow(ctx, query,
		product.UserID, product.Name, product.Category, product.DefaultUnit,
		product.Notes, product.StockControlState,
	).Scan(&product.ID)
	if err != nil {
		return Product{}, fmt.Errorf("criar insumo: %w", err)
	}
	return product, nil
}

func (r *Repository) List(ctx context.Context, userID string) ([]Product, error) {
	// O ID desempata nomes iguais. COALESCE também cobre cadastros anteriores à coluna notes.
	const query = `
		SELECT id::text, name, category, default_unit, COALESCE(notes, ''), stock_control_state
		FROM input_products
		WHERE user_id = $1
		ORDER BY name, id
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("consultar insumos: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		product := Product{UserID: userID}
		if err := rows.Scan(
			&product.ID, &product.Name, &product.Category, &product.DefaultUnit,
			&product.Notes, &product.StockControlState,
		); err != nil {
			return nil, fmt.Errorf("ler insumo: %w", err)
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar insumos: %w", err)
	}
	return products, nil
}
