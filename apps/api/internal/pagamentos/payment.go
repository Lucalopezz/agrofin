package pagamentos

import (
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
)

type Payment struct {
	ID           string
	UserID       string
	EmployeeID   string
	PeriodStart  time.Time
	PeriodEnd    time.Time
	PaidAt       time.Time
	AmountCents  int64
	Notes        string
	RequestedKey string
}

type CreatePaymentInput struct {
	EmployeeID   string
	PeriodStart  time.Time
	PeriodEnd    time.Time
	AmountCents  int64
	Notes        string
	RequestedKey string
}

// normalizePaymentAmount aplica a regra compartilhada e preserva o erro do domínio de pagamentos.
func normalizePaymentAmount(amountCents int64) (int64, error) {
	if !money.IsPositiveCents(amountCents) {
		return 0, ErrInvalidPaymentAmount
	}
	return amountCents, nil
}
