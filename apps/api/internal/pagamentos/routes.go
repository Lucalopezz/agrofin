// Registra as rotas privadas de criação de pagamento e resumo da folha.
package pagamentos

import "net/http"

// RegisterRoutes registra a criação de pagamentos e o resumo financeiro privado.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /payroll-periods/summary", requireSession(http.HandlerFunc(handler.summarizePeriod)))
	mux.Handle("POST /employee-payments", requireSession(http.HandlerFunc(handler.createPayment)))
}
