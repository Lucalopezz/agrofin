// Coordena os endpoints HTTP de vendas e recebimentos sem acessar persistência.
package vendas

import (
	"context"
	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
	"net/http"
)

type saleService interface {
	Create(context.Context, string, CreateSaleInput) (Sale, error)
	List(context.Context, string, SalePeriod) ([]Sale, error)
	Receive(context.Context, string, CreateReceiptInput) (Receipt, bool, error)
}
type Handler struct{ service saleService }

func NewHandler(service saleService) *Handler { return &Handler{service: service} }

func (h *Handler) createSale(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, 401, errorResponse{Error: "Autenticação necessária"})
		return
	}
	in, ok := decodeCreateSaleRequest(w, r)
	if !ok {
		return
	}
	sale, err := h.service.Create(r.Context(), p.UserID, in)
	writeSaleResponse(w, sale, err)
}
func (h *Handler) listSales(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, 401, errorResponse{Error: "Autenticação necessária"})
		return
	}
	period, ok := decodeSalePeriod(w, r)
	if !ok {
		return
	}
	sales, err := h.service.List(r.Context(), p.UserID, period)
	writeSaleListResponse(w, sales, err)
}
func (h *Handler) createReceipt(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, 401, errorResponse{Error: "Autenticação necessária"})
		return
	}
	saleID, err := uuid.Normalize(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 400, errorResponse{Error: "O ID da venda deve ser um UUID válido"})
		return
	}
	in, ok := decodeReceiptRequest(w, r)
	if !ok {
		return
	}
	in.SaleID = saleID
	receipt, replayed, err := h.service.Receive(r.Context(), p.UserID, in)
	writeReceiptResponse(w, receipt, replayed, err)
}
