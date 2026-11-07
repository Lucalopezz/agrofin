package pagamentos

import "errors"

var (
	ErrInvalidPayrollOwner    = errors.New("pagamentos: gerente inválido")
	ErrInvalidPayrollPeriod   = errors.New("pagamentos: período inválido")
	ErrInvalidPayrollConfig   = errors.New("pagamentos: serviço de resumo não configurado")
	ErrInvalidPayrollAmount   = errors.New("pagamentos: valor calculado inválido")
	ErrPayrollSummaryOverflow = errors.New("pagamentos: total do resumo excede int64")
)
