// Coordena o cadastro e a listagem HTTP de insumos para a sessão atual.
package insumos

import (
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	// O vínculo com a conta vem do principal confirmado pelo middleware.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}
	input, ok := decodeCreateRequest(w, r)
	if !ok {
		return
	}

	product, err := h.service.Create(r.Context(), principal.UserID, input)
	if err != nil {
		writePersistenceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newProductResponse(product))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	products, err := h.service.List(r.Context(), principal.UserID)
	if err != nil {
		writePersistenceError(w, err)
		return
	}
	writeListResponse(w, products)
}
