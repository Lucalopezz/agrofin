// Monta respostas HTTP e traduz erros de funcionários para status públicos.
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

func writeListEmployeesResponse(w http.ResponseWriter, employees []Employee, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para listar funcionários esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// O erro do driver pode conter dados da consulta; registramos apenas a operação.
		log.Print("erro de persistência ao listar funcionários")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível listar os funcionários"})
	default:
		// O DTO omite o vínculo técnico UserID e garante [] mesmo com uma slice nil.
		response := make([]employeeResponse, 0, len(employees))
		for _, employee := range employees {
			response = append(response, employeeResponse{
				ID: employee.ID, Name: employee.Name, Active: employee.Active,
			})
		}
		writeJSON(w, http.StatusOK, response)
	}
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

func writeRenameEmployeeResponse(w http.ResponseWriter, employee Employee, err error) {
	switch {
	case errors.Is(err, ErrInvalidName):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Nome obrigatório"})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "Funcionário não encontrado"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para renomear funcionário esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// Erros do driver podem incluir dados privados; só a operação vai ao log.
		log.Print("erro de persistência ao renomear funcionário")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível renomear o funcionário"})
	default:
		writeJSON(w, http.StatusOK, employeeResponse{ID: employee.ID, Name: employee.Name, Active: employee.Active})
	}
}

func writeDeactivateEmployeeResponse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "Funcionário não encontrado"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para desativar funcionário esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// O log identifica a operação sem incluir detalhes privados do driver.
		log.Print("erro de persistência ao desativar funcionário")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível desativar o funcionário"})
	default:
		// 204 não transporta JSON nem outro corpo, mesmo em chamadas repetidas.
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeReactivateEmployeeResponse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "Funcionário não encontrado"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para reativar funcionário esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// Só a operação vai ao log; detalhes do driver podem conter dados privados.
		log.Print("erro de persistência ao reativar funcionário")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível reativar o funcionário"})
	default:
		// A repetição mantém o mesmo resultado e não envia representação no 204.
		w.WriteHeader(http.StatusNoContent)
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
