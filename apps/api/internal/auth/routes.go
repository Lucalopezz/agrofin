package auth

import "net/http"

// RegisterRoutes separa endpoints públicos de autenticação das rotas que exigem
// sessão. O main injeta o middleware para deixar a composição explícita.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	// Cadastro e login precisam ser alcançáveis sem sessão; o cadastro ainda exige
	// o segredo de bootstrap e só pode criar o gerente único.
	mux.HandleFunc("POST /auth/register", handler.register)
	mux.HandleFunc("POST /auth/login", handler.login)
	mux.HandleFunc("POST /auth/logout", handler.logout)
	// /me permite ao frontend recuperar o estado de autenticação após um reload.
	mux.Handle("GET /auth/me", requireSession(http.HandlerFunc(handler.me)))
}
