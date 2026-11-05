package diarias

import "errors"

var (
	ErrInvalidDailyWorkOwner           = errors.New("diarias: gerente do lançamento obrigatório")
	ErrInvalidDailyWorkAmount          = errors.New("diarias: valor da diária inválido")
	ErrInvalidDailyWorkDate            = errors.New("diarias: data da diária inválida")
	ErrInvalidEmployeeID               = errors.New("diarias: funcionário da diária inválido")
	ErrInvalidDailyWorkConfig          = errors.New("diarias: configuração do serviço inválida")
	ErrInvalidDailyWorkCipherConfig    = errors.New("diarias: configuração da proteção financeira inválida")
	ErrInvalidDailyWorkCipherContext   = errors.New("diarias: contexto da proteção financeira inválido")
	ErrInvalidProtectedDailyWorkAmount = errors.New("diarias: valor protegido da diária inválido")
	ErrEmployeeNotFound                = errors.New("diarias: funcionário não encontrado")
	ErrEmployeeInactive                = errors.New("diarias: funcionário inativo")
)
