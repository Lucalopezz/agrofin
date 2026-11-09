package pagamentos

import (
	"time"
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
