// Decodifica corpos e filtros HTTP do domínio de vendas.
package vendas

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
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

func decodeCreateSaleRequest(w http.ResponseWriter, r *http.Request, userID string) (CreateSaleInput, bool) {
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
	input := CreateSaleInput{SoldAt: soldAt, BuyerName: request.BuyerName, Notes: request.Notes, Items: items, Discount: discount}
	input, err = normalizeSale(userID, input)
	if err != nil {
		writeSaleResponse(w, Sale{}, err)
		return CreateSaleInput{}, false
	}
	return input, true
}
func decodeReceiptRequest(w http.ResponseWriter, r *http.Request) (CreateReceiptInput, bool) {
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
	if request.AmountCents <= 0 {
		writeJSON(w, 400, errorResponse{Error: "O valor do recebimento deve ser maior que zero"})
		return CreateReceiptInput{}, false
	}
	return CreateReceiptInput{AmountCents: request.AmountCents}, true
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
	if end.Before(start) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve ser igual ou posterior a start_date"})
		return SalePeriod{}, false
	}
	return SalePeriod{StartDate: start, EndDate: end}, true
}

// decodeSaleID valida o identificador recebido no caminho antes de chamar o serviço.
func decodeSaleID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := uuid.Normalize(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 400, errorResponse{Error: "O ID da venda deve ser um UUID válido"})
		return "", false
	}
	return id, true
}

func normalizeSale(userID string, input CreateSaleInput) (CreateSaleInput, error) {
	if _, err := uuid.Normalize(userID); err != nil {
		return CreateSaleInput{}, ErrInvalidSaleOwner
	}
	if input.SoldAt.IsZero() {
		return CreateSaleInput{}, ErrInvalidSaleDate
	}
	if len(input.Items) == 0 {
		return CreateSaleInput{}, ErrInvalidSaleItems
	}

	input.BuyerName = strings.TrimSpace(input.BuyerName)
	input.Notes = strings.TrimSpace(input.Notes)
	seen := make(map[SaleType]struct{}, len(input.Items))
	for _, item := range input.Items {
		if !item.SaleType.isValid() {
			return CreateSaleInput{}, ErrInvalidSaleType
		}
		if _, ok := seen[item.SaleType]; ok {
			return CreateSaleInput{}, ErrDuplicateSaleType
		}
		seen[item.SaleType] = struct{}{}
		if item.Quantity <= 0 {
			return CreateSaleInput{}, ErrInvalidSaleQuantity
		}
		if !money.IsPositiveCents(item.UnitPriceCents) {
			return CreateSaleInput{}, ErrInvalidSalePrice
		}
	}
	if input.Discount != nil {
		discount := *input.Discount
		discount.Reason = strings.TrimSpace(discount.Reason)
		if discount.Reason == "" || !money.IsPositiveCents(discount.AmountCents) {
			return CreateSaleInput{}, ErrInvalidSaleDiscount
		}
		input.Discount = &discount
	}
	return input, nil
}
