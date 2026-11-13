// Implementa criação idempotente, soma e abertura dos envelopes de pagamento.
package pagamentos

import (
	"context"
	"strings"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type paymentCreatorRepository interface {
	FindPaymentByRequestKey(ctx context.Context, userID, requestKey string) (ProtectedPayment, bool, error)
	withEmployeePaymentLock(
		ctx context.Context,
		userID, employeeID, requestKey string,
		operation func(paymentTransaction, *ProtectedPayment) error,
	) error
	ListPayments(ctx context.Context, userID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error)
	ListPaymentsCoveringDate(ctx context.Context, userID, employeeID string, date time.Time) ([]ProtectedPayment, error)
}

type paymentAmountProtector interface {
	EncryptPaymentAmount(payment Payment) (ProtectedPayment, error)
	DecryptPaymentAmount(payment ProtectedPayment) (int64, error)
}

// normalizeCreatePayment pertence ao caso de uso: valida a entrada e monta a entidade antes da cifra.
func normalizeCreatePayment(userID string, input CreatePaymentInput) (Payment, error) {
	ownerID, err := uuid.Normalize(userID)
	if err != nil {
		return Payment{}, ErrInvalidPayrollOwner
	}
	employeeID, err := uuid.Normalize(input.EmployeeID)
	if err != nil {
		return Payment{}, ErrInvalidPaymentEmployee
	}
	start, endInclusive, err := calendarDate.NormalizeInclusivePeriod(input.PeriodStart, input.PeriodEnd)
	if err != nil {
		return Payment{}, ErrInvalidPaymentPeriod
	}
	amountCents, err := normalizePaymentAmount(input.AmountCents)
	if err != nil {
		return Payment{}, err
	}
	requestKey, err := normalizePaymentRequestKey(input.RequestKey)
	if err != nil {
		return Payment{}, err
	}

	return Payment{
		UserID: ownerID, EmployeeID: employeeID,
		PeriodStart: start, PeriodEnd: endInclusive.AddDate(0, 0, 1),
		AmountCents: amountCents, Notes: strings.TrimSpace(input.Notes),
		RequestKey: requestKey,
	}, nil
}

// Create valida a entrada, calcula o devido e grava um único pagamento protegido.
func (s *PayrollService) Create(ctx context.Context, userID string, input CreatePaymentInput) (PaymentCreation, error) {
	if err := ctx.Err(); err != nil {
		return PaymentCreation{}, err
	}
	request, err := normalizeCreatePayment(userID, input)
	if err != nil {
		return PaymentCreation{}, err
	}
	if s == nil || s.repo == nil || s.protector == nil || s.harvest == nil || s.daily == nil {
		return PaymentCreation{}, ErrInvalidPaymentConfig
	}
	// Uma repetição idêntica retorna o registro original mesmo que o cadastro de origem
	// tenha mudado depois do primeiro envio; request_key é a identidade do comando aceito.
	existing, found, err := s.repo.FindPaymentByRequestKey(ctx, request.UserID, request.RequestKey)
	if err != nil {
		return PaymentCreation{}, err
	}
	if found {
		payment, err := s.openProtectedPayment(existing, request.UserID)
		if err != nil {
			return PaymentCreation{}, err
		}
		if !samePaymentRequest(payment, request) {
			return PaymentCreation{}, ErrPaymentIdempotencyConflict
		}
		return PaymentCreation{Payment: payment, Replayed: true}, nil
	}

	var creation PaymentCreation
	err = s.repo.withEmployeePaymentLock(ctx, request.UserID, request.EmployeeID, request.RequestKey,
		func(tx paymentTransaction, existing *ProtectedPayment) error {
			if existing != nil {
				payment, err := s.openProtectedPayment(*existing, request.UserID)
				if err != nil {
					return err
				}
				if !samePaymentRequest(payment, request) {
					return ErrPaymentIdempotencyConflict
				}
				creation = PaymentCreation{Payment: payment, Replayed: true}
				return nil
			}
			// O devido é recalculado depois do bloqueio do funcionário: agora colheitas
			// podem ser editadas, então o total lido antes da transação poderia ficar obsoleto.
			dueSummary, err := s.summarizeDue(ctx, request.UserID, request.inclusivePeriod())
			if err != nil {
				return err
			}
			dueAmount, err := dueAmountForEmployee(dueSummary, request.EmployeeID)
			if err != nil {
				return err
			}

			previous, err := tx.ListPayments(ctx, request.EmployeeID, request.PeriodStart, request.PeriodEnd)
			if err != nil {
				return err
			}
			paidAmount, err := s.sumProtectedPayments(ctx, previous, request.UserID)
			if err != nil {
				return err
			}
			if paidAmount > dueAmount {
				return ErrInvalidPaymentBalance
			}
			remaining := dueAmount - paidAmount
			if request.AmountCents > remaining {
				return ErrPaymentExceedsBalance
			}
			if err := ctx.Err(); err != nil {
				return err
			}

			request.ID, err = uuid.New()
			if err != nil {
				return err
			}
			// O PostgreSQL armazena TIMESTAMPTZ em microssegundos; usar a mesma precisão
			// antes da cifra mantém idêntico o contexto autenticado após gravar e ler.
			request.PaidAt = time.Now().UTC().Truncate(time.Microsecond)
			protected, err := s.protector.EncryptPaymentAmount(request)
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			stored, inserted, err := tx.InsertPayment(ctx, protected)
			if err != nil {
				return err
			}
			if !inserted {
				// A restrição UNIQUE é uma última defesa para processos antigos ou escritores
				// que não usem a trava consultiva; ainda assim comparamos o conteúdo no serviço.
				existingPayment, err := s.openProtectedPayment(stored, request.UserID)
				if err != nil {
					return err
				}
				if !samePaymentRequest(existingPayment, request) {
					return ErrPaymentIdempotencyConflict
				}
				creation = PaymentCreation{Payment: existingPayment, Replayed: true}
				return nil
			}
			creation = PaymentCreation{Payment: request}
			return nil
		})
	if err != nil {
		return PaymentCreation{}, err
	}
	return creation, nil
}

// summarizePayments abre envelopes já filtrados por dono e período no repositório.
// O chamador fornece o dono e os limites de período normalizados.
func (s *PayrollService) summarizePayments(ctx context.Context, userID string, period Period) (map[string]int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ownerID := userID
	periodEndExclusive := period.EndDate.AddDate(0, 0, 1)
	payments, err := s.repo.ListPayments(ctx, ownerID, period.StartDate, periodEndExclusive)
	if err != nil {
		return nil, err
	}

	byEmployee := make(map[string]int64)
	for _, protected := range payments {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		payment, err := s.openProtectedPayment(protected, ownerID)
		if err != nil {
			return nil, err
		}
		if payment.PeriodStart != period.StartDate || payment.PeriodEnd != periodEndExclusive {
			return nil, ErrInvalidPaymentCipherContext
		}
		total, err := addAmount(byEmployee[payment.EmployeeID], payment.AmountCents)
		if err != nil {
			return nil, err
		}
		byEmployee[payment.EmployeeID] = total
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return byEmployee, nil
}

func dueAmountForEmployee(summary PeriodSummary, employeeID string) (int64, error) {
	for _, employee := range summary.Employees {
		if employee.EmployeeID != employeeID {
			continue
		}
		if employee.DueAmountCents < 0 {
			return 0, ErrInvalidPayrollAmount
		}
		return employee.DueAmountCents, nil
	}
	return 0, nil
}

func (s *PayrollService) sumProtectedPayments(ctx context.Context, payments []ProtectedPayment, userID string) (int64, error) {
	var total int64
	for _, protected := range payments {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		payment, err := s.openProtectedPayment(protected, userID)
		if err != nil {
			return 0, err
		}
		amount, err := addAmount(total, payment.AmountCents)
		if err != nil {
			return 0, err
		}
		total = amount
	}
	return total, nil
}

func (s *PayrollService) openProtectedPayment(protected ProtectedPayment, expectedUserID string) (Payment, error) {
	userID, err := uuid.Normalize(protected.UserID)
	if err != nil || userID != expectedUserID {
		return Payment{}, ErrInvalidPaymentCipherContext
	}
	id, err := uuid.Normalize(protected.ID)
	if err != nil {
		return Payment{}, ErrInvalidPaymentCipherContext
	}
	employeeID, err := uuid.Normalize(protected.EmployeeID)
	if err != nil {
		return Payment{}, ErrInvalidPaymentEmployee
	}
	requestKey, err := normalizePaymentRequestKey(protected.RequestKey)
	if err != nil {
		return Payment{}, ErrInvalidPaymentCipherContext
	}
	periodStart := calendarDate.Normalize(protected.PeriodStart)
	periodEnd := calendarDate.Normalize(protected.PeriodEnd)
	if periodStart.IsZero() || periodEnd.IsZero() || !periodEnd.After(periodStart) {
		return Payment{}, ErrInvalidPaymentPeriod
	}
	if protected.PaidAt.IsZero() {
		return Payment{}, ErrInvalidPaymentCipherContext
	}
	paidAt := protected.PaidAt.UTC().Truncate(time.Microsecond)
	metadata := Payment{
		ID: id, UserID: userID, EmployeeID: employeeID,
		PeriodStart: periodStart, PeriodEnd: periodEnd,
		PaidAt: paidAt, Notes: strings.TrimSpace(protected.Notes), RequestKey: requestKey,
	}
	amount, err := s.protector.DecryptPaymentAmount(ProtectedPayment{
		ID: id, UserID: userID, EmployeeID: employeeID,
		PeriodStart: periodStart, PeriodEnd: periodEnd,
		PaidAt: paidAt, Amount: protected.Amount, Notes: metadata.Notes, RequestKey: requestKey,
	})
	if err != nil {
		return Payment{}, err
	}
	if amount <= 0 {
		return Payment{}, ErrInvalidPaymentAmount
	}
	metadata.AmountCents = amount
	return metadata, nil
}

func samePaymentRequest(existing, requested Payment) bool {
	return existing.UserID == requested.UserID &&
		existing.EmployeeID == requested.EmployeeID &&
		existing.PeriodStart.Equal(requested.PeriodStart) &&
		existing.PeriodEnd.Equal(requested.PeriodEnd) &&
		existing.AmountCents == requested.AmountCents &&
		existing.Notes == requested.Notes &&
		existing.RequestKey == requested.RequestKey
}

var _ paymentAmountProtector = (*Cipher)(nil)
