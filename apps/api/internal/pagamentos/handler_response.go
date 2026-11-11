// Monta respostas financeiras e traduz erros de folha para status HTTP.
package pagamentos

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

type errorResponse struct {
	Error string `json:"error"`
}

type periodSummaryResponse struct {
	Period    periodResponse            `json:"period"`
	Employees []employeeSummaryResponse `json:"employees"`
}

type periodResponse struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type employeeSummaryResponse struct {
	Employee             employeeResponse `json:"employee"`
	NormalBoxes          int64            `json:"normal_boxes"`
	MillingBoxes         int64            `json:"milling_boxes"`
	NormalAmountCents    int64            `json:"normal_amount_cents"`
	MillingAmountCents   int64            `json:"milling_amount_cents"`
	HarvestAmountCents   int64            `json:"harvest_amount_cents"`
	DailyWorkAmountCents int64            `json:"daily_work_amount_cents"`
	DueAmountCents       int64            `json:"due_amount_cents"`
	PaidAmountCents      int64            `json:"paid_amount_cents"`
	RemainingAmountCents int64            `json:"remaining_amount_cents"`
}

type employeeResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type paymentResponse struct {
	ID          string `json:"id"`
	EmployeeID  string `json:"employee_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	PaidAt      string `json:"paid_at"`
	AmountCents int64  `json:"amount_cents"`
	Notes       string `json:"notes,omitempty"`
}

func writePeriodSummaryResponse(w http.ResponseWriter, summary PeriodSummary, err error) {
	switch {
	case errors.Is(err, ErrInvalidPayrollPeriod):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve ser igual ou posterior a start_date"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para resumir valores devidos esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// A falha pode vir do banco ou da autenticação de um envelope financeiro.
		// A resposta e o log não incluem valores nem detalhes da cifra.
		log.Print("erro ao resumir valores devidos por período")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível resumir os valores devidos"})
	default:
		response := periodSummaryResponse{
			Period: periodResponse{
				StartDate: summary.Period.StartDate.Format(calendarDate.Layout),
				EndDate:   summary.Period.EndDate.Format(calendarDate.Layout),
			},
			Employees: make([]employeeSummaryResponse, 0, len(summary.Employees)),
		}
		for _, employee := range summary.Employees {
			response.Employees = append(response.Employees, employeeSummaryResponse{
				Employee:    employeeResponse{ID: employee.EmployeeID, Name: employee.EmployeeName},
				NormalBoxes: employee.NormalBoxes, MillingBoxes: employee.MillingBoxes,
				NormalAmountCents:    employee.NormalAmountCents,
				MillingAmountCents:   employee.MillingAmountCents,
				HarvestAmountCents:   employee.HarvestAmountCents,
				DailyWorkAmountCents: employee.DailyWorkAmountCents,
				DueAmountCents:       employee.DueAmountCents,
				PaidAmountCents:      employee.PaidAmountCents,
				RemainingAmountCents: employee.RemainingAmountCents,
			})
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func writeCreatePaymentResponse(w http.ResponseWriter, creation PaymentCreation, err error) {
	switch {
	case errors.Is(err, ErrInvalidPaymentEmployee):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo employee_id deve ser um UUID válido"})
	case errors.Is(err, ErrInvalidPaymentRequestKey):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O cabeçalho Idempotency-Key deve ser um UUID válido"})
	case errors.Is(err, ErrInvalidPayrollOwner), errors.Is(err, ErrInvalidPaymentPeriod),
		errors.Is(err, ErrInvalidPaymentAmount):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Confira funcionário, período, valor e Idempotency-Key"})
	case errors.Is(err, ErrPaymentEmployeeNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "Funcionário não encontrado"})
	case errors.Is(err, ErrPaymentExceedsBalance):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "O valor excede o saldo devido no período"})
	case errors.Is(err, ErrPaymentIdempotencyConflict):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "Idempotency-Key já utilizada com dados diferentes"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para registrar o pagamento esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// Não registrar valor, envelope, chave de idempotência ou dados do funcionário no log.
		log.Print("erro ao criar pagamento protegido")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível registrar o pagamento"})
	default:
		status := http.StatusCreated
		if creation.Replayed {
			status = http.StatusOK
		}
		writeJSON(w, status, newPaymentResponse(creation.Payment))
	}
}

func newPaymentResponse(payment Payment) paymentResponse {
	period := payment.inclusivePeriod()
	return paymentResponse{
		ID: payment.ID, EmployeeID: payment.EmployeeID,
		StartDate:   period.StartDate.Format(calendarDate.Layout),
		EndDate:     period.EndDate.Format(calendarDate.Layout),
		PaidAt:      payment.PaidAt.UTC().Format(time.RFC3339Nano),
		AmountCents: payment.AmountCents, Notes: payment.Notes,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// As respostas financeiras incluem resumo e criação autorizada; o cliente não deve armazená-las em cache.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resumo financeiro")
	}
}
