// Implementa criação, soma e abertura dos envelopes de pagamento.
package pagamentos

import (
	"context"
	"strings"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type paymentCreatorRepository interface {
	withEmployeePaymentLock(
		ctx context.Context,
		userID, employeeID string,
		operation func(paymentTransaction) error,
	) error
	ListPayments(ctx context.Context, userID string, periodStart, periodEnd time.Time) ([]ProtectedPayment, error)
	ListPaymentsByPaidAt(ctx context.Context, userID string, start, end time.Time) ([]ProtectedPaymentRecord, error)
	ListPaymentsCoveringDate(ctx context.Context, userID, employeeID string, date time.Time) ([]ProtectedPayment, error)
}

type paymentAmountProtector interface {
	EncryptPaymentAmount(payment Payment) (ProtectedPayment, error)
	DecryptPaymentAmount(payment ProtectedPayment) (int64, error)
}

// Create recebe a entrada validada, calcula o devido e grava um único pagamento protegido.
func (s *PayrollService) Create(ctx context.Context, userID string, input CreatePaymentInput) (Payment, error) {
	if err := ctx.Err(); err != nil {
		return Payment{}, err
	}
	// O request entrega os campos normalizados; o fim é exclusivo no banco.
	request := Payment{UserID: userID, EmployeeID: input.EmployeeID, PeriodStart: input.PeriodStart, PeriodEnd: input.PeriodEnd.AddDate(0, 0, 1), AmountCents: input.AmountCents, Notes: input.Notes}
	var err error
	if s == nil || s.repo == nil || s.protector == nil || s.harvest == nil || s.daily == nil {
		return Payment{}, ErrInvalidPaymentConfig
	}
	var creation Payment
	err = s.repo.withEmployeePaymentLock(ctx, request.UserID, request.EmployeeID,
		func(tx paymentTransaction) error {
			// O devido é recalculado depois do bloqueio do funcionário: agora colheitas
			// podem ser editadas, então o total lido antes da transação poderia ficar obsoleto.
			dueSummary, err := s.SummarizeDue(ctx, request.UserID, request.inclusivePeriod())
			if err != nil {
				return err
			}
			dueAmount := dueAmountForEmployee(dueSummary, request.EmployeeID)

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
			if err := tx.InsertPayment(ctx, protected); err != nil {
				return err
			}
			creation = request
			return nil
		})
	if err != nil {
		return Payment{}, err
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

func dueAmountForEmployee(summary PeriodSummary, employeeID string) int64 {
	for _, employee := range summary.Employees {
		if employee.EmployeeID != employeeID {
			continue
		}
		return employee.DueAmountCents
	}
	return 0
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
		PaidAt: paidAt, Notes: strings.TrimSpace(protected.Notes),
	}
	amount, err := s.protector.DecryptPaymentAmount(ProtectedPayment{
		ID: id, UserID: userID, EmployeeID: employeeID,
		PeriodStart: periodStart, PeriodEnd: periodEnd,
		PaidAt: paidAt, Amount: protected.Amount, Notes: metadata.Notes,
	})
	if err != nil {
		return Payment{}, err
	}
	metadata.AmountCents = amount
	return metadata, nil
}

var _ paymentAmountProtector = (*Cipher)(nil)
