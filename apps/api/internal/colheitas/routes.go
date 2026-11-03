package colheitas

import "net/http"

// RegisterRoutes mantém as tarifas privadas e recebe a sessão na composição da API.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /harvest-rates", requireSession(http.HandlerFunc(handler.list)))
	mux.Handle("POST /harvest-rates", requireSession(http.HandlerFunc(handler.create)))
}
