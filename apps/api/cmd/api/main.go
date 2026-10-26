package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/funcionarios"
)

type healthResponse struct {
	Status string `json:"status"`
}

func main() {
	// O main compõe as rotas; os handlers de negócio ficam nos pacotes das funcionalidades.
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", health)

	// As rotas de funcionários são registradas no mux pelo próprio pacote de funcionários.
	// e chamados aqui na main, para passar o mux como argumento.
	funcionarios.RegisterRoutes(mux)

	log.Println("api em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// A rota de saúde fica independente de banco e outras funcionalidades para indicar que o servidor responde.
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
