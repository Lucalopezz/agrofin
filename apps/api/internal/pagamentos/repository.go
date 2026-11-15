// Persiste pagamentos protegidos e serializa operações por funcionário.
package pagamentos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository lê e grava pagamentos protegidos usando o pool compartilhado da API.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type paymentTransaction interface {
	ListPayments(ctx context.Context, employeeID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error)
	InsertPayment(ctx context.Context, payment ProtectedPayment) error
}

type paymentTx struct {
	tx     pgx.Tx
	userID string
}

// withEmployeePaymentLock serializa pagamentos e edição de colheitas da mesma pessoa.
func (r *Repository) withEmployeePaymentLock(
	ctx context.Context,
	userID, employeeID string,
	operation func(paymentTransaction) error,
) error {
	if r == nil || r.pool == nil {
		return ErrInvalidPaymentConfig
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar gravação do pagamento: %w", err)
	}
	defer func() {
		// A desconexão HTTP não deve deixar uma transação aberta ocupando a conexão.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	// A linha bloqueada protege a leitura do saldo e a gravação sob concorrência.
	var lockedEmployeeID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM employees WHERE user_id = $1 AND id = $2 FOR UPDATE`, userID, employeeID).Scan(&lockedEmployeeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPaymentEmployeeNotFound
	}
	if err != nil {
		return fmt.Errorf("bloquear funcionário para pagamento: %w", err)
	}

	if err := operation(paymentTx{tx: tx, userID: userID}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar pagamento: %w", err)
	}
	return nil
}

// ListPayments busca, para o resumo, os pagamentos de todos os funcionários da
// conta no intervalo exato. O filtro pelo dono acontece no banco, antes de o
// serviço receber e descriptografar os envelopes financeiros.
//
// Esta consulta usa o pool porque o resumo não participa da transação de criação
// de um pagamento. Embora tenha o mesmo nome de paymentTx.ListPayments, ela tem
// outro receiver e outra finalidade: aquele método filtra um único funcionário
// e usa a transação que mantém o bloqueio durante a conferência do saldo.
func (r *Repository) ListPayments(ctx context.Context, userID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error) {
	if r == nil || r.pool == nil {
		return nil, ErrInvalidPaymentConfig
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, employee_id::text, period_start, period_end, paid_at,
		       amount_ciphertext, amount_nonce, amount_key_version,
		       COALESCE(notes, '')
		FROM employee_payments
		WHERE user_id = $1
		  AND period_start = $2::date
		  AND period_end = $3::date
		ORDER BY employee_id, paid_at, id
	`, userID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("listar pagamentos protegidos: %w", err)
	}
	defer rows.Close()

	payments := make([]ProtectedPayment, 0)
	for rows.Next() {
		payment, err := scanProtectedPayment(rows, userID)
		if err != nil {
			return nil, fmt.Errorf("ler pagamento protegido: %w", err)
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar pagamentos protegidos: %w", err)
	}
	return payments, nil
}

// ListPaymentsCoveringDate localiza intervalos personalizados que incluem a data alterada.
func (r *Repository) ListPaymentsCoveringDate(ctx context.Context, userID, employeeID string, date time.Time) ([]ProtectedPayment, error) {
	if r == nil || r.pool == nil {
		return nil, ErrInvalidPaymentConfig
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, employee_id::text, period_start, period_end, paid_at,
		       amount_ciphertext, amount_nonce, amount_key_version,
		       COALESCE(notes, '')
		FROM employee_payments
		WHERE user_id=$1 AND employee_id=$2
		  AND period_start <= $3::date AND period_end > $3::date
		ORDER BY period_start, period_end, paid_at, id
	`, userID, employeeID, date)
	if err != nil {
		return nil, fmt.Errorf("listar pagamentos que incluem a data da colheita: %w", err)
	}
	defer rows.Close()
	payments := make([]ProtectedPayment, 0)
	for rows.Next() {
		payment, err := scanProtectedPayment(rows, userID)
		if err != nil {
			return nil, fmt.Errorf("ler pagamento protegido: %w", err)
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar períodos pagos: %w", err)
	}
	return payments, nil
}

// ListPayments busca os pagamentos de um funcionário no intervalo exato para
// calcular quanto já foi pago antes de registrar um novo pagamento. O userID vem
// da própria paymentTx, criada para a conta autenticada, e continua obrigatório
// no filtro para impedir a leitura de registros de outra conta.
//
// A consulta usa tx.tx, a mesma transação que bloqueou a linha do funcionário e
// fará a inserção. Assim, outro pagamento concorrente para a mesma pessoa espera
// a conclusão desta operação antes de conferir o saldo. Usar aqui o ListPayments
// do Repository consultaria pelo pool, sem garantir essa mesma transação; além
// disso, aquele método retorna todos os funcionários para o resumo do período.
func (tx paymentTx) ListPayments(ctx context.Context, employeeID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error) {
	rows, err := tx.tx.Query(ctx, `
		SELECT id::text, employee_id::text, period_start, period_end, paid_at,
		       amount_ciphertext, amount_nonce, amount_key_version,
		       COALESCE(notes, '')
		FROM employee_payments
		WHERE user_id = $1
		  AND employee_id = $2
		  AND period_start = $3::date
		  AND period_end = $4::date
		ORDER BY paid_at, id
	`, tx.userID, employeeID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("somar pagamentos anteriores do período: %w", err)
	}
	defer rows.Close()

	payments := make([]ProtectedPayment, 0)
	for rows.Next() {
		payment, err := scanProtectedPayment(rows, tx.userID)
		if err != nil {
			return nil, fmt.Errorf("ler pagamento anterior protegido: %w", err)
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("somar pagamentos anteriores do período: %w", err)
	}
	return payments, nil
}

// InsertPayment grava o envelope já validado dentro da transação do saldo.
func (tx paymentTx) InsertPayment(ctx context.Context, payment ProtectedPayment) error {
	_, err := tx.tx.Exec(ctx,
		`INSERT INTO employee_payments (id,user_id,employee_id,period_start,period_end,paid_at,amount_ciphertext,amount_nonce,amount_key_version,notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''))`,
		payment.ID, payment.UserID, payment.EmployeeID,
		payment.PeriodStart, payment.PeriodEnd, payment.PaidAt, payment.Amount.Ciphertext,
		payment.Amount.Nonce, payment.Amount.KeyVersion, payment.Notes)
	if err != nil {
		return fmt.Errorf("gravar pagamento protegido: %w", err)
	}
	return nil
}

type paymentRowScanner interface {
	Scan(dest ...any) error
}

// Lê uma linha retornada pelo PostgreSQL e monta um ProtectedPayment. Ela é usada nas consultas que buscam pagamentos já gravados.
// retorna o valor ainda cifrado. Quem o abre depois é o serviço, usando a cifra e os dados que identificam o pagamento.
func scanProtectedPayment(row paymentRowScanner, userID string) (ProtectedPayment, error) {
	var payment ProtectedPayment
	// Scan copia cada coluna da consulta para o campo correspondente. O & passa o endereço do campo para que Scan possa preenchê-lo.
	err := row.Scan(
		&payment.ID, &payment.EmployeeID, &payment.PeriodStart, &payment.PeriodEnd, &payment.PaidAt,
		&payment.Amount.Ciphertext, &payment.Amount.Nonce, &payment.Amount.KeyVersion,
		&payment.Notes,
	)
	if err != nil {
		return ProtectedPayment{}, err
	}
	// O mesmo userID usado no WHERE é a fonte confiável para reconstruir o contexto da cifra.
	payment.UserID = userID
	return payment, nil
}
