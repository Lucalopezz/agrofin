package diarias

import (
	"context"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type dailyWorkService interface {
	Create(ctx context.Context, userID string, input CreateDailyWorkInput) (DailyWork, error)
}

// Handler adapta o caso de uso de diária ao HTTP, sem consultar o banco.
type Handler struct {
	service dailyWorkService
}

func NewHandler(service dailyWorkService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	input, ok := decodeCreateDailyWorkRequest(w, r)
	if !ok {
		return
	}
	work, err := h.service.Create(r.Context(), principal.UserID, input)
	writeCreateDailyWorkResponse(w, work, err)
}
