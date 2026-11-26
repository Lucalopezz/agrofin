// Define vendas, recebimentos e a soma segura de valores em centavos.
package vendas

import (
	"math"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
)

type SaleType string

const (
	SaleTypeExportacao SaleType = "exportacao"
	SaleTypeMercado    SaleType = "mercado"
	SaleTypeMoagem     SaleType = "moagem"
)

func (t SaleType) isValid() bool {
	switch t {
	case SaleTypeExportacao, SaleTypeMercado, SaleTypeMoagem:
		return true
	default:
		return false
	}
}

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

// addItem recebe um item com quantidade positiva e preço validado na entrada
// ou autenticado pela cifra. As duas origens compartilham o cálculo do bruto.
func (s *Sale) addItem(item SaleItem) error {
	quantity := int64(item.Quantity)
	if item.UnitPriceCents > math.MaxInt64/quantity {
		return ErrSaleOverflow
	}
	item.LineAmountCents = item.UnitPriceCents * quantity
	gross, err := addCents(s.GrossCents, item.LineAmountCents)
	if err != nil {
		return err
	}
	s.GrossCents = gross
	s.Items = append(s.Items, item)
	return nil
}

func (s *Sale) applyDiscount(discount *SaleDiscount) error {
	if discount != nil {
		if discount.AmountCents > s.GrossCents {
			return ErrDiscountExceedsGross
		}
		s.Discount = discount
		s.DiscountCents = discount.AmountCents
	}
	s.NetCents = s.GrossCents - s.DiscountCents
	return nil
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
	if current < 0 || addition < 0 || current > math.MaxInt64-addition {
		return 0, ErrSaleOverflow
	}
	return current + addition, nil
}
