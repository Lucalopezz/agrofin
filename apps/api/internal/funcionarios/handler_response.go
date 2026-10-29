package funcionarios

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type employeeResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// A conversão dos erros para status HTTP fica na borda, sem expor detalhes do driver.
func writeCreateEmployeeResponse(w http.ResponseWriter, employee Employee, err error) {
	switch {
	case errors.Is(err, ErrInvalidName):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "nome obrigatório"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "tempo para cadastrar funcionário esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "requisição cancelada"})
	case err != nil:
		// Detalhes do driver podem incluir dados da consulta e não pertencem
		// à resposta HTTP nem aos logs deste cadastro.
		log.Print("erro de persistência ao cadastrar funcionário")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "não foi possível cadastrar o funcionário"})
	default:
		writeJSON(w, http.StatusCreated, employeeResponse{ID: employee.ID, Name: employee.Name, Active: employee.Active})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// O status já foi enviado: tentar escrever outra resposta corromperia o JSON.
		log.Print("erro ao escrever resposta de funcionários")
	}
}
