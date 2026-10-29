package funcionarios

import (
	"context"
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

// NewHandler recebe uma conta definida pelo servidor, nunca pelo corpo ou
// cabeçalhos do cadastro. A integração com o login ainda está pendente.
func NewHandler(service employeeService, accountID string) *Handler {
	return &Handler{service: service, accountID: accountID}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	// A aplicação ainda não fornece a conta do cliente autenticado. Até o login
	// ser integrado, interrompemos o cadastro antes de chamar o serviço.
	if h.accountID == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "cadastro indisponível até a integração com o login"})
		return
	}

	input, ok := decodeCreateEmployeeRequest(w, r)
	if !ok {
		return
	}

	// O contexto nasce na requisição e segue até o SQL. A conta é do servidor;
	// a normalização do nome e a montagem de Employee continuam no serviço.
	employee, err := h.service.Create(r.Context(), h.accountID, input.Name)
	writeCreateEmployeeResponse(w, employee, err)
}
