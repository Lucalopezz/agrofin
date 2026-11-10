package pagamentos

import (
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type Payment struct {
	ID          string
	UserID      string
	EmployeeID  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	PaidAt      time.Time
	AmountCents int64
	Notes       string
	RequestKey  string
}

// PaidAt será da aplicação, não do cliente, para evitar que o usuário force
// a data de pagamento e gere inconsistências com o histórico de lançamentos.
type CreatePaymentInput struct {
	EmployeeID  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	AmountCents int64
	Notes       string
	RequestKey  string
}

// ProtectedPayment é a entidade pronta para gravação; ela transporta o envelope,
// nunca os centavos em claro.
type ProtectedPayment struct {
	ID          string
	UserID      string
	EmployeeID  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	PaidAt      time.Time
	Amount      ProtectedPaymentAmount
	Notes       string
	RequestKey  string
}

// normalizePaymentAmount aplica a regra compartilhada e preserva o erro do domínio de pagamentos.
func normalizePaymentAmount(amountCents int64) (int64, error) {
	if !money.IsPositiveCents(amountCents) {
		return 0, ErrInvalidPaymentAmount
	}
	return amountCents, nil
}

func normalizePaymentRequestKey(requestKey string) (string, error) {
	key, err := uuid.Normalize(requestKey)
	if err != nil {
		return "", ErrInvalidPaymentRequestKey
	}
	return key, nil
}
