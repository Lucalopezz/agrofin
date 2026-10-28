package funcionarios

import (
	"encoding/json"
	"log"
	"net/http"
)

// RegisterRoutes mantém as rotas de funcionários junto da funcionalidade,
// enquanto o main apenas compõe as partes da API.
func RegisterRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("GET /api/v1/employees", listEmployees)
	mux.HandleFunc("POST /api/v1/employees", handler.create)
}

func listEmployees(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// A lista ainda não tem armazenamento; uma slice vazia produz [] em vez de null.
	if err := json.NewEncoder(w).Encode([]employeeResponse{}); err != nil {
		log.Printf("erro ao escrever funcionários: %v", err)
	}
}
