// Package uuid gera identificadores UUID v4 para entidades criadas pela aplicação.
package uuid

import googleuuid "github.com/google/uuid"

// New gera o identificador antes da persistência, quando ele já é necessário
// para compor o contexto autenticado da cifra do registro.
func New() (string, error) {
	id, err := googleuuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
