package funcionarios

import (
	"context"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

// O handler depende do caso de uso, sem acessar o banco ou validar o domínio.
type employeeService interface {
	Create(ctx context.Context, userID, name string) (Employee, error)
	List(ctx context.Context, userID string, active *bool) ([]Employee, error)
	Rename(ctx context.Context, userID, employeeID, name string) (Employee, error)
	Deactivate(ctx context.Context, userID, employeeID string) error
	Reactivate(ctx context.Context, userID, employeeID string) error
}

type Handler struct {
	service employeeService
}

func NewHandler(service employeeService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	// Mesmo sem o middleware, o handler não consulta dados sem identidade.
	// O vínculo vem da sessão, nunca de um filtro escolhido pelo navegador.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	active, ok := decodeListEmployeesRequest(w, r)
	if !ok {
		return
	}
	employees, err := h.service.List(r.Context(), principal.UserID, active)
	writeListEmployeesResponse(w, employees, err)
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
	input, ok := decodeEmployeeNameRequest(w, r)
	if !ok {
		return
	}

	// O contexto e o user ID seguem da requisição: o primeiro preserva prazo e
	// cancelamento; o segundo vem da sessão. O Service valida o nome, monta Employee
	// e só então chama o Repository.
	employee, err := h.service.Create(r.Context(), principal.UserID, input.Name)
	writeCreateEmployeeResponse(w, employee, err)
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	// A identidade deve existir mesmo se o handler for usado sem o middleware;
	// o cliente não escolhe o dono por JSON, query ou cabeçalho.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	employeeID, ok := decodeEmployeeID(w, r)
	if !ok {
		return
	}
	input, ok := decodeEmployeeNameRequest(w, r)
	if !ok {
		return
	}

	employee, err := h.service.Rename(r.Context(), principal.UserID, employeeID, input.Name)
	writeRenameEmployeeResponse(w, employee, err)
}

func (h *Handler) deactivate(w http.ResponseWriter, r *http.Request) {
	// A sessão é a única fonte do dono, mesmo se o handler for usado sem middleware.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	employeeID, ok := decodeEmployeeStateChangeRequest(w, r)
	if !ok {
		return
	}
	err := h.service.Deactivate(r.Context(), principal.UserID, employeeID)
	writeDeactivateEmployeeResponse(w, err)
}

func (h *Handler) reactivate(w http.ResponseWriter, r *http.Request) {
	// O principal confirmado pela sessão continua sendo a única fonte do dono.
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Autenticação necessária"})
		return
	}

	employeeID, ok := decodeEmployeeStateChangeRequest(w, r)
	if !ok {
		return
	}
	err := h.service.Reactivate(r.Context(), principal.UserID, employeeID)
	writeReactivateEmployeeResponse(w, err)
}
