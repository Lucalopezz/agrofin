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
	// /me recupera o perfil após reload; PATCH altera o nome de exibição sem mudar login_name.
	mux.Handle("GET /auth/me", requireSession(http.HandlerFunc(handler.me)))
	mux.Handle("PATCH /auth/me", requireSession(http.HandlerFunc(handler.updateName)))
	// A única conta não consegue se reativar. O PATCH existe para desativação; a
	// reativação é uma ação operacional fora da sessão do gerente.
	mux.Handle("PATCH /auth/me/status", requireSession(http.HandlerFunc(handler.updateStatus)))
}
