package auth

import (
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Manager contém somente os dados prontos para persistência; a senha original
// não atravessa a fronteira entre Service e Repository.
type Manager struct {
	ID string
	// Name é o nome de exibição; não substitui nem altera o login usado para entrar.
	Name string
	// LoginName já chega normalizado pelo Service, para que a regra não fique duplicada no SQL.
	LoginName string
	// PasswordHash é verificável, mas não reversível; a senha original não faz parte da entidade.
	PasswordHash string
	// Active permite invalidar login e sessões sem apagar o histórico associado ao gerente.
	Active bool
}

// ManagerProfile contém somente os campos que a API pode mostrar ao gerente.
// Separar esse tipo de Manager evita devolver o hash de senha em respostas HTTP.
type ManagerProfile struct {
	ID        string
	Name      string
	LoginName string
	Active    bool
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

// normalizeManagerName preserva a forma escrita, mas remove espaços externos e
// unifica representações Unicode equivalentes antes de validar e persistir.
func normalizeManagerName(name string) (string, error) {
	if !utf8.ValidString(name) {
		return "", invalidInput("nome do gerente inválido")
	}
	name = strings.TrimSpace(norm.NFC.String(name))
	runeCount := utf8.RuneCountInString(name)
	if runeCount == 0 || runeCount > 100 {
		return "", invalidInput("nome do gerente deve ter entre 1 e 100 caracteres")
	}
	return name, nil
}
