// Declara erros de domínio para vendas e recebimentos.
package vendas

import "errors"

var (
	ErrInvalidSaleOwner           = errors.New("vendas: cliente da venda inválido")
	ErrInvalidSaleDate            = errors.New("vendas: data da venda inválida")
	ErrInvalidSaleItems           = errors.New("vendas: venda precisa conter itens")
	ErrInvalidSaleType            = errors.New("vendas: classificação inválida")
	ErrDuplicateSaleType          = errors.New("vendas: classificação repetida")
	ErrInvalidSaleQuantity        = errors.New("vendas: quantidade inválida")
	ErrInvalidSalePrice           = errors.New("vendas: preço unitário inválido")
	ErrInvalidSaleDiscount        = errors.New("vendas: desconto ou motivo inválido")
	ErrDiscountExceedsGross       = errors.New("vendas: desconto excede o valor bruto")
	ErrSaleOverflow               = errors.New("vendas: total excede int64")
	ErrInvalidSaleID              = errors.New("vendas: identificador da venda inválido")
	ErrSaleNotFound               = errors.New("vendas: venda não encontrada")
	ErrInvalidReceipt             = errors.New("vendas: recebimento inválido")
	ErrReceiptExceedsBalance      = errors.New("vendas: recebimento excede o saldo a receber")
	ErrReceiptIdempotencyConflict = errors.New("vendas: chave de idempotência usada com outro pedido")
	ErrInvalidSaleConfig          = errors.New("vendas: repositório ou cifra não configurados")
	ErrInvalidSaleCipherConfig    = errors.New("vendas: configuração de cifra inválida")
	ErrInvalidSaleCipherContext   = errors.New("vendas: contexto de cifra inválido")
	ErrInvalidProtectedSaleAmount = errors.New("vendas: envelope financeiro inválido")
)
