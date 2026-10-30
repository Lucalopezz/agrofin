package auth

import "errors"

// Erros sentinela permitem que o Handler mapeie falhas conhecidas para status
// HTTP sem depender do texto interno retornado pelo banco ou pelo Service.
var ErrManagerAlreadyRegistered = errors.New("auth: gerente já cadastrado")
var ErrInvalidCredentials = errors.New("auth: credenciais inválidas")
var ErrInvalidSession = errors.New("auth: sessão inválida")
var ErrAuthenticationBusy = errors.New("auth: autenticação temporariamente ocupada")

var errManagerNotFound = errors.New("auth: gerente não encontrado")

type validationError struct {
	// A mensagem é destinada a uma resposta de validação; falhas internas usam
	// erros genéricos para não revelar detalhes de banco ou credenciais.
	message string
}

func (e validationError) Error() string {
	return e.message
}

func invalidInput(message string) error {
	return validationError{message: message}
}

func isValidationError(err error) bool {
	var target validationError
	return errors.As(err, &target)
}
