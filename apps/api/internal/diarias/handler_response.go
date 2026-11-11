// Monta respostas HTTP e traduz erros da criação de diárias.
package diarias

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

type dailyWorkResponse struct {
	ID          string `json:"id"`
	EmployeeID  string `json:"employee_id"`
	WorkDate    string `json:"work_date"`
	AmountCents int64  `json:"amount_cents"`
	Notes       string `json:"notes,omitempty"`
}

func writeCreateDailyWorkResponse(w http.ResponseWriter, work DailyWork, err error) {
	switch {
	case errors.Is(err, ErrInvalidDailyWorkOwner), errors.Is(err, ErrInvalidEmployeeID),
		errors.Is(err, ErrInvalidDailyWorkDate), errors.Is(err, ErrInvalidDailyWorkAmount):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Confira funcionário, data e valor da diária"})
	case errors.Is(err, ErrEmployeeNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "Funcionário não encontrado"})
	case errors.Is(err, ErrEmployeeInactive):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "Não é possível registrar diária para funcionário inativo"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para registrar diária esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// Erros internos podem conter dados do banco ou da cifra; logamos somente a operação.
		log.Print("erro ao persistir diária protegida")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível registrar a diária"})
	default:
		writeJSON(w, http.StatusCreated, newDailyWorkResponse(work))
	}
}

func newDailyWorkResponse(work DailyWork) dailyWorkResponse {
	return dailyWorkResponse{
		ID: work.ID, EmployeeID: work.EmployeeID,
		WorkDate:    work.WorkDate.Format(calendarDate.Layout),
		AmountCents: work.AmountCents, Notes: work.Notes,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resposta de diária")
	}
}
