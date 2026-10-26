package funcionarios

import (
	"encoding/json"
	"log"
	"net/http"
)

type employeeResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RegisterRoutes mantém as rotas de funcionários junto da funcionalidade,
// enquanto o main apenas compõe as partes da API.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/employees", listEmployees)
}

func listEmployees(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// A lista ainda não tem armazenamento; uma slice vazia produz [] em vez de null.
	if err := json.NewEncoder(w).Encode([]employeeResponse{}); err != nil {
		log.Printf("erro ao escrever funcionários: %v", err)
	}
}
