// Monta respostas financeiras e traduz erros de folha para status HTTP.
package pagamentos

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

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
}

type employeeResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
			})
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// O resumo revela valores financeiros da conta; não deve ser guardado em cache.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resumo financeiro")
	}
}
