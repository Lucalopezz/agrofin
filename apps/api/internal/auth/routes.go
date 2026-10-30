package auth

import "net/http"

// RegisterRoutes mantém o cadastro inicial junto da funcionalidade de autenticação.
func RegisterRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("POST /auth/register", handler.register)
}
