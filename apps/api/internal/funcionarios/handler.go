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
	// O middleware fornece a identidade; esta checagem mantém o handler seguro se usado sem a rota protegida.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "autenticação necessária"})
		return
	}

	// O corpo contém só dados do funcionário. O dono do registro vem da sessão,
	// assim o cliente não consegue criar alguém em nome de outro user_id.
	input, ok := decodeCreateEmployeeRequest(w, r)
	if !ok {
		return
	}

	// O contexto e o user ID seguem da requisição: o primeiro preserva prazo e
	// cancelamento; o segundo vem da sessão. O Service valida o nome, monta Employee
	// e só então chama o Repository.
	employee, err := h.service.Create(r.Context(), principal.UserID, input.Name)
	writeCreateEmployeeResponse(w, employee, err)
}
