// Declara erros de domínio para cadastro e estado de funcionários.
package funcionarios

import "errors"

var (
	ErrInvalidName = errors.New("funcionarios: nome obrigatório")
	ErrNotFound    = errors.New("funcionarios: funcionário não encontrado")
)
