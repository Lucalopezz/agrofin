// Serializa respostas financeiras de vendas e traduz erros para HTTP.
package vendas

import (
	"context"
	"encoding/json"
	"errors"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"log"
	"net/http"
	"time"
)

type errorResponse struct {
	Error string `json:"error"`
}
type saleResponse struct {
	ID              string                `json:"id"`
	SoldAt          string                `json:"sold_at"`
	BuyerName       string                `json:"buyer_name,omitempty"`
	Notes           string                `json:"notes,omitempty"`
	Items           []saleItemResponse    `json:"items"`
	GrossCents      int64                 `json:"gross_amount_cents"`
	Discount        *saleDiscountResponse `json:"discount,omitempty"`
	NetCents        int64                 `json:"net_amount_cents"`
	ReceivedCents   int64                 `json:"received_amount_cents"`
	ReceivableCents int64                 `json:"receivable_amount_cents"`
}
type saleItemResponse struct {
	SaleType        SaleType `json:"sale_type"`
	Quantity        int32    `json:"quantity"`
	UnitPriceCents  int64    `json:"unit_price_cents"`
	LineAmountCents int64    `json:"line_amount_cents"`
}
type saleDiscountResponse struct {
	Reason      string `json:"reason"`
	AmountCents int64  `json:"amount_cents"`
}
type receiptResponse struct {
	ID          string `json:"id"`
	SaleID      string `json:"sale_id"`
	ReceivedAt  string `json:"received_at"`
	AmountCents int64  `json:"amount_cents"`
}

func writeSaleResponse(w http.ResponseWriter, sale Sale, err error) {
	switch {
	case errors.Is(err, ErrInvalidSaleDate), errors.Is(err, ErrInvalidSaleItems), errors.Is(err, ErrInvalidSaleType), errors.Is(err, ErrDuplicateSaleType), errors.Is(err, ErrInvalidSaleQuantity), errors.Is(err, ErrInvalidSalePrice), errors.Is(err, ErrInvalidSaleDiscount), errors.Is(err, ErrDiscountExceedsGross), errors.Is(err, ErrInvalidSaleOwner):
		writeJSON(w, 400, errorResponse{Error: "Confira data, classificações, quantidades, preços e desconto"})
	case errors.Is(err, ErrSaleOverflow):
		writeJSON(w, 400, errorResponse{Error: "O total da venda excede o limite permitido"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, 504, errorResponse{Error: "Tempo para registrar a venda esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, 408, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		log.Print("erro ao registrar venda protegida")
		writeJSON(w, 500, errorResponse{Error: "Não foi possível registrar a venda"})
	default:
		writeJSON(w, 201, newSaleResponse(sale))
	}
}
func writeSaleListResponse(w http.ResponseWriter, sales []Sale, err error) {
	switch {
	case errors.Is(err, ErrInvalidSaleDate):
		writeJSON(w, 400, errorResponse{Error: "Informe um período válido"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, 504, errorResponse{Error: "Tempo para listar vendas esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, 408, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		log.Print("erro ao listar vendas protegidas")
		writeJSON(w, 500, errorResponse{Error: "Não foi possível listar as vendas"})
	default:
		out := make([]saleResponse, 0, len(sales))
		for _, sale := range sales {
			out = append(out, newSaleResponse(sale))
		}
		writeJSON(w, 200, out)
	}
}
func writeReceiptResponse(w http.ResponseWriter, receipt Receipt, replayed bool, err error) {
	switch {
	case errors.Is(err, ErrInvalidSaleID), errors.Is(err, ErrInvalidReceipt):
		writeJSON(w, 400, errorResponse{Error: "Confira o ID da venda, a chave e o valor do recebimento"})
	case errors.Is(err, ErrSaleNotFound):
		writeJSON(w, 404, errorResponse{Error: "Venda não encontrada"})
	case errors.Is(err, ErrReceiptExceedsBalance), errors.Is(err, ErrReceiptIdempotencyConflict):
		writeJSON(w, 409, errorResponse{Error: "O recebimento excede o saldo ou a chave já identifica outro pedido"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, 504, errorResponse{Error: "Tempo para registrar o recebimento esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, 408, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		log.Print("erro ao registrar recebimento protegido")
		writeJSON(w, 500, errorResponse{Error: "Não foi possível registrar o recebimento"})
	default:
		status := 201
		if replayed {
			status = 200
		}
		writeJSON(w, status, receiptResponse{ID: receipt.ID, SaleID: receipt.SaleID, ReceivedAt: receipt.ReceivedAt.Format(time.RFC3339Nano), AmountCents: receipt.AmountCents})
	}
}
func newSaleResponse(s Sale) saleResponse {
	items := make([]saleItemResponse, 0, len(s.Items))
	for _, i := range s.Items {
		items = append(items, saleItemResponse{SaleType: i.SaleType, Quantity: i.Quantity, UnitPriceCents: i.UnitPriceCents, LineAmountCents: i.LineAmountCents})
	}
	var discount *saleDiscountResponse
	if s.Discount != nil {
		discount = &saleDiscountResponse{Reason: s.Discount.Reason, AmountCents: s.Discount.AmountCents}
	}
	return saleResponse{ID: s.ID, SoldAt: s.SoldAt.Format(calendarDate.Layout), BuyerName: s.BuyerName, Notes: s.Notes, Items: items, GrossCents: s.GrossCents, Discount: discount, NetCents: s.NetCents, ReceivedCents: s.ReceivedCents, ReceivableCents: s.ReceivableCents}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resposta financeira de venda")
	}
}
