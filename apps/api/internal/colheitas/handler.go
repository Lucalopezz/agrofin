package colheitas

import (
	"context"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type rateCreator interface {
	CreateRate(ctx context.Context, userID string, input CreateRateInput) (Rate, error)
}

// Handler adapta o cadastro de tarifas ao protocolo HTTP sem assumir a persistência.
type Handler struct {
	service rateCreator
}

func NewHandler(service rateCreator) *Handler {
	return &Handler{service: service}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	// O gerente vem da sessão validada; aceitar user_id do JSON permitiria gravar
	// uma tarifa em outra conta e também quebraria o vínculo autenticado da cifra.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	input, ok := decodeCreateRateRequest(w, r)
	if !ok {
		return
	}

	rate, err := h.service.CreateRate(r.Context(), principal.UserID, input)
	writeCreateRateResponse(w, rate, err)
}
