package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

func main() {
	// Configura o roteador HTTP
	mux := http.NewServeMux()

	// Define a rota para o endpoint de saúde
	mux.HandleFunc("GET /api/v1/health", health)

	log.Println("api em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func health(w http.ResponseWriter, _ *http.Request) {
	// Define o cabeçalho de resposta como JSON
	w.Header().Set("Content-Type", "application/json")

	// Cria a resposta de saúde
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
