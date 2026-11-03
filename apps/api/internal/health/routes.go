package health

import "net/http"

// RegisterRoutes registra a verificação pública do processo HTTP.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", handleHealth)
}
