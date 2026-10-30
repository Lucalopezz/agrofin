package auth

import "context"

// O tipo privado, em vez de uma string, evita colisões com chaves de outros pacotes.
type principalContextKey struct{}

// Principal contém a identidade que a autenticação confirmou para uma requisição.
type Principal struct {
	UserID string
}

// WithPrincipal permite que o middleware compartilhe a identidade validada sem
// aceitar IDs do cliente. O contexto derivado preserva o cancelamento e o prazo
// originais; não cria um contexto independente.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext retorna somente a identidade anexada pelo middleware de
// autenticação. O bool é false se não houver principal ou se ele estiver incompleto.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.UserID != ""
}
