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
	ErrInvalidHarvestOwner        = errors.New("colheitas: gerente do lançamento obrigatório")
	ErrInvalidHarvestEmployee     = errors.New("colheitas: funcionário do lançamento inválido")
	ErrInvalidHarvestDate         = errors.New("colheitas: data do lançamento inválida")
	ErrInvalidHarvestPeriod       = errors.New("colheitas: período de consulta inválido")
	ErrInvalidHarvestItems        = errors.New("colheitas: lançamento precisa conter caixas")
	ErrInvalidHarvestQuantity     = errors.New("colheitas: quantidade de caixas deve ser positiva")
	ErrDuplicateHarvestItem       = errors.New("colheitas: tipo de caixa repetido no lançamento")
	ErrRateNotFound               = errors.New("colheitas: não há tarifa vigente para a data e o tipo")
	ErrEmployeeNotFound           = errors.New("colheitas: funcionário não encontrado")
	ErrEmployeeInactive           = errors.New("colheitas: funcionário inativo")
)
