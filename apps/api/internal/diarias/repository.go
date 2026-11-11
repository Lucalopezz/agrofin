// Persiste diárias protegidas e consulta envelopes por dono e período.
package diarias

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
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

// ListDailyWorks retorna os envelopes cifrados do período e o nome atual do
// funcionário. O filtro por dono ocorre antes de qualquer descriptografia.
func (r *Repository) ListDailyWorks(ctx context.Context, userID string, period DailyWorkPeriod) ([]ProtectedDailyWorkRecord, error) {
	const query = `
		SELECT d.id::text, d.employee_id::text, e.name, d.work_date,
		       d.amount_ciphertext, d.amount_nonce, d.amount_key_version
		FROM daily_work d
		JOIN employees e
		  ON e.user_id = d.user_id AND e.id = d.employee_id
		WHERE d.user_id = $1
		  AND d.work_date >= $2::date
		  AND d.work_date <= $3::date
		ORDER BY d.work_date DESC, d.created_at DESC, d.id DESC
	`
	rows, err := r.pool.Query(ctx, query, userID, period.StartDate, period.EndDate)
	if err != nil {
		return nil, fmt.Errorf("listar diárias protegidas: %w", err)
	}
	defer rows.Close()

	works := make([]ProtectedDailyWorkRecord, 0)
	for rows.Next() {
		var record ProtectedDailyWorkRecord
		var work ProtectedDailyWork
		err := rows.Scan(
			&work.ID, &work.EmployeeID, &record.EmployeeName, &work.WorkDate,
			&work.Amount.Ciphertext, &work.Amount.Nonce, &work.Amount.KeyVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("ler diária protegida: %w", err)
		}
		// Reutilizamos o userID confiável usado no filtro SQL ao reconstruir o AAD;
		// não dependemos de outro campo lido da linha para escolher a conta da cifra.
		work.UserID = userID
		work.WorkDate = calendarDate.Normalize(work.WorkDate)
		record.Work = work
		works = append(works, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar diárias protegidas: %w", err)
	}
	return works, nil
}
