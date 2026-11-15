// Registra as rotas de despesas protegidas por sessão.
package despesas

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler, session func(http.Handler) http.Handler) {
	mux.Handle("POST /expenses", session(http.HandlerFunc(h.create)))
	mux.Handle("GET /expenses", session(http.HandlerFunc(h.list)))
	mux.Handle("PUT /expenses/{id}", session(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /expenses/{id}", session(http.HandlerFunc(h.delete)))
}
