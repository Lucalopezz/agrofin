// Package money compartilha regras para valores representados em centavos inteiros.
package money

// IsPositiveCents informa se um valor financeiro é maior que zero.
// Tarifas, diárias e pagamentos não aceitam lançamentos de valor zero.
func IsPositiveCents(amountCents int64) bool {
	return amountCents > 0
}
