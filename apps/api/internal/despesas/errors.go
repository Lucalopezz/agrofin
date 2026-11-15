package despesas

import "errors"

var (
	ErrInvalid  = errors.New("despesas: dados inválidos")
	ErrNotFound = errors.New("despesas: despesa não encontrada")
)
