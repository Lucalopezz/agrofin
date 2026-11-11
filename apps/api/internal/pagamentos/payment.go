// Define pagamentos, entradas de criação e entidades protegidas para persistência.
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
	PeriodEnd   time.Time // limite exclusivo do intervalo persistido
	PaidAt      time.Time
	AmountCents int64
	Notes       string
	RequestKey  string
}

// PaidAt será da aplicação, não do cliente, para evitar que o usuário force
// a data de pagamento e gere inconsistências com o histórico de lançamentos.
type CreatePaymentInput struct {
	EmployeeID string
	// PeriodEnd é inclusivo na entrada; a entidade gravada usa fim exclusivo.
	PeriodStart time.Time
	PeriodEnd   time.Time
	AmountCents int64
	Notes       string
	RequestKey  string // UUID recebido no cabeçalho Idempotency-Key.
}

// PaymentCreation informa se o resultado foi criado agora ou veio de uma repetição idempotente.
type PaymentCreation struct {
	Payment Payment
	// Replayed indica que a chave já existia e o serviço devolveu o pagamento original, sem criar outro.
	Replayed bool
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

// inclusivePeriod converte o intervalo semiaberto persistido para a forma usada nas consultas HTTP.
func (payment Payment) inclusivePeriod() Period {
	return Period{StartDate: payment.PeriodStart, EndDate: payment.PeriodEnd.AddDate(0, 0, -1)}
}
