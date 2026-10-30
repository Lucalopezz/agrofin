package auth

import "net/http"

// RegisterRoutes mantém o cadastro inicial junto da funcionalidade de autenticação.
func RegisterRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("POST /api/v1/auth/register", handler.register)
}
