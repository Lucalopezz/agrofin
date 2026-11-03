package colheitas

import (
	"context"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type rateService interface {
	CreateRate(ctx context.Context, userID string, input CreateRateInput) (Rate, error)
	ListRates(ctx context.Context, userID string) ([]Rate, error)
}

type harvestCreator interface {
	Create(ctx context.Context, userID string, input CreateHarvestInput) (Harvest, error)
}

// Handler adapta as operações de colheita e tarifa ao protocolo HTTP, sem persistência.
type Handler struct {
	service        rateService
	harvestCreator harvestCreator
}

func NewHandler(service rateService, harvestCreator harvestCreator) *Handler {
	return &Handler{service: service, harvestCreator: harvestCreator}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	// A sessão é a única fonte do dono. A query não pode trocar a conta consultada.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	rates, err := h.service.ListRates(r.Context(), principal.UserID)
	writeListRatesResponse(w, rates, err)
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

func (h *Handler) createHarvest(w http.ResponseWriter, r *http.Request) {
	// O lançamento sempre pertence ao principal autenticado, nunca a um user_id
	// enviado no corpo junto com os dados da colheita.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	input, ok := decodeCreateHarvestRequest(w, r)
	if !ok {
		return
	}

	harvest, err := h.harvestCreator.Create(r.Context(), principal.UserID, input)
	writeCreateHarvestResponse(w, harvest, err)
}
