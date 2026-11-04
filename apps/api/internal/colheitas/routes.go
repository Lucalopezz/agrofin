package colheitas

import "net/http"

// RegisterRoutes mantém lançamentos e tarifas privados e recebe a sessão na composição.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /harvest-rates", requireSession(http.HandlerFunc(handler.list)))
	mux.Handle("POST /harvest-rates", requireSession(http.HandlerFunc(handler.create)))
	mux.Handle("GET /harvest-entries", requireSession(http.HandlerFunc(handler.listHarvestEntries)))
	mux.Handle("POST /harvest-entries", requireSession(http.HandlerFunc(handler.createHarvest)))
}
