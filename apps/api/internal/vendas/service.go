// Valida vendas, protege valores e calcula valores vendidos e recebidos.
package vendas

import (
	"context"
	"math"
	"strings"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type saleRepository interface {
	CreateSale(context.Context, ProtectedSale) error
	GetSale(context.Context, string, string) (ProtectedSaleRecord, error)
	ListSales(context.Context, string, time.Time, time.Time) ([]ProtectedSaleRecord, error)
	FindReceiptByRequestKey(context.Context, string, string) (ProtectedReceipt, bool, error)
	withSaleReceiptLock(context.Context, string, string, string, func(receiptTx, *ProtectedReceipt) error) error
}

type saleProtector interface {
	encrypt(string, string, string, string, []string, int64) (ProtectedAmount, error)
	decrypt(string, string, string, string, []string, ProtectedAmount) (int64, error)
}

type Service struct {
	repo      saleRepository
	protector saleProtector
}

func NewService(repo saleRepository, protector saleProtector) *Service {
	return &Service{repo: repo, protector: protector}
}

func (s *Service) Create(ctx context.Context, userID string, input CreateSaleInput) (Sale, error) {
	if err := ctx.Err(); err != nil {
		return Sale{}, err
	}
	sale, err := normalizeSale(userID, input)
	if err != nil {
		return Sale{}, err
	}
	if s == nil || s.repo == nil || s.protector == nil {
		return Sale{}, ErrInvalidSaleConfig
	}
	sale.ID, err = uuid.New()
	if err != nil {
		return Sale{}, err
	}
	protected := ProtectedSale{ID: sale.ID, UserID: sale.UserID, SoldAt: sale.SoldAt, BuyerName: sale.BuyerName, Notes: sale.Notes, Items: make([]ProtectedSaleItem, 0, len(sale.Items))}
	for i := range sale.Items {
		item := &sale.Items[i]
		item.ID, err = uuid.New()
		if err != nil {
			return Sale{}, err
		}
		amount, err := s.protector.encrypt("agrofin.sale.unit-price.v1", sale.UserID, sale.ID, item.ID, []string{string(item.SaleType), saleDateAttribute(sale.SoldAt)}, item.UnitPriceCents)
		if err != nil {
			return Sale{}, err
		}
		protected.Items = append(protected.Items, ProtectedSaleItem{ID: item.ID, SaleType: item.SaleType, Quantity: item.Quantity, UnitPrice: amount})
	}
	if sale.Discount != nil {
		sale.Discount.ID, err = uuid.New()
		if err != nil {
			return Sale{}, err
		}
		amount, err := s.protector.encrypt("agrofin.sale.discount.v1", sale.UserID, sale.ID, sale.Discount.ID, []string{saleDateAttribute(sale.SoldAt), sale.Discount.Reason}, sale.Discount.AmountCents)
		if err != nil {
			return Sale{}, err
		}
		protected.Discount = &ProtectedDiscount{ID: sale.Discount.ID, Reason: sale.Discount.Reason, Amount: amount}
	}
	if err := ctx.Err(); err != nil {
		return Sale{}, err
	}
	if err := s.repo.CreateSale(ctx, protected); err != nil {
		return Sale{}, err
	}
	return sale, nil
}

func (s *Service) List(ctx context.Context, userID string, period SalePeriod) ([]Sale, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userID) != userID || userID == "" {
		return nil, ErrInvalidSaleOwner
	}
	start, end, err := calendarDate.NormalizeInclusivePeriod(period.StartDate, period.EndDate)
	if err != nil {
		return nil, ErrInvalidSaleDate
	}
	if s == nil || s.repo == nil || s.protector == nil {
		return nil, ErrInvalidSaleConfig
	}
	records, err := s.repo.ListSales(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]Sale, 0, len(records))
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sale, err := s.openSale(userID, record)
		if err != nil {
			return nil, err
		}
		result = append(result, sale)
	}
	return result, nil
}

func (s *Service) Receive(ctx context.Context, userID string, input CreateReceiptInput) (Receipt, bool, error) {
	if err := ctx.Err(); err != nil {
		return Receipt{}, false, err
	}
	ownerID, err := uuid.Normalize(userID)
	if err != nil {
		return Receipt{}, false, ErrInvalidSaleOwner
	}
	saleID, err := uuid.Normalize(input.SaleID)
	if err != nil {
		return Receipt{}, false, ErrInvalidSaleID
	}
	requestKey, err := uuid.Normalize(input.RequestKey)
	if err != nil || input.AmountCents <= 0 {
		return Receipt{}, false, ErrInvalidReceipt
	}
	if s == nil || s.repo == nil || s.protector == nil {
		return Receipt{}, false, ErrInvalidSaleConfig
	}
	if existing, found, err := s.repo.FindReceiptByRequestKey(ctx, ownerID, requestKey); err != nil {
		return Receipt{}, false, err
	} else if found {
		opened, err := s.openReceipt(existing)
		if err != nil {
			return Receipt{}, false, err
		}
		if opened.SaleID != saleID || opened.AmountCents != input.AmountCents {
			return Receipt{}, false, ErrReceiptIdempotencyConflict
		}
		return opened, true, nil
	}
	record, err := s.repo.GetSale(ctx, ownerID, saleID)
	if err != nil {
		return Receipt{}, false, err
	}
	sale, err := s.openSale(ownerID, record)
	if err != nil {
		return Receipt{}, false, err
	}
	newReceipt := Receipt{UserID: ownerID, SaleID: saleID, AmountCents: input.AmountCents, RequestKey: requestKey}
	newReceipt.ID, err = uuid.New()
	if err != nil {
		return Receipt{}, false, err
	}
	newReceipt.ReceivedAt = time.Now().UTC().Truncate(time.Microsecond)
	protected, err := s.protector.encrypt("agrofin.sale.receipt.v1", ownerID, saleID, newReceipt.ID, []string{newReceipt.ReceivedAt.Format(time.RFC3339Nano)}, newReceipt.AmountCents)
	if err != nil {
		return Receipt{}, false, err
	}
	stored := ProtectedReceipt{ID: newReceipt.ID, UserID: ownerID, SaleID: saleID, ReceivedAt: newReceipt.ReceivedAt, Amount: protected, RequestKey: requestKey}
	creation := newReceipt
	replayed := false
	err = s.repo.withSaleReceiptLock(ctx, ownerID, saleID, requestKey, func(tx receiptTx, existing *ProtectedReceipt) error {
		if existing != nil {
			opened, err := s.openReceipt(*existing)
			if err != nil {
				return err
			}
			if opened.SaleID != saleID || opened.AmountCents != input.AmountCents {
				return ErrReceiptIdempotencyConflict
			}
			creation = opened
			replayed = true
			return nil
		}
		prior, err := tx.List(ctx, ownerID, saleID)
		if err != nil {
			return err
		}
		var received int64
		for _, p := range prior {
			amount, err := s.openReceipt(p)
			if err != nil {
				return err
			}
			received, err = addCents(received, amount.AmountCents)
			if err != nil {
				return err
			}
		}
		if received > sale.NetCents || input.AmountCents > sale.NetCents-received {
			return ErrReceiptExceedsBalance
		}
		inserted, ok, err := tx.Insert(ctx, stored)
		if err != nil {
			return err
		}
		if !ok {
			opened, err := s.openReceipt(inserted)
			if err != nil {
				return err
			}
			if opened.SaleID != saleID || opened.AmountCents != input.AmountCents {
				return ErrReceiptIdempotencyConflict
			}
			creation = opened
			replayed = true
		}
		return nil
	})
	if err != nil {
		return Receipt{}, false, err
	}
	return creation, replayed, nil
}

func (s *Service) openSale(userID string, record ProtectedSaleRecord) (Sale, error) {
	p := record.Sale
	sale := Sale{ID: p.ID, UserID: userID, SoldAt: calendarDate.Normalize(p.SoldAt), BuyerName: p.BuyerName, Notes: p.Notes}
	for _, item := range p.Items {
		amount, err := s.protector.decrypt("agrofin.sale.unit-price.v1", userID, p.ID, item.ID, []string{string(item.SaleType), saleDateAttribute(sale.SoldAt)}, item.UnitPrice)
		if err != nil {
			return Sale{}, err
		}
		if item.Quantity <= 0 || amount <= 0 {
			return Sale{}, ErrInvalidProtectedSaleAmount
		}
		line, err := multiplyCents(amount, int64(item.Quantity))
		if err != nil {
			return Sale{}, err
		}
		sale.GrossCents, err = addCents(sale.GrossCents, line)
		if err != nil {
			return Sale{}, err
		}
		sale.Items = append(sale.Items, SaleItem{ID: item.ID, SaleType: item.SaleType, Quantity: item.Quantity, UnitPriceCents: amount, LineAmountCents: line})
	}
	if p.Discount != nil {
		discount, err := s.protector.decrypt("agrofin.sale.discount.v1", userID, p.ID, p.Discount.ID, []string{saleDateAttribute(sale.SoldAt), p.Discount.Reason}, p.Discount.Amount)
		if err != nil {
			return Sale{}, err
		}
		if discount > sale.GrossCents {
			return Sale{}, ErrDiscountExceedsGross
		}
		sale.Discount = &SaleDiscount{ID: p.Discount.ID, Reason: p.Discount.Reason, AmountCents: discount}
		sale.DiscountCents = discount
	}
	sale.NetCents = sale.GrossCents - sale.DiscountCents
	for _, p := range record.Receipts {
		receipt, err := s.openReceipt(p)
		if err != nil {
			return Sale{}, err
		}
		sale.ReceivedCents, err = addCents(sale.ReceivedCents, receipt.AmountCents)
		if err != nil {
			return Sale{}, err
		}
	}
	if sale.ReceivedCents > sale.NetCents {
		return Sale{}, ErrReceiptExceedsBalance
	}
	sale.ReceivableCents = sale.NetCents - sale.ReceivedCents
	return sale, nil
}

func (s *Service) openReceipt(p ProtectedReceipt) (Receipt, error) {
	amount, err := s.protector.decrypt("agrofin.sale.receipt.v1", p.UserID, p.SaleID, p.ID, []string{p.ReceivedAt.Format(time.RFC3339Nano)}, p.Amount)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{ID: p.ID, UserID: p.UserID, SaleID: p.SaleID, ReceivedAt: p.ReceivedAt, AmountCents: amount, RequestKey: p.RequestKey}, nil
}

func multiplyCents(unit, quantity int64) (int64, error) {
	if quantity <= 0 || unit <= 0 || unit > math.MaxInt64/quantity {
		return 0, ErrSaleOverflow
	}
	return unit * quantity, nil
}
