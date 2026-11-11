// Monta respostas HTTP e traduz erros de autenticação em mensagens públicas.
package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type registerResponse struct {
	Registered bool `json:"registered"`
}

type sessionResponse struct {
	Authenticated bool `json:"authenticated"`
}

type userResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LoginName string `json:"login_name"`
	Active    bool   `json:"active"`
}

type managerProfileHTTPResponse struct {
	Authenticated bool         `json:"authenticated"`
	User          userResponse `json:"user"`
}

type managerStatusResponse struct {
	Active bool `json:"active"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// writeError padroniza erros sem incluir detalhes internos de SQL ou criptografia.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func newManagerProfileResponse(profile ManagerProfile) managerProfileHTTPResponse {
	return managerProfileHTTPResponse{
		Authenticated: true,
		User:          userResponse(profile),
	}
}

// writeManagerProfileError mantém validação, ausência e falha de persistência
// separadas na borda HTTP sem expor mensagens do driver.
func writeManagerProfileError(w http.ResponseWriter, err error, internalMessage string) {
	switch {
	case isValidationError(err):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errManagerNotFound):
		writeError(w, http.StatusNotFound, "conta não encontrada")
	default:
		log.Print(internalMessage)
		writeError(w, http.StatusInternalServerError, internalMessage)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	// Respostas de autenticação não devem ser armazenadas em cache compartilhado:
	// isso evita reutilizar estado de sessão/resposta entre navegadores.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resposta de autenticação")
	}
}
