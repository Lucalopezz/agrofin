package colheitas

import "net/http"

// RegisterRoutes mantém a criação de tarifas privada e recebe a sessão na composição da API.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("POST /harvest-rates", requireSession(http.HandlerFunc(handler.create)))
}
