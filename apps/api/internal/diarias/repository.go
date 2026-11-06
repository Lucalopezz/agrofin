package diarias

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository grava lançamentos de diária usando o pool compartilhado da API.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CheckActiveEmployee(ctx context.Context, userID, employeeID string) error {
	var active bool
	err := r.pool.QueryRow(ctx, `
		SELECT active
		FROM employees
		WHERE user_id = $1 AND id = $2
	`, userID, employeeID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrEmployeeNotFound
	}
	if err != nil {
		return fmt.Errorf("consultar funcionário da diária: %w", err)
	}
	if !active {
		return ErrEmployeeInactive
	}
	return nil
}

// CreateDailyWork revalida e bloqueia o funcionário na mesma transação da
// inserção, evitando aceitar diária enquanto outra operação o desativa.
func (r *Repository) CreateDailyWork(ctx context.Context, work ProtectedDailyWork) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar gravação da diária: %w", err)
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	var active bool
	err = tx.QueryRow(ctx, `
		SELECT active
		FROM employees
		WHERE user_id = $1 AND id = $2
		FOR SHARE
	`, work.UserID, work.EmployeeID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrEmployeeNotFound
	}
	if err != nil {
		return fmt.Errorf("validar funcionário da diária: %w", err)
	}
	if !active {
		return ErrEmployeeInactive
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO daily_work (
			id, user_id, employee_id, work_date,
			amount_ciphertext, amount_nonce, amount_key_version, notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))
	`, work.ID, work.UserID, work.EmployeeID, work.WorkDate,
		work.Amount.Ciphertext, work.Amount.Nonce, work.Amount.KeyVersion, work.Notes)
	if err != nil {
		return fmt.Errorf("gravar diária protegida: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar diária: %w", err)
	}
	return nil
}
