package auth

import "context"

type principalContextKey struct{}

// Principal contém a identidade que a autenticação confirmou para uma requisição.
type Principal struct {
	UserID string
}

// WithPrincipal permite que o middleware compartilhe a identidade validada sem aceitar IDs do cliente.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext retorna somente a identidade anexada pelo middleware de autenticação.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.UserID != ""
}
