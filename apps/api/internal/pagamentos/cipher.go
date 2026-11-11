// Adapta a cifra financeira ao contexto autenticado dos pagamentos.
package pagamentos

import (
	"errors"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

const paymentAmountPurpose = "agrofin.employee-payment.amount.v1"

// paymentCipherPurpose impede misturar envelopes de pagamento com outros domínios.
type paymentCipherPurpose struct{}

// ProtectedPaymentAmount mantém o tipo de envelope específico do pagamento.
type ProtectedPaymentAmount = financialcipher.Amount[paymentCipherPurpose]

// Cipher adapta a cifra compartilhada ao contexto e aos erros do domínio de pagamentos.
type Cipher struct {
	amounts *financialcipher.Cipher[paymentCipherPurpose]
}

// NewCipher reutiliza o chaveiro validado na composição da API sem duplicar chaves.
func NewCipher(keyring *financialcipher.Keyring) (*Cipher, error) {
	amounts, err := financialcipher.NewCipher[paymentCipherPurpose](keyring)
	if err != nil {
		return nil, ErrInvalidPaymentCipherConfig
	}
	return &Cipher{amounts: amounts}, nil
}

// EncryptPaymentAmount cifra o valor de um pagamento já validado pelo Service.
// O ID precisa existir antes desta chamada porque também autentica o valor cifrado.
func (c *Cipher) EncryptPaymentAmount(payment Payment) (ProtectedPayment, error) {
	if c == nil || c.amounts == nil {
		return ProtectedPayment{}, ErrInvalidPaymentCipherConfig
	}

	amount, err := c.amounts.Encrypt(paymentAmountContext(payment), payment.AmountCents)
	if err != nil {
		return ProtectedPayment{}, mapPaymentCipherError(err)
	}
	return ProtectedPayment{
		ID: payment.ID, UserID: payment.UserID, EmployeeID: payment.EmployeeID,
		PeriodStart: payment.PeriodStart, PeriodEnd: payment.PeriodEnd,
		PaidAt: payment.PaidAt, Amount: amount, Notes: payment.Notes,
		RequestKey: payment.RequestKey,
	}, nil
}

// DecryptPaymentAmount abre o envelope usando os mesmos metadados autenticados na gravação.
func (c *Cipher) DecryptPaymentAmount(payment ProtectedPayment) (int64, error) {
	if c == nil || c.amounts == nil {
		return 0, ErrInvalidPaymentCipherConfig
	}
	amount, err := c.amounts.Decrypt(paymentAmountContext(Payment{
		ID: payment.ID, UserID: payment.UserID, EmployeeID: payment.EmployeeID,
		PeriodStart: payment.PeriodStart, PeriodEnd: payment.PeriodEnd,
		PaidAt: payment.PaidAt, RequestKey: payment.RequestKey,
	}), payment.Amount)
	return amount, mapPaymentCipherError(err)
}

// paymentAmountContext serializa em ordem estável os campos já normalizados pelo Service.
// Essa ordem integra o formato autenticado persistido e precisa permanecer estável.
func paymentAmountContext(payment Payment) financialcipher.Context[paymentCipherPurpose] {
	return financialcipher.Context[paymentCipherPurpose]{
		Purpose:  paymentAmountPurpose,
		UserID:   payment.UserID,
		RecordID: payment.ID,
		Attributes: []string{
			payment.EmployeeID,
			payment.PeriodStart.Format(calendarDate.Layout),
			payment.PeriodEnd.Format(calendarDate.Layout),
			payment.PaidAt.Format(time.RFC3339Nano),
			payment.RequestKey,
		},
	}
}

func mapPaymentCipherError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, financialcipher.ErrInvalidConfig):
		return ErrInvalidPaymentCipherConfig
	case errors.Is(err, financialcipher.ErrInvalidContext):
		return ErrInvalidPaymentCipherContext
	case errors.Is(err, financialcipher.ErrInvalidAmount):
		return ErrInvalidPaymentAmount
	case errors.Is(err, financialcipher.ErrInvalidProtected):
		return ErrInvalidProtectedPaymentAmount
	default:
		return err
	}
}
