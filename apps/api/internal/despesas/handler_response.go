// Traduz despesas e erros de domínio para respostas HTTP.
package despesas

import (
	"context"
	"encoding/json"
	"errors"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"log"
	"net/http"
)

type expenseResponse struct {
	ID          string `json:"id"`
	PaidAt      string `json:"paid_at"`
	Category    string `json:"category"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
}

func response(e Expense) expenseResponse {
	return expenseResponse{e.ID, e.PaidAt.Format(calendarDate.Layout), e.Category, e.Description, e.AmountCents}
}
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		writeError(w, 400, "Confira os dados da despesa e o período")
	case errors.Is(err, ErrNotFound):
		writeError(w, 404, "Despesa não encontrada")
	case errors.Is(err, context.Canceled):
		writeError(w, 408, "Requisição cancelada")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, 504, "Tempo da requisição esgotado")
	default:
		log.Print("erro ao processar despesa protegida")
		writeError(w, 500, "Não foi possível processar a despesa")
	}
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resposta de despesa")
	}
}
