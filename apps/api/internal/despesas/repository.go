// Persiste e consulta despesas protegidas no PostgreSQL.
package despesas

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type rowScanner interface{ Scan(...any) error }

func scanExpense(row rowScanner, owner string) (protectedExpense, error) {
	var p protectedExpense
	p.Expense.UserID = owner
	err := row.Scan(&p.Expense.ID, &p.Expense.PaidAt, &p.Expense.Category, &p.Expense.Description, &p.Amount.Ciphertext, &p.Amount.Nonce, &p.Amount.KeyVersion)
	return p, err
}

func (r *Repository) Create(ctx context.Context, p protectedExpense) error {
	e := p.Expense
	_, err := r.pool.Exec(ctx, `INSERT INTO expenses(id,user_id,paid_at,category,description,amount_ciphertext,amount_nonce,amount_key_version) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8)`, e.ID, e.UserID, e.PaidAt, e.Category, e.Description, p.Amount.Ciphertext, p.Amount.Nonce, p.Amount.KeyVersion)
	return err
}

func (r *Repository) List(ctx context.Context, owner string, start, end time.Time) ([]protectedExpense, error) {
	// Limites em UTC representam datas civis, sem depender do fuso configurado no PostgreSQL.
	rows, err := r.pool.Query(ctx, `SELECT id::text,paid_at,category,COALESCE(description,''),amount_ciphertext,amount_nonce,amount_key_version FROM expenses WHERE user_id=$1 AND paid_at >= $2 AND paid_at < $3 ORDER BY paid_at DESC,created_at DESC,id DESC`, owner, start, end.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]protectedExpense, 0)
	for rows.Next() {
		p, err := scanExpense(rows, owner)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, p protectedExpense) (protectedExpense, error) {
	e := p.Expense
	row := r.pool.QueryRow(ctx, `UPDATE expenses SET paid_at=$3,category=$4,description=NULLIF($5,''),amount_ciphertext=$6,amount_nonce=$7,amount_key_version=$8 WHERE user_id=$1 AND id=$2 RETURNING id::text,paid_at,category,COALESCE(description,''),amount_ciphertext,amount_nonce,amount_key_version`, e.UserID, e.ID, e.PaidAt, e.Category, e.Description, p.Amount.Ciphertext, p.Amount.Nonce, p.Amount.KeyVersion)
	saved, err := scanExpense(row, e.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return protectedExpense{}, ErrNotFound
	}
	return saved, err
}

func (r *Repository) Delete(ctx context.Context, owner, id string) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM expenses WHERE user_id=$1 AND id=$2`, owner, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
