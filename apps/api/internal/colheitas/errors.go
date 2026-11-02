package colheitas

import "errors"

var (
	ErrInvalidBoxType             = errors.New("colheitas: tipo de caixa inválido")
	ErrInvalidRateAmount          = errors.New("colheitas: tarifa deve ser maior que zero")
	ErrInvalidRatePeriod          = errors.New("colheitas: período de vigência inválido")
	ErrInvalidRateCipherConfig    = errors.New("colheitas: configuração de proteção da tarifa inválida")
	ErrInvalidRateCipherContext   = errors.New("colheitas: identificação para proteção da tarifa inválida")
	ErrInvalidProtectedRateAmount = errors.New("colheitas: valor protegido da tarifa inválido")
	ErrInvalidRateOwner           = errors.New("colheitas: gerente da tarifa obrigatório")
	ErrRateOverlap                = errors.New("colheitas: tarifa com vigência sobreposta")
)
