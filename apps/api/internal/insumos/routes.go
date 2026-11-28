// Registra as rotas de insumos protegidas por sessão.
package insumos

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("POST /input-products", requireSession(http.HandlerFunc(h.create)))
	mux.Handle("GET /input-products", requireSession(http.HandlerFunc(h.list)))
}
