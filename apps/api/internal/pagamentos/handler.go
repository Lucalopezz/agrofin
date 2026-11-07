package pagamentos

import (
	"context"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type payrollSummaryService interface {
	Summarize(ctx context.Context, userID string, period Period) (PeriodSummary, error)
}

// Handler coordena a consulta HTTP sem conhecer SQL nem abrir cifras diretamente.
type Handler struct {
	service payrollSummaryService
}

func NewHandler(service payrollSummaryService) *Handler {
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
