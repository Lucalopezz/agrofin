package funcionarios

import (
	"encoding/json"
	"log"
	"net/http"
)

// RegisterRoutes mantém as rotas de funcionários privadas e recebe o middleware na composição da API.
func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	// A listagem também é privada: mesmo retornando uma lista vazia hoje, ela não
	// deve virar uma rota pública por acidente quando ganhar persistência.
	mux.Handle("GET /employees", requireSession(http.HandlerFunc(listEmployees)))
	// O middleware envolve o handler: primeiro autentica, depois o handler pode
	// ler o principal e chamar o Service de funcionários.
	mux.Handle("POST /employees", requireSession(http.HandlerFunc(handler.create)))
}

func listEmployees(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// A lista ainda não tem armazenamento; uma slice vazia produz [] em vez de null.
	if err := json.NewEncoder(w).Encode([]employeeResponse{}); err != nil {
		log.Printf("erro ao escrever funcionários: %v", err)
	}
}
