// Define vendas, recebimentos e a soma segura de valores em centavos.
package vendas

import (
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
)

type SaleType string

const (
	SaleTypeExportacao SaleType = "exportacao"
	SaleTypeMercado    SaleType = "mercado"
	SaleTypeMoagem     SaleType = "moagem"
)

type Sale struct {
	ID              string
	UserID          string
	SoldAt          time.Time
	BuyerName       string
	Notes           string
	Items           []SaleItem
	Discount        *SaleDiscount
	GrossCents      int64
	DiscountCents   int64
	NetCents        int64
	ReceivedCents   int64
	ReceivableCents int64
}

type SaleItem struct {
	ID              string
	SaleType        SaleType
	Quantity        int32
	UnitPriceCents  int64
	LineAmountCents int64
}

type SaleDiscount struct {
	ID          string
	Reason      string
	AmountCents int64
}

type CreateSaleInput struct {
	SoldAt    time.Time
	BuyerName string
	Notes     string
	Items     []SaleItemInput
	Discount  *SaleDiscountInput
}

type SaleItemInput struct {
	SaleType       SaleType
	Quantity       int32
	UnitPriceCents int64
}

type SaleDiscountInput struct {
	Reason      string
	AmountCents int64
}

type CreateReceiptInput struct {
	SaleID      string
	AmountCents int64
}

type SalePeriod struct{ StartDate, EndDate time.Time }

type Receipt struct {
	ID          string
	UserID      string
	SaleID      string
	ReceivedAt  time.Time
	AmountCents int64
}

type ProtectedAmount = financialcipher.Amount[salePurpose]

type ProtectedSaleItem struct {
	ID        string
	SaleType  SaleType
	Quantity  int32
	UnitPrice ProtectedAmount
}

type ProtectedDiscount struct {
	ID     string
	Reason string
	Amount ProtectedAmount
}

type ProtectedSale struct {
	ID        string
	UserID    string
	SoldAt    time.Time
	BuyerName string
	Notes     string
	Items     []ProtectedSaleItem
	Discount  *ProtectedDiscount
}

type ProtectedReceipt struct {
	ID         string
	UserID     string
	SaleID     string
	ReceivedAt time.Time
	Amount     ProtectedAmount
}

type ProtectedSaleRecord struct {
	Sale     ProtectedSale
	Receipts []ProtectedReceipt
}

func addCents(current, addition int64) (int64, error) {
	if current < 0 || addition < 0 || current > int64(^uint64(0)>>1)-addition {
		return 0, ErrSaleOverflow
	}
	return current + addition, nil
}
