// Package funcionarios contém a definição da estrutura Employee e funções relacionadas à validação de nomes de funcionários.
package funcionarios

import (
	"errors"
	"strings"
)

// Publico a outros pacotes

type Employee struct {
	ID     string
	UserID string
	Name   string
	Active bool
}

var ErrInvalidName = errors.New("funcionarios: nome obrigatório")

func normalizeName(name string) (string, error) {
	// Remove espaços apenas das extremidades para evitar nomes vazios ou
	// preenchimento acidental, preservando a escrita e os espaços internos.
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidName
	}

	return name, nil
}
