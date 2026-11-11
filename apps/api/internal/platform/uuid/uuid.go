// Package uuid gera identificadores e normaliza UUIDs recebidos pela aplicação.
package uuid

import (
	"errors"

	googleuuid "github.com/google/uuid"
)

// ErrInvalidFormat indica que o valor não usa o formato UUID canônico esperado.
var ErrInvalidFormat = errors.New("uuid: invalid format")

// New gera o identificador antes da persistência, quando ele já é necessário
// para compor o contexto autenticado da cifra do registro.
func New() (string, error) {
	id, err := googleuuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// Normalize valida o formato UUID canônico e devolve a representação em minúsculas.
// A forma estável é importante quando o identificador participa de consultas ou AAD.
func Normalize(value string) (string, error) {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return "", ErrInvalidFormat
	}

	id, err := googleuuid.Parse(value)
	if err != nil {
		return "", ErrInvalidFormat
	}
	return id.String(), nil
}
