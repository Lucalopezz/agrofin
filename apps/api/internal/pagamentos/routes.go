package pagamentos

import "net/http"

// RegisterRoutes registra a consulta privada do resumo financeiro do período.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /payroll-periods/summary", requireSession(http.HandlerFunc(handler.summarizePeriod)))
}
