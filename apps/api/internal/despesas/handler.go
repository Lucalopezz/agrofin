// Coordena os casos de uso HTTP de despesas.
package despesas

import (
	"context"
	"net/http"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type expenseService interface {
	Create(context.Context, string, Input) (Expense, error)
	List(context.Context, string, time.Time, time.Time) ([]Expense, error)
	Update(context.Context, string, string, Input) (Expense, error)
	Delete(context.Context, string, string) error
}
type Handler struct{ service expenseService }

func NewHandler(s expenseService) *Handler { return &Handler{service: s} }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	e, err := h.service.Create(r.Context(), p.UserID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response(e))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}
	start, end, ok := decodePeriod(w, r)
	if !ok {
		return
	}
	expenses, err := h.service.List(r.Context(), p.UserID, start, end)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]expenseResponse, 0, len(expenses))
	for _, e := range expenses {
		out = append(out, response(e))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	id, ok := decodeExpenseID(w, r)
	if !ok {
		return
	}
	e, err := h.service.Update(r.Context(), p.UserID, id, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response(e))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}
	id, ok := decodeExpenseID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), p.UserID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
