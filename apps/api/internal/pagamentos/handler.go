// Coordena os endpoints HTTP de resumo da folha e criação de pagamento.
package pagamentos

import (
	"context"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type payrollService interface {
	Summarize(ctx context.Context, userID string, period Period) (PeriodSummary, error)
	Create(ctx context.Context, userID string, input CreatePaymentInput) (PaymentCreation, error)
}

// Handler coordena a consulta HTTP sem conhecer SQL nem abrir cifras diretamente.
type Handler struct {
	service payrollService
}

func NewHandler(service payrollService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) summarizePeriod(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}
	period, ok := decodePeriodQuery(w, r)
	if !ok {
		return
	}
	summary, err := h.service.Summarize(r.Context(), principal.UserID, period)
	writePeriodSummaryResponse(w, summary, err)
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}
	input, ok := decodeCreatePaymentRequest(w, r)
	if !ok {
		return
	}
	creation, err := h.service.Create(r.Context(), principal.UserID, input)
	writeCreatePaymentResponse(w, creation, err)
}
