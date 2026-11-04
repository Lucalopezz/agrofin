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

type harvestService interface {
	Create(ctx context.Context, userID string, input CreateHarvestInput) (Harvest, error)
	List(ctx context.Context, userID string, period HarvestPeriod) ([]Harvest, error)
	Summarize(ctx context.Context, userID string, period HarvestPeriod) ([]HarvestSummary, error)
}

// Handler adapta as operações de colheita e tarifa ao protocolo HTTP, sem persistência.
type Handler struct {
	service        rateService
	harvestService harvestService
}

func NewHandler(service rateService, harvestService harvestService) *Handler {
	return &Handler{service: service, harvestService: harvestService}
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

	harvest, err := h.harvestService.Create(r.Context(), principal.UserID, input)
	writeCreateHarvestResponse(w, harvest, err)
}

func (h *Handler) listHarvestEntries(w http.ResponseWriter, r *http.Request) {
	// A sessão define a conta consultada; employee_id e user_id não são filtros
	// da listagem para evitar cruzar escopos ou transformar metadados em autorização.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	period, ok := decodeHarvestPeriodQuery(w, r)
	if !ok {
		return
	}

	harvests, err := h.harvestService.List(r.Context(), principal.UserID, period)
	writeListHarvestEntriesResponse(w, harvests, err)
}

func (h *Handler) summarizeHarvestEntries(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	period, ok := decodeHarvestPeriodQuery(w, r)
	if !ok {
		return
	}

	summaries, err := h.harvestService.Summarize(r.Context(), principal.UserID, period)
	writeHarvestSummaryResponse(w, summaries, err)
}
