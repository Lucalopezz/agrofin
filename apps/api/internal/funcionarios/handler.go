package funcionarios

import (
	"context"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

// O handler depende do caso de uso, sem acessar o banco ou validar o domínio.
type employeeService interface {
	Create(ctx context.Context, userID, name string) (Employee, error)
}

type Handler struct {
	service employeeService
}

func NewHandler(service employeeService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	// O ID precisa vir do middleware por requisição; sem login integrado, não criamos registros sem proprietário.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "cadastro indisponível até a integração com o login"})
		return
	}

	input, ok := decodeCreateEmployeeRequest(w, r)
	if !ok {
		return
	}

	// O contexto nasce na requisição e segue até o SQL. O user ID é do servidor;
	// a normalização do nome e a montagem de Employee continuam no serviço.
	employee, err := h.service.Create(r.Context(), principal.UserID, input.Name)
	writeCreateEmployeeResponse(w, employee, err)
}
