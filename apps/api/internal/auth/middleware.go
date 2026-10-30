package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
)

type sessionResolver interface {
	AuthenticateSession(ctx context.Context, token string) (Principal, error)
}

// Middleware resolve a sessão antes de deixar a requisição acessar rotas privadas.
// Sua dependência só valida o token e obtém o principal, sem acoplar o middleware
// ao Repository ou às demais operações do Service.
type Middleware struct {
	resolver     sessionResolver
	cookieSecure bool
}

// NewMiddleware recebe o serviço que valida sessões e a política do cookie do ambiente.
func NewMiddleware(resolver sessionResolver, cookieSecure bool) *Middleware {
	return &Middleware{resolver: resolver, cookieSecure: cookieSecure}
}

// RequireSession só continua quando o cookie corresponde a uma sessão ativa no servidor.
func (m *Middleware) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// O cookie é a credencial apresentada pelo navegador. Não aceitamos user_id
		// em query, JSON ou cabeçalho, pois esses valores seriam escolhidos pelo cliente.
		cookie, err := r.Cookie(sessionCookieName(m.cookieSecure))
		if err != nil {
			clearSessionCookie(w, m.cookieSecure)
			writeError(w, http.StatusUnauthorized, "autenticação necessária")
			return
		}

		// A existência do cookie, por si só, nunca autoriza: o Service transforma
		// seu valor em hash e confirma no banco que a sessão segue ativa.
		principal, err := m.resolver.AuthenticateSession(r.Context(), cookie.Value)
		if errors.Is(err, ErrInvalidSession) {
			clearSessionCookie(w, m.cookieSecure)
			writeError(w, http.StatusUnauthorized, "sessão inválida ou expirada")
			return
		}
		if err != nil {
			log.Print("falha ao validar sessão")
			writeError(w, http.StatusInternalServerError, "não foi possível validar a sessão")
			return
		}

		// Derivar do contexto HTTP preserva prazo e cancelamento até o handler e o SQL.
		ctx := WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
