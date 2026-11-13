// Persiste pagamentos protegidos e serializa operações por funcionário e chave.
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

// FindPaymentByRequestKey permite responder uma repetição antes de recalcular o devido do período.
func (r *Repository) FindPaymentByRequestKey(ctx context.Context, userID, requestKey string) (ProtectedPayment, bool, error) {
	if r == nil || r.pool == nil {
		return ProtectedPayment{}, false, ErrInvalidPaymentConfig
	}
	return findPaymentByRequestKey(ctx, r.pool, userID, requestKey)
}

type paymentTransaction interface {
	ListPayments(ctx context.Context, employeeID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error)
	InsertPayment(ctx context.Context, payment ProtectedPayment) (ProtectedPayment, bool, error)
}

type paymentTx struct {
	tx     pgx.Tx
	userID string
}

// withEmployeePaymentLock serializa pagamentos da mesma pessoa e protege a chave idempotente.
// A trava consultiva por chave resolve repetições concorrentes mesmo quando tentam funcionários distintos;
// a trava exclusiva do funcionário mantém soma dos pagamentos e inserção na mesma seção serializada.
func (r *Repository) withEmployeePaymentLock(
	ctx context.Context,
	userID, employeeID, requestKey string,
	operation func(paymentTransaction, *ProtectedPayment) error,
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

	// UUIDs normalizados não contêm ':', então essa composição identifica o par dono/chave sem ambiguidade.
	// Uma colisão do hash apenas serializa duas requisições independentes; a unicidade real continua no banco.
	lockKey := userID + ":" + requestKey
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return fmt.Errorf("serializar chave de idempotência do pagamento: %w", err)
	}

	existing, found, err := findPaymentByRequestKey(ctx, tx, userID, requestKey)
	if err != nil {
		return err
	}
	var existingPayment *ProtectedPayment
	if found {
		existingPayment = &existing
	}
	if !found {
		// Colheitas e diárias usam FOR SHARE para manter ativo/dono estáveis durante a gravação.
		// FOR UPDATE faz novas gravações dessas origens aguardarem enquanto o saldo é conferido.
		var lockedEmployeeID string
		err := tx.QueryRow(ctx, `
			SELECT id::text
			FROM employees
			WHERE user_id = $1 AND id = $2
			FOR UPDATE
		`, userID, employeeID).Scan(&lockedEmployeeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPaymentEmployeeNotFound
		}
		if err != nil {
			return fmt.Errorf("bloquear funcionário para pagamento: %w", err)
		}
	}

	if err := operation(paymentTx{tx: tx, userID: userID}, existingPayment); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar pagamento: %w", err)
	}
	return nil
}

// ListPayments filtra dono e intervalo antes de devolver envelopes cifrados ao serviço.
func (r *Repository) ListPayments(ctx context.Context, userID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error) {
	if r == nil || r.pool == nil {
		return nil, ErrInvalidPaymentConfig
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, employee_id::text, period_start, period_end, paid_at,
		       amount_ciphertext, amount_nonce, amount_key_version,
		       COALESCE(notes, ''), request_key::text
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
		       COALESCE(notes, ''), request_key::text
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

func (tx paymentTx) ListPayments(ctx context.Context, employeeID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error) {
	rows, err := tx.tx.Query(ctx, `
		SELECT id::text, employee_id::text, period_start, period_end, paid_at,
		       amount_ciphertext, amount_nonce, amount_key_version,
		       COALESCE(notes, ''), request_key::text
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

// InsertPayment usa a restrição única como última defesa contra escritores que não tomem a trava consultiva.
func (tx paymentTx) InsertPayment(ctx context.Context, payment ProtectedPayment) (ProtectedPayment, bool, error) {
	var insertedID string
	err := tx.tx.QueryRow(ctx, `
		INSERT INTO employee_payments (
			id, user_id, employee_id, period_start, period_end, paid_at,
			amount_ciphertext, amount_nonce, amount_key_version, notes, request_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''), $11)
		ON CONFLICT (user_id, request_key) DO NOTHING
		RETURNING id::text
	`, payment.ID, payment.UserID, payment.EmployeeID, payment.PeriodStart, payment.PeriodEnd,
		payment.PaidAt, payment.Amount.Ciphertext, payment.Amount.Nonce,
		payment.Amount.KeyVersion, payment.Notes, payment.RequestKey,
	).Scan(&insertedID)
	if err == nil {
		return payment, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ProtectedPayment{}, false, fmt.Errorf("gravar pagamento protegido: %w", err)
	}

	// ON CONFLICT não aborta a transação; o serviço abre o registro existente e compara o pedido original.
	existing, found, err := findPaymentByRequestKey(ctx, tx.tx, payment.UserID, payment.RequestKey)
	if err != nil {
		return ProtectedPayment{}, false, err
	}
	if !found {
		return ProtectedPayment{}, false, fmt.Errorf("chave única do pagamento conflitou sem registro correspondente")
	}
	return existing, false, nil
}

type paymentRowScanner interface {
	Scan(dest ...any) error
}

func findPaymentByRequestKey(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID, requestKey string,
) (ProtectedPayment, bool, error) {
	payment, err := scanProtectedPayment(queryer.QueryRow(ctx, `
		SELECT id::text, employee_id::text, period_start, period_end, paid_at,
		       amount_ciphertext, amount_nonce, amount_key_version,
		       COALESCE(notes, ''), request_key::text
		FROM employee_payments
		WHERE user_id = $1 AND request_key = $2
	`, userID, requestKey), userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProtectedPayment{}, false, nil
	}
	if err != nil {
		return ProtectedPayment{}, false, fmt.Errorf("consultar chave de idempotência do pagamento: %w", err)
	}
	return payment, true, nil
}

func scanProtectedPayment(row paymentRowScanner, userID string) (ProtectedPayment, error) {
	var payment ProtectedPayment
	err := row.Scan(
		&payment.ID, &payment.EmployeeID, &payment.PeriodStart, &payment.PeriodEnd, &payment.PaidAt,
		&payment.Amount.Ciphertext, &payment.Amount.Nonce, &payment.Amount.KeyVersion,
		&payment.Notes, &payment.RequestKey,
	)
	if err != nil {
		return ProtectedPayment{}, err
	}
	// O mesmo userID usado no WHERE é a fonte confiável para reconstruir o contexto da cifra.
	payment.UserID = userID
	return payment, nil
}
