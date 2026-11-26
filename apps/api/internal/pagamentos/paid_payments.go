// Consulta pagamentos realizados pelo instante da saída de caixa.
package pagamentos

import (
	"context"
	"fmt"
	"time"
)

type PaidPayment struct {
	Payment
	EmployeeName string
}

type ProtectedPaymentRecord struct {
	ProtectedPayment
	EmployeeName string
}

// ListPaymentsByPaidAt usa uma janela de instantes com fim exclusivo. O período
// de trabalho continua na linha para identificar a origem de cada saída.
func (r *Repository) ListPaymentsByPaidAt(ctx context.Context, userID string, start, end time.Time) ([]ProtectedPaymentRecord, error) {
	const query = `
		SELECT p.id::text, p.employee_id::text, p.period_start, p.period_end, p.paid_at,
		       p.amount_ciphertext, p.amount_nonce, p.amount_key_version,
		       COALESCE(p.notes, ''), e.name
		FROM employee_payments p
		JOIN employees e ON e.user_id = p.user_id AND e.id = p.employee_id
		WHERE p.user_id = $1 AND p.paid_at >= $2 AND p.paid_at < $3
		ORDER BY p.paid_at DESC, p.id DESC
	`
	rows, err := r.db.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar pagamentos por data da saída: %w", err)
	}
	defer rows.Close()

	result := make([]ProtectedPaymentRecord, 0)
	for rows.Next() {
		record := ProtectedPaymentRecord{ProtectedPayment: ProtectedPayment{UserID: userID}}
		p := &record.ProtectedPayment
		err := rows.Scan(
			&p.ID, &p.EmployeeID, &p.PeriodStart, &p.PeriodEnd, &p.PaidAt,
			&p.Amount.Ciphertext, &p.Amount.Nonce, &p.Amount.KeyVersion,
			&p.Notes, &record.EmployeeName,
		)
		if err != nil {
			return nil, fmt.Errorf("ler pagamento protegido: %w", err)
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *PayrollService) ListPaid(ctx context.Context, userID string, start, end time.Time) ([]PaidPayment, error) {
	records, err := s.repo.ListPaymentsByPaidAt(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]PaidPayment, 0, len(records))
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		payment, err := s.openProtectedPayment(record.ProtectedPayment, userID)
		if err != nil {
			return nil, err
		}
		result = append(result, PaidPayment{Payment: payment, EmployeeName: record.EmployeeName})
	}
	return result, ctx.Err()
}
