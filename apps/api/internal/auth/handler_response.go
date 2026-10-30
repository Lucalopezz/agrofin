package auth

import (
	"encoding/json"
	"log"
	"net/http"
)

type registerResponse struct {
	Registered bool `json:"registered"`
}

type sessionResponse struct {
	Authenticated bool `json:"authenticated"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// writeError padroniza erros sem incluir detalhes internos de SQL ou criptografia.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
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
