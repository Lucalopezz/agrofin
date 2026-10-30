package auth

import "time"

// Manager contém somente os dados prontos para persistência; a senha original
// não atravessa a fronteira entre Service e Repository.
type Manager struct {
	ID string
	// LoginName já chega normalizado pelo Service, para que a regra não fique duplicada no SQL.
	LoginName string
	// PasswordHash é verificável, mas não reversível; a senha original não faz parte da entidade.
	PasswordHash string
	// Active permite invalidar login e sessões sem apagar o histórico associado ao gerente.
	Active bool
}

// Session contém o hash do token, nunca o valor que será entregue ao navegador.
type Session struct {
	UserID string
	// TokenHash é SHA-256 dos 32 bytes aleatórios do cookie, nunca o token utilizável.
	TokenHash []byte
	// ExpiresAt é o prazo absoluto; o prazo de inatividade usa last_seen_at no banco.
	ExpiresAt time.Time
}

// createdSession mantém o token em memória apenas até o Handler emitir o cookie.
type createdSession struct {
	Token     string
	ExpiresAt time.Time
}
