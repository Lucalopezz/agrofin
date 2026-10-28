package funcionarios

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
		expectedErr  error
	}{
		{name: "nome simples", input: "Ana", expectedName: "Ana"},
		{name: "espaços nas extremidades", input: "  Ana Maria  ", expectedName: "Ana Maria"},
		{name: "nome vazio", input: "", expectedErr: ErrInvalidName},
		{name: "somente espaços", input: "   ", expectedErr: ErrInvalidName},
		// Colar texto pode trazer tabulações, quebras de linha e espaços Unicode;
		// esses caracteres nas extremidades não tornam um nome vazio válido.
		{name: "outros espaços em branco", input: "\t\n\r\u00a0", expectedErr: ErrInvalidName},
		// A normalização não deve alterar a grafia escolhida pelo cliente.
		{name: "preserva grafia e espaços internos", input: "\t João  da Conceição \n", expectedName: "João  da Conceição"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeName(tt.input)
			assert.Equal(t, tt.expectedName, got)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
