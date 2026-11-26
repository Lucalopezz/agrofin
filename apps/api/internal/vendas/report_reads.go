// Consulta valores vendidos e recebimentos pelas respectivas datas de origem.
package vendas

import (
	"context"
	"fmt"
	"time"
)

type ReceiptRecord struct {
	Receipt
	SoldAt time.Time
}

type ProtectedReceiptRecord struct {
	ProtectedReceipt
	SoldAt time.Time
}

// ListSold retorna o valor negociado. A leitura usa itens e descontos sem
// carregar os acertos de cada venda, que seguem seu próprio recorte temporal.
func (s *Service) ListSold(ctx context.Context, userID string, period SalePeriod) ([]Sale, error) {
	records, err := s.repo.ListSoldSales(ctx, userID, period.StartDate, period.EndDate)
	if err != nil {
		return nil, err
	}
	result := make([]Sale, 0, len(records))
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sale, err := s.openSoldSale(userID, record.Sale)
		if err != nil {
			return nil, err
		}
		result = append(result, sale)
	}
	return result, ctx.Err()
}

func (r *Repository) ListReceiptsByReceivedAt(ctx context.Context, userID string, start, end time.Time) ([]ProtectedReceiptRecord, error) {
	const query = `
		SELECT r.id::text, r.sale_id::text, r.received_at, s.sold_at,
		       r.amount_ciphertext, r.amount_nonce, r.amount_key_version
		FROM sale_receipts r
		JOIN sales s ON s.user_id = r.user_id AND s.id = r.sale_id
		WHERE r.user_id = $1 AND r.received_at >= $2 AND r.received_at < $3
		ORDER BY r.received_at DESC, r.id DESC
	`
	rows, err := r.db.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar entradas por data do recebimento: %w", err)
	}
	defer rows.Close()

	result := make([]ProtectedReceiptRecord, 0)
	for rows.Next() {
		record := ProtectedReceiptRecord{ProtectedReceipt: ProtectedReceipt{UserID: userID}}
		p := &record.ProtectedReceipt
		err := rows.Scan(
			&p.ID, &p.SaleID, &p.ReceivedAt, &record.SoldAt,
			&p.Amount.Ciphertext, &p.Amount.Nonce, &p.Amount.KeyVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("ler recebimento protegido: %w", err)
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *Service) ListReceived(ctx context.Context, userID string, start, end time.Time) ([]ReceiptRecord, error) {
	records, err := s.repo.ListReceiptsByReceivedAt(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]ReceiptRecord, 0, len(records))
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		receipt, err := s.openReceipt(record.ProtectedReceipt)
		if err != nil {
			return nil, err
		}
		if receipt.AmountCents <= 0 {
			return nil, ErrInvalidProtectedSaleAmount
		}
		result = append(result, ReceiptRecord{Receipt: receipt, SoldAt: record.SoldAt})
	}
	return result, ctx.Err()
}
