// Protege valores de vendas e calcula valores vendidos e recebidos.
package vendas

import (
	"context"
	"math"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type saleRepository interface {
	CreateSale(context.Context, ProtectedSale) error
	GetSale(context.Context, string, string) (ProtectedSaleRecord, error)
	ListSales(context.Context, string, time.Time, time.Time) ([]ProtectedSaleRecord, error)
	ListSoldSales(context.Context, string, time.Time, time.Time) ([]ProtectedSaleRecord, error)
	ListReceiptsByReceivedAt(context.Context, string, time.Time, time.Time) ([]ProtectedReceiptRecord, error)
	withSaleReceiptLock(context.Context, string, string, func(receiptTx) error) error
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
	// O request já validou classificações, valores, desconto e limites da soma.
	sale := Sale{UserID: userID, SoldAt: input.SoldAt, BuyerName: input.BuyerName, Notes: input.Notes}
	for _, item := range input.Items {
		line := int64(item.Quantity) * item.UnitPriceCents
		sale.GrossCents += line
		sale.Items = append(sale.Items, SaleItem{SaleType: item.SaleType, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents, LineAmountCents: line})
	}
	if input.Discount != nil {
		sale.Discount = &SaleDiscount{Reason: input.Discount.Reason, AmountCents: input.Discount.AmountCents}
		sale.DiscountCents = input.Discount.AmountCents
	}
	sale.NetCents = sale.GrossCents - sale.DiscountCents
	sale.ReceivableCents = sale.NetCents
	var err error
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
	if s == nil || s.repo == nil || s.protector == nil {
		return nil, ErrInvalidSaleConfig
	}
	records, err := s.repo.ListSales(ctx, userID, period.StartDate, period.EndDate)
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

// Receive registra cada envio válido e serializa a conferência do saldo da venda.
func (s *Service) Receive(ctx context.Context, userID string, input CreateReceiptInput) (Receipt, error) {
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	// O request valida o ID e o valor; a conta vem da sessão autenticada.
	ownerID, saleID := userID, input.SaleID
	record, err := s.repo.GetSale(ctx, ownerID, saleID)
	if err != nil {
		return Receipt{}, err
	}
	sale, err := s.openSale(ownerID, record)
	if err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{UserID: ownerID, SaleID: saleID, AmountCents: input.AmountCents}
	receipt.ID, err = uuid.New()
	if err != nil {
		return Receipt{}, err
	}
	receipt.ReceivedAt = time.Now().UTC().Truncate(time.Microsecond)
	amount, err := s.protector.encrypt("agrofin.sale.receipt.v1", ownerID, saleID, receipt.ID, []string{receipt.ReceivedAt.Format(time.RFC3339Nano)}, receipt.AmountCents)
	if err != nil {
		return Receipt{}, err
	}
	stored := ProtectedReceipt{ID: receipt.ID, UserID: ownerID, SaleID: saleID, ReceivedAt: receipt.ReceivedAt, Amount: amount}
	// O bloqueio da venda mantém a soma e a inserção atômicas sob concorrência.
	err = s.repo.withSaleReceiptLock(ctx, ownerID, saleID, func(tx receiptTx) error {
		prior, err := tx.List(ctx, ownerID, saleID)
		if err != nil {
			return err
		}
		var received int64
		for _, p := range prior {
			opened, err := s.openReceipt(p)
			if err != nil {
				return err
			}
			received, err = addCents(received, opened.AmountCents)
			if err != nil {
				return err
			}
		}
		if received > sale.NetCents || input.AmountCents > sale.NetCents-received {
			return ErrReceiptExceedsBalance
		}
		return tx.Insert(ctx, stored)
	})
	if err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

// openSale abre os valores protegidos e recompõe os totais para devolver a venda legível.
// A consulta já foi limitada ao dono no repositório; o userID também vincula a decifra.
func (s *Service) openSale(userID string, record ProtectedSaleRecord) (Sale, error) {
	sale, err := s.openSoldSale(userID, record.Sale)
	if err != nil {
		return Sale{}, err
	}
	for _, p := range record.Receipts {
		// Abre cada recebimento para somar somente valores conferidos pela cifra.
		receipt, err := s.openReceipt(p)
		if err != nil {
			return Sale{}, err
		}
		sale.ReceivedCents, err = addCents(sale.ReceivedCents, receipt.AmountCents)
		if err != nil {
			return Sale{}, err
		}
	}
	// Dados protegidos inconsistentes não podem gerar saldo negativo.
	if sale.ReceivedCents > sale.NetCents {
		return Sale{}, ErrReceiptExceedsBalance
	}
	sale.ReceivableCents = sale.NetCents - sale.ReceivedCents
	return sale, nil
}

// openSoldSale compartilha o cálculo do vendido entre a listagem e os relatórios.
func (s *Service) openSoldSale(userID string, p ProtectedSale) (Sale, error) {
	sale := Sale{ID: p.ID, UserID: userID, SoldAt: calendarDate.Normalize(p.SoldAt), BuyerName: p.BuyerName, Notes: p.Notes}
	for _, item := range p.Items {
		// Tipo e data fazem parte do contexto da cifra; repetir os valores aqui
		// detecta troca de envelope entre itens ou vendas.
		amount, err := s.protector.decrypt("agrofin.sale.unit-price.v1", userID, p.ID, item.ID, []string{string(item.SaleType), saleDateAttribute(sale.SoldAt)}, item.UnitPrice)
		if err != nil {
			return Sale{}, err
		}
		// Dados inválidos no banco não podem virar totais aparentemente válidos.
		if item.Quantity <= 0 || amount <= 0 {
			return Sale{}, ErrInvalidProtectedSaleAmount
		}
		// Calcula cada linha em centavos e soma com proteção contra estouro.
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
		// A justificativa também faz parte do contexto autenticado da cifra.
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
	// O desconto define o líquido da venda no dia em que ela foi realizada.
	sale.NetCents = sale.GrossCents - sale.DiscountCents
	return sale, nil
}

// openReceipt decifra um recebimento para que o serviço possa exibi-lo e somá-lo ao saldo.
// Os dados da venda, do recebimento e da data autenticam o contexto da cifra.
func (s *Service) openReceipt(p ProtectedReceipt) (Receipt, error) {
	// A data e os IDs autenticam o valor; isso detecta envelope trocado entre
	// recebimentos ou entre vendas.
	// O PostgreSQL pode devolver o mesmo instante em outro fuso; a cifra foi
	// criada em UTC e exige exatamente a mesma representação ao abrir.
	amount, err := s.protector.decrypt("agrofin.sale.receipt.v1", p.UserID, p.SaleID, p.ID, []string{p.ReceivedAt.UTC().Format(time.RFC3339Nano)}, p.Amount)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{ID: p.ID, UserID: p.UserID, SaleID: p.SaleID, ReceivedAt: p.ReceivedAt, AmountCents: amount}, nil
}

// multiplyCents calcula o total da linha sem permitir valores inválidos ou estouro de int64.
func multiplyCents(unit, quantity int64) (int64, error) {
	// Confere os operandos e o limite antes de multiplicar para evitar que um
	// overflow em int64 transforme o valor da linha em um número incorreto.
	if quantity <= 0 || unit <= 0 || unit > math.MaxInt64/quantity {
		return 0, ErrSaleOverflow
	}
	return unit * quantity, nil
}
