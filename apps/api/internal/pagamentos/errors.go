// Declara erros de domínio para folha, pagamentos e proteção financeira.
package pagamentos

import "errors"

var (
	ErrInvalidPayrollOwner           = errors.New("pagamentos: gerente inválido")
	ErrInvalidPayrollPeriod          = errors.New("pagamentos: período inválido")
	ErrInvalidPayrollConfig          = errors.New("pagamentos: serviço de resumo não configurado")
	ErrInvalidPayrollAmount          = errors.New("pagamentos: valor calculado inválido")
	ErrPayrollSummaryOverflow        = errors.New("pagamentos: total do resumo excede int64")
	ErrInvalidPaymentPeriod          = errors.New("pagamentos: período de pagamento inválido")
	ErrInvalidPaymentEmployee        = errors.New("pagamentos: funcionário do pagamento inválido")
	ErrInvalidPaymentAmount          = errors.New("pagamentos: valor do pagamento inválido")
	ErrInvalidPaymentRequestKey      = errors.New("pagamentos: chave de requisição inválida")
	ErrInvalidPaymentCipherConfig    = errors.New("pagamentos: configuração da proteção financeira inválida")
	ErrInvalidPaymentCipherContext   = errors.New("pagamentos: contexto da proteção financeira inválido")
	ErrInvalidProtectedPaymentAmount = errors.New("pagamentos: valor protegido inválido")
)
