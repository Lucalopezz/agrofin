package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrManagerAlreadyRegistered = errors.New("auth: gerente já cadastrado")

// Repository persiste os dados de identidade usando o pool compartilhado da API.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository recebe um pool já aberto; o ciclo de vida continua com a aplicação.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// RegisterManager cadastra o único gerente; a transação e o índice do banco protegem contra duas primeiras inscrições concorrentes.
func (r *Repository) RegisterManager(ctx context.Context, loginName, password string) error {
	loginName, err := normalizeLoginName(loginName)
	if err != nil {
		return err
	}
	passwordHash, err := createPasswordHash(password, loginName)
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.New("não foi possível iniciar o cadastro do gerente")
	}
	defer func() {
		// O rollback precisa de um prazo próprio se o contexto do cadastro já tiver expirado.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	var hasOwner bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users)").Scan(&hasOwner); err != nil {
		return errors.New("não foi possível consultar o gerente cadastrado")
	}
	if hasOwner {
		return ErrManagerAlreadyRegistered
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (login_name, password_hash)
		VALUES ($1, $2)
	`, loginName, passwordHash); err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return ErrManagerAlreadyRegistered
		}
		return errors.New("não foi possível cadastrar o gerente")
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.New("não foi possível finalizar o cadastro do gerente")
	}
	return nil
}
