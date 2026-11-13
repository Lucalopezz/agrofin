// Registra as rotas privadas de vendas e recebimentos.
package vendas

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("POST /sales", requireSession(http.HandlerFunc(handler.createSale)))
	mux.Handle("GET /sales", requireSession(http.HandlerFunc(handler.listSales)))
	mux.Handle("POST /sales/{id}/receipts", requireSession(http.HandlerFunc(handler.createReceipt)))
}
