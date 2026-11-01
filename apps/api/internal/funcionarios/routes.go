package funcionarios

import "net/http"

// RegisterRoutes mantém as rotas de funcionários privadas e recebe o middleware na composição da API.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	// O middleware envolve o handler: primeiro autentica, depois o handler pode
	// ler o principal e chamar o Service de funcionários.
	mux.Handle("GET /employees", requireSession(http.HandlerFunc(handler.list)))
	mux.Handle("POST /employees", requireSession(http.HandlerFunc(handler.create)))
	mux.Handle("PATCH /employees/{id}", requireSession(http.HandlerFunc(handler.rename)))
	mux.Handle("DELETE /employees/{id}", requireSession(http.HandlerFunc(handler.deactivate)))
}
