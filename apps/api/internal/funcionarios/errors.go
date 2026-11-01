package funcionarios

import "errors"

var (
	ErrInvalidName = errors.New("funcionarios: nome obrigatório")
	ErrNotFound    = errors.New("funcionarios: funcionário não encontrado")
)
