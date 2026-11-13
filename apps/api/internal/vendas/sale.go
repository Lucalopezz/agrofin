// Define vendas, recebimentos e as validações de seus valores em centavos.
package vendas

import (
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
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
	RequestKey  string
}

type SalePeriod struct{ StartDate, EndDate time.Time }

type Receipt struct {
	ID          string
	UserID      string
	SaleID      string
	ReceivedAt  time.Time
	AmountCents int64
	RequestKey  string
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
	RequestKey string
}

type ProtectedSaleRecord struct {
	Sale     ProtectedSale
	Receipts []ProtectedReceipt
}

func normalizeSale(userID string, input CreateSaleInput) (Sale, error) {
	ownerID, err := uuid.Normalize(userID)
	if err != nil {
		return Sale{}, ErrInvalidSaleOwner
	}
	if input.SoldAt.IsZero() {
		return Sale{}, ErrInvalidSaleDate
	}
	if len(input.Items) == 0 {
		return Sale{}, ErrInvalidSaleItems
	}
	sale := Sale{UserID: ownerID, SoldAt: date.Normalize(input.SoldAt), BuyerName: strings.TrimSpace(input.BuyerName), Notes: strings.TrimSpace(input.Notes)}
	seen := make(map[SaleType]struct{}, len(input.Items))
	for _, in := range input.Items {
		switch in.SaleType {
		case SaleTypeExportacao, SaleTypeMercado, SaleTypeMoagem:
		default:
			return Sale{}, ErrInvalidSaleType
		}
		if _, ok := seen[in.SaleType]; ok {
			return Sale{}, ErrDuplicateSaleType
		}
		seen[in.SaleType] = struct{}{}
		if in.Quantity <= 0 {
			return Sale{}, ErrInvalidSaleQuantity
		}
		if !money.IsPositiveCents(in.UnitPriceCents) {
			return Sale{}, ErrInvalidSalePrice
		}
		q := int64(in.Quantity)
		if in.UnitPriceCents > int64(^uint64(0)>>1)/q {
			return Sale{}, ErrSaleOverflow
		}
		line := in.UnitPriceCents * q
		gross, err := addCents(sale.GrossCents, line)
		if err != nil {
			return Sale{}, err
		}
		sale.GrossCents = gross
		sale.Items = append(sale.Items, SaleItem{SaleType: in.SaleType, Quantity: in.Quantity, UnitPriceCents: in.UnitPriceCents, LineAmountCents: line})
	}
	if input.Discount != nil {
		reason := strings.TrimSpace(input.Discount.Reason)
		if reason == "" || !money.IsPositiveCents(input.Discount.AmountCents) {
			return Sale{}, ErrInvalidSaleDiscount
		}
		if input.Discount.AmountCents > sale.GrossCents {
			return Sale{}, ErrDiscountExceedsGross
		}
		sale.Discount = &SaleDiscount{Reason: reason, AmountCents: input.Discount.AmountCents}
		sale.DiscountCents = input.Discount.AmountCents
	}
	sale.NetCents = sale.GrossCents - sale.DiscountCents
	sale.ReceivableCents = sale.NetCents
	return sale, nil
}

func addCents(current, addition int64) (int64, error) {
	if current < 0 || addition < 0 || current > int64(^uint64(0)>>1)-addition {
		return 0, ErrSaleOverflow
	}
	return current + addition, nil
}
