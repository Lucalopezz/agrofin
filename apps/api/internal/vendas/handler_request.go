// Decodifica corpos e filtros HTTP do domínio de vendas.
package vendas

import (
	"errors"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
	"net/http"
	"net/url"
)

type createSaleRequest struct {
	SoldAt    string               `json:"sold_at"`
	BuyerName string               `json:"buyer_name"`
	Notes     string               `json:"notes"`
	Items     []saleItemRequest    `json:"items"`
	Discount  *saleDiscountRequest `json:"discount"`
}
type saleItemRequest struct {
	SaleType       SaleType `json:"sale_type"`
	Quantity       int32    `json:"quantity"`
	UnitPriceCents int64    `json:"unit_price_cents"`
}
type saleDiscountRequest struct {
	Reason      string `json:"reason"`
	AmountCents int64  `json:"amount_cents"`
}
type receiptRequest struct {
	AmountCents int64 `json:"amount_cents"`
}

func decodeCreateSaleRequest(w http.ResponseWriter, r *http.Request) (CreateSaleInput, bool) {
	request, err := httpjson.Decode[createSaleRequest](w, r)
	if errors.Is(err, httpjson.ErrUnsupportedMediaType) {
		writeJSON(w, 415, errorResponse{Error: "Envie o corpo como application/json"})
		return CreateSaleInput{}, false
	}
	if errors.Is(err, httpjson.ErrBodyTooLarge) {
		writeJSON(w, 413, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreateSaleInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, 400, errorResponse{Error: "Envie um único objeto JSON com os dados da venda"})
		return CreateSaleInput{}, false
	}
	soldAt, err := calendarDate.Parse(request.SoldAt)
	if err != nil {
		writeJSON(w, 400, errorResponse{Error: "sold_at deve usar AAAA-MM-DD"})
		return CreateSaleInput{}, false
	}
	items := make([]SaleItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, SaleItemInput{SaleType: item.SaleType, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents})
	}
	var discount *SaleDiscountInput
	if request.Discount != nil {
		discount = &SaleDiscountInput{Reason: request.Discount.Reason, AmountCents: request.Discount.AmountCents}
	}
	return CreateSaleInput{SoldAt: soldAt, BuyerName: request.BuyerName, Notes: request.Notes, Items: items, Discount: discount}, true
}
func decodeReceiptRequest(w http.ResponseWriter, r *http.Request) (CreateReceiptInput, bool) {
	requestKey := r.Header.Get("Idempotency-Key")
	request, err := httpjson.Decode[receiptRequest](w, r)
	if errors.Is(err, httpjson.ErrUnsupportedMediaType) {
		writeJSON(w, 415, errorResponse{Error: "Envie o corpo como application/json"})
		return CreateReceiptInput{}, false
	}
	if errors.Is(err, httpjson.ErrBodyTooLarge) {
		writeJSON(w, 413, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreateReceiptInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, 400, errorResponse{Error: "Envie um único objeto JSON com o recebimento"})
		return CreateReceiptInput{}, false
	}
	return CreateReceiptInput{AmountCents: request.AmountCents, RequestKey: requestKey}, true
}
func decodeSalePeriod(w http.ResponseWriter, r *http.Request) (SalePeriod, bool) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeJSON(w, 400, errorResponse{Error: "Parâmetros de consulta inválidos"})
		return SalePeriod{}, false
	}
	startv, hs := query["start_date"]
	endv, he := query["end_date"]
	if len(query) != 2 || !hs || !he || len(startv) != 1 || len(endv) != 1 {
		writeJSON(w, 400, errorResponse{Error: "Informe start_date e end_date uma única vez"})
		return SalePeriod{}, false
	}
	start, err := calendarDate.Parse(startv[0])
	if err != nil {
		writeJSON(w, 400, errorResponse{Error: "start_date deve usar AAAA-MM-DD"})
		return SalePeriod{}, false
	}
	end, err := calendarDate.Parse(endv[0])
	if err != nil {
		writeJSON(w, 400, errorResponse{Error: "end_date deve usar AAAA-MM-DD"})
		return SalePeriod{}, false
	}
	return SalePeriod{StartDate: start, EndDate: end}, true
}
