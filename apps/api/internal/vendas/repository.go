// Persiste vendas e recebimentos protegidos no PostgreSQL.
package vendas

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) CreateSale(ctx context.Context, sale ProtectedSale) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar gravação da venda: %w", err)
	}
	defer rollbackSale(ctx, tx)
	_, err = tx.Exec(ctx, `INSERT INTO sales(id,user_id,sold_at,buyer_name,notes) VALUES($1,$2,$3,NULLIF($4,''),NULLIF($5,''))`, sale.ID, sale.UserID, sale.SoldAt, sale.BuyerName, sale.Notes)
	if err != nil {
		return fmt.Errorf("gravar venda: %w", err)
	}
	for _, item := range sale.Items {
		_, err = tx.Exec(ctx, `INSERT INTO sale_items(id,user_id,sale_id,sale_type,quantity,unit_price_ciphertext,unit_price_nonce,unit_price_key_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, item.ID, sale.UserID, sale.ID, string(item.SaleType), item.Quantity, item.UnitPrice.Ciphertext, item.UnitPrice.Nonce, item.UnitPrice.KeyVersion)
		if err != nil {
			return fmt.Errorf("gravar item protegido da venda: %w", err)
		}
	}
	if sale.Discount != nil {
		_, err = tx.Exec(ctx, `INSERT INTO sale_discounts(id,user_id,sale_id,reason,amount_ciphertext,amount_nonce,amount_key_version) VALUES($1,$2,$3,$4,$5,$6,$7)`, sale.Discount.ID, sale.UserID, sale.ID, sale.Discount.Reason, sale.Discount.Amount.Ciphertext, sale.Discount.Amount.Nonce, sale.Discount.Amount.KeyVersion)
		if err != nil {
			return fmt.Errorf("gravar desconto protegido: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar venda: %w", err)
	}
	return nil
}

func (r *Repository) ListSales(ctx context.Context, userID string, start, end time.Time) ([]ProtectedSaleRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text,sold_at,COALESCE(buyer_name,''),COALESCE(notes,'') FROM sales WHERE user_id=$1 AND sold_at >= $2::date AND sold_at <= $3::date ORDER BY sold_at DESC,created_at DESC,id DESC`, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar vendas: %w", err)
	}
	type header struct {
		id           string
		sold         time.Time
		buyer, notes string
	}
	headers := make([]header, 0)
	for rows.Next() {
		var h header
		if err := rows.Scan(&h.id, &h.sold, &h.buyer, &h.notes); err != nil {
			rows.Close()
			return nil, fmt.Errorf("ler venda: %w", err)
		}
		headers = append(headers, h)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return nil, fmt.Errorf("listar vendas: %w", rowsErr)
	}
	result := make([]ProtectedSaleRecord, 0, len(headers))
	for _, h := range headers {
		record, err := r.getSale(ctx, userID, h.id)
		if err != nil {
			return nil, err
		}
		record.Sale.SoldAt = calendarDate.Normalize(h.sold)
		record.Sale.BuyerName = h.buyer
		record.Sale.Notes = h.notes
		result = append(result, record)
	}
	return result, nil
}

func (r *Repository) GetSale(ctx context.Context, userID, saleID string) (ProtectedSaleRecord, error) {
	return r.getSale(ctx, userID, saleID)
}

func (r *Repository) getSale(ctx context.Context, userID, saleID string) (ProtectedSaleRecord, error) {
	var record ProtectedSaleRecord
	var sold time.Time
	err := r.pool.QueryRow(ctx, `SELECT sold_at,COALESCE(buyer_name,''),COALESCE(notes,'') FROM sales WHERE user_id=$1 AND id=$2`, userID, saleID).Scan(&sold, &record.Sale.BuyerName, &record.Sale.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProtectedSaleRecord{}, ErrSaleNotFound
	}
	if err != nil {
		return ProtectedSaleRecord{}, fmt.Errorf("consultar venda: %w", err)
	}
	record.Sale.ID = saleID
	record.Sale.UserID = userID
	record.Sale.SoldAt = calendarDate.Normalize(sold)
	rows, err := r.pool.Query(ctx, `SELECT id::text,sale_type,quantity,unit_price_ciphertext,unit_price_nonce,unit_price_key_version FROM sale_items WHERE user_id=$1 AND sale_id=$2 ORDER BY sale_type,id`, userID, saleID)
	if err != nil {
		return ProtectedSaleRecord{}, fmt.Errorf("listar itens protegidos: %w", err)
	}
	for rows.Next() {
		var item ProtectedSaleItem
		var typ string
		if err := rows.Scan(&item.ID, &typ, &item.Quantity, &item.UnitPrice.Ciphertext, &item.UnitPrice.Nonce, &item.UnitPrice.KeyVersion); err != nil {
			rows.Close()
			return ProtectedSaleRecord{}, fmt.Errorf("ler item protegido: %w", err)
		}
		item.SaleType = SaleType(typ)
		record.Sale.Items = append(record.Sale.Items, item)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return ProtectedSaleRecord{}, fmt.Errorf("listar itens protegidos: %w", rowsErr)
	}
	var discount ProtectedDiscount
	err = r.pool.QueryRow(ctx, `SELECT id::text,reason,amount_ciphertext,amount_nonce,amount_key_version FROM sale_discounts WHERE user_id=$1 AND sale_id=$2 ORDER BY id LIMIT 1`, userID, saleID).Scan(&discount.ID, &discount.Reason, &discount.Amount.Ciphertext, &discount.Amount.Nonce, &discount.Amount.KeyVersion)
	if err == nil {
		record.Sale.Discount = &discount
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ProtectedSaleRecord{}, fmt.Errorf("consultar desconto protegido: %w", err)
	}
	receipts, err := r.listReceipts(ctx, userID, saleID)
	if err != nil {
		return ProtectedSaleRecord{}, err
	}
	record.Receipts = receipts
	return record, nil
}

func (r *Repository) listReceipts(ctx context.Context, userID, saleID string) ([]ProtectedReceipt, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text,received_at,amount_ciphertext,amount_nonce,amount_key_version,request_key::text FROM sale_receipts WHERE user_id=$1 AND sale_id=$2 ORDER BY received_at,id`, userID, saleID)
	if err != nil {
		return nil, fmt.Errorf("listar recebimentos protegidos: %w", err)
	}
	defer rows.Close()
	out := make([]ProtectedReceipt, 0)
	for rows.Next() {
		var p ProtectedReceipt
		if err := rows.Scan(&p.ID, &p.ReceivedAt, &p.Amount.Ciphertext, &p.Amount.Nonce, &p.Amount.KeyVersion, &p.RequestKey); err != nil {
			return nil, fmt.Errorf("ler recebimento protegido: %w", err)
		}
		p.UserID = userID
		p.SaleID = saleID
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar recebimentos protegidos: %w", err)
	}
	return out, nil
}

func (r *Repository) FindReceiptByRequestKey(ctx context.Context, userID, requestKey string) (ProtectedReceipt, bool, error) {
	var p ProtectedReceipt
	err := r.pool.QueryRow(ctx, `SELECT id::text,sale_id::text,received_at,amount_ciphertext,amount_nonce,amount_key_version,request_key::text FROM sale_receipts WHERE user_id=$1 AND request_key=$2`, userID, requestKey).Scan(&p.ID, &p.SaleID, &p.ReceivedAt, &p.Amount.Ciphertext, &p.Amount.Nonce, &p.Amount.KeyVersion, &p.RequestKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProtectedReceipt{}, false, nil
	}
	if err != nil {
		return ProtectedReceipt{}, false, fmt.Errorf("consultar chave do recebimento: %w", err)
	}
	p.UserID = userID
	return p, true, nil
}

type receiptTx interface {
	List(ctx context.Context, userID, saleID string) ([]ProtectedReceipt, error)
	Insert(ctx context.Context, receipt ProtectedReceipt) (ProtectedReceipt, bool, error)
}

func (r *Repository) withSaleReceiptLock(ctx context.Context, userID, saleID, requestKey string, fn func(receiptTx, *ProtectedReceipt) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar recebimento: %w", err)
	}
	defer rollbackSale(ctx, tx)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, userID+":"+requestKey)
	if err != nil {
		return fmt.Errorf("serializar chave do recebimento: %w", err)
	}
	existing, found, err := findReceipt(ctx, tx, userID, requestKey)
	if err != nil {
		return err
	}
	var existingPtr *ProtectedReceipt
	if found {
		existingPtr = &existing
	} else {
		var locked string
		err = tx.QueryRow(ctx, `SELECT id::text FROM sales WHERE user_id=$1 AND id=$2 FOR UPDATE`, userID, saleID).Scan(&locked)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSaleNotFound
		}
		if err != nil {
			return fmt.Errorf("bloquear venda para recebimento: %w", err)
		}
	}
	if err := fn(saleReceiptTx{tx: tx}, existingPtr); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar recebimento: %w", err)
	}
	return nil
}

type saleReceiptTx struct{ tx pgx.Tx }

func (t saleReceiptTx) List(ctx context.Context, userID, saleID string) ([]ProtectedReceipt, error) {
	return listReceiptsTx(ctx, t.tx, userID, saleID)
}
func (t saleReceiptTx) Insert(ctx context.Context, p ProtectedReceipt) (ProtectedReceipt, bool, error) {
	var id string
	err := t.tx.QueryRow(ctx, `INSERT INTO sale_receipts(id,user_id,sale_id,received_at,amount_ciphertext,amount_nonce,amount_key_version,request_key) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(user_id,request_key) DO NOTHING RETURNING id::text`, p.ID, p.UserID, p.SaleID, p.ReceivedAt, p.Amount.Ciphertext, p.Amount.Nonce, p.Amount.KeyVersion, p.RequestKey).Scan(&id)
	if err == nil {
		return p, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ProtectedReceipt{}, false, fmt.Errorf("gravar recebimento protegido: %w", err)
	}
	existing, found, err := findReceipt(ctx, t.tx, p.UserID, p.RequestKey)
	if err != nil {
		return ProtectedReceipt{}, false, err
	}
	if !found {
		return ProtectedReceipt{}, false, fmt.Errorf("chave do recebimento conflitou sem registro")
	}
	return existing, false, nil
}

func listReceiptsTx(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, userID, saleID string) ([]ProtectedReceipt, error) {
	rows, err := q.Query(ctx, `SELECT id::text,received_at,amount_ciphertext,amount_nonce,amount_key_version,request_key::text FROM sale_receipts WHERE user_id=$1 AND sale_id=$2 ORDER BY received_at,id`, userID, saleID)
	if err != nil {
		return nil, fmt.Errorf("listar recebimentos na transação: %w", err)
	}
	defer rows.Close()
	out := make([]ProtectedReceipt, 0)
	for rows.Next() {
		var p ProtectedReceipt
		if err := rows.Scan(&p.ID, &p.ReceivedAt, &p.Amount.Ciphertext, &p.Amount.Nonce, &p.Amount.KeyVersion, &p.RequestKey); err != nil {
			return nil, err
		}
		p.UserID = userID
		p.SaleID = saleID
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
func findReceipt(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID, requestKey string) (ProtectedReceipt, bool, error) {
	var p ProtectedReceipt
	err := q.QueryRow(ctx, `SELECT id::text,sale_id::text,received_at,amount_ciphertext,amount_nonce,amount_key_version,request_key::text FROM sale_receipts WHERE user_id=$1 AND request_key=$2`, userID, requestKey).Scan(&p.ID, &p.SaleID, &p.ReceivedAt, &p.Amount.Ciphertext, &p.Amount.Nonce, &p.Amount.KeyVersion, &p.RequestKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProtectedReceipt{}, false, nil
	}
	if err != nil {
		return ProtectedReceipt{}, false, fmt.Errorf("consultar recebimento: %w", err)
	}
	p.UserID = userID
	return p, true, nil
}
func rollbackSale(ctx context.Context, tx pgx.Tx) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = tx.Rollback(cleanup)
}
