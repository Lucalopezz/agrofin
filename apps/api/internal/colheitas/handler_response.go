package colheitas

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

type rateResponse struct {
	ID          string  `json:"id"`
	BoxType     BoxType `json:"box_type"`
	AmountCents int64   `json:"amount_cents"`
	ValidFrom   string  `json:"valid_from"`
	ValidUntil  *string `json:"valid_until"`
}

func writeCreateRateResponse(w http.ResponseWriter, rate Rate, err error) {
	switch {
	case errors.Is(err, ErrInvalidBoxType):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo box_type deve ser normal ou moagem"})
	case errors.Is(err, ErrInvalidRateAmount):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo amount_cents deve ser maior que zero"})
	case errors.Is(err, ErrInvalidRatePeriod):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "A vigência da tarifa é inválida"})
	case errors.Is(err, ErrRateOverlap):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "Já existe uma tarifa desse tipo com vigência sobreposta"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para cadastrar tarifa esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// Não registrar o erro do driver nem dados da tarifa reduz exposição em logs.
		log.Print("erro ao persistir tarifa protegida")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível cadastrar a tarifa"})
	default:
		response := rateResponse{
			ID: rate.ID, BoxType: rate.BoxType, AmountCents: rate.AmountCents,
			ValidFrom: rate.ValidFrom.Format(rateDateLayout),
		}
		if rate.ValidUntil != nil {
			validUntil := rate.ValidUntil.Format(rateDateLayout)
			response.ValidUntil = &validUntil
		}
		writeJSON(w, http.StatusCreated, response)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Tarifas contêm valores financeiros; não permitimos que navegador ou proxy os armazene.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// O status já foi enviado; uma segunda resposta produziria JSON inválido.
		log.Print("erro ao escrever resposta de tarifas")
	}
}
