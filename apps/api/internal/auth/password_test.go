package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLoginName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{name: "remove espaços e converte para minúsculas", input: "  Gerente_1  ", expected: "gerente_1"},
		{name: "nome vazio", input: "  ", shouldError: true},
		{name: "nome curto demais", input: "ab", shouldError: true},
		{name: "nome longo demais", input: strings.Repeat("a", 65), shouldError: true},
		{name: "caractere não permitido", input: "gerente@1", shouldError: true},
		{name: "separador no início", input: ".gerente", shouldError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeLoginName(tt.input)
			if tt.shouldError {
				assert.True(t, isValidationError(err))
				assert.Empty(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestValidateNewPassword(t *testing.T) {
	decomposedPassword := strings.Repeat("a", 14) + "e\u0301"
	tests := []struct {
		name        string
		password    string
		loginName   string
		expected    string
		shouldError bool
	}{
		{
			name:      "normaliza unicode em NFC",
			password:  decomposedPassword,
			loginName: "gerente",
			expected:  strings.Repeat("a", 14) + "é",
		},
		{name: "menos de quinze caracteres", password: "curta", loginName: "gerente", shouldError: true},
		{name: "mais de 128 caracteres", password: strings.Repeat("a", 129), loginName: "gerente", shouldError: true},
		{name: "senha comum bloqueada", password: "passwordpassword", loginName: "gerente", shouldError: true},
		{name: "senha igual ao login sem diferença de caixa", password: "ABCDEFGHIJKLMNOP", loginName: "abcdefghijklmnop", shouldError: true},
		{name: "utf8 inválido", password: string([]byte{0xff}), loginName: "gerente", shouldError: true},
		{name: "entrada com bytes demais", password: strings.Repeat("a", 513), loginName: "gerente", shouldError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateNewPassword(tt.password, tt.loginName)
			if tt.shouldError {
				assert.True(t, isValidationError(err))
				assert.Empty(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
