// Package funcionarios define o cadastro Employee e normaliza seus nomes.
package funcionarios

import (
	"strings"
)

// Publico a outros pacotes

type Employee struct {
	ID     string
	UserID string
	Name   string
	Active bool
}

func normalizeName(name string) (string, error) {
	// Remove espaços apenas das extremidades para evitar nomes vazios ou
	// preenchimento acidental, preservando a escrita e os espaços internos.
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidName
	}

	return name, nil
}
