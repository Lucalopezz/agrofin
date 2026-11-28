// Converte insumos e erros de persistência em respostas HTTP.
package insumos

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type productResponse struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Category          Category `json:"category"`
	DefaultUnit       string   `json:"default_unit"`
	Notes             string   `json:"notes,omitempty"`
	StockControlState string   `json:"stock_control_state"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newProductResponse(product Product) productResponse {
	return productResponse{
		ID:                product.ID,
		Name:              product.Name,
		Category:          product.Category,
		DefaultUnit:       product.DefaultUnit,
		Notes:             product.Notes,
		StockControlState: product.StockControlState,
	}
}

func writeListResponse(w http.ResponseWriter, products []Product) {
	// A conta sem insumos recebe uma lista vazia, mesmo quando o repositório retorna nil.
	response := make([]productResponse, 0, len(products))
	for _, product := range products {
		response = append(response, newProductResponse(product))
	}
	writeJSON(w, http.StatusOK, response)
}

func writePersistenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "Tempo para processar insumos esgotado")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "Requisição cancelada")
	default:
		// Erros do driver podem carregar dados privados da conta.
		log.Print("erro de persistência de insumos")
		writeError(w, http.StatusInternalServerError, "Não foi possível processar os insumos")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resposta de insumos")
	}
}
