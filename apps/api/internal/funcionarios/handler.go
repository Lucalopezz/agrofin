package funcionarios

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
)

// O handler depende do caso de uso, sem acessar o banco ou validar o domínio.
type employeeService interface {
	Create(ctx context.Context, accountID, name string) (Employee, error)
}

type Handler struct {
	service   employeeService
	accountID string
}

// NewHandler recebe uma conta definida pelo servidor. Temporariamente ela vem
// da configuração explícita de desenvolvimento, nunca do corpo ou cabeçalhos.
func NewHandler(service employeeService, accountID string) *Handler {
	return &Handler{service: service, accountID: accountID}
}

type createEmployeeRequest struct {
	Name string `json:"name"`
}

type employeeResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	// Sem autenticação, só habilitamos a escrita quando a aplicação compõe
	// explicitamente o handler com uma conta de desenvolvimento validada.
	if h.accountID == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "cadastro indisponível: conta de desenvolvimento não configurada"})
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "envie o corpo como application/json"})
		return
	}

	// O cadastro contém apenas um nome. Limitamos o corpo, incluindo espaços
	// extras, para não consumir memória sem limite ao decodificar a requisição.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input *createEmployeeRequest
	err = decoder.Decode(&input)
	if err == nil && input != nil {
		// Um segundo JSON ou conteúdo extra não pode ser ignorado depois do nome.
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				err = errors.New("mais de um valor json")
			} else {
				err = trailingErr
			}
		}
	}
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "corpo da requisição excede 4 KiB"})
		return
	}
	if err != nil || input == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "envie um único objeto JSON contendo somente name"})
		return
	}

	// O contexto nasce na requisição e segue até o SQL. A conta é do servidor;
	// a normalização do nome e a montagem de Employee continuam no serviço.
	employee, err := h.service.Create(r.Context(), h.accountID, input.Name)
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
