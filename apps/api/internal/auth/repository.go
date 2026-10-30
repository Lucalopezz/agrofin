package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persiste os dados de identidade usando o pool compartilhado da API.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository recebe um pool já aberto; o ciclo de vida continua com a aplicação.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateManager persiste a entidade validada e preparada pelo Service.
// O índice único da instalação protege contra cadastros concorrentes.
func (r *Repository) CreateManager(ctx context.Context, manager Manager) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (login_name, password_hash)
		VALUES ($1, $2)
	`, manager.LoginName, manager.PasswordHash)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return ErrManagerAlreadyRegistered
		}
		return errors.New("não foi possível cadastrar o gerente")
	}
	return nil
}
