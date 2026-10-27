package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/funcionarios"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/postgres"
)

type healthResponse struct {
	Status string `json:"status"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Só inicia HTTP após confirmar o banco; o prazo limita a espera na inicialização.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()

	// O main compõe as rotas; os handlers de negócio ficam nos pacotes das funcionalidades.
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", health)

	// O pacote de funcionários continua dono das próprias rotas; sua consulta ao
	// banco será ligada quando o cadastro deixar de ser uma listagem vazia.
	funcionarios.RegisterRoutes(mux)

	log.Println("api em http://localhost:8080")
	return http.ListenAndServe(":8080", mux)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// A rota de saúde fica independente de banco e outras funcionalidades para indicar que o servidor responde.
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
