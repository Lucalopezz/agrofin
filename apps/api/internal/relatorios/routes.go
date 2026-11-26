// Registra o resumo e as consultas privadas que permitem conferir seus valores.
package relatorios

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /reports/period-summary", requireSession(periodHandler(h.summary)))
	mux.Handle("GET /reports/sold-sales", requireSession(periodHandler(h.soldSales)))
	mux.Handle("GET /employee-payments", requireSession(periodHandler(h.payments)))
	mux.Handle("GET /sale-receipts", requireSession(periodHandler(h.receipts)))
	mux.Handle("GET /daily-work", requireSession(periodHandler(h.dailyWorks)))
}
