package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
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

	// O main monta as dependências; HTTP, domínio e SQL continuam na feature.
	repo := funcionarios.NewRepository(pool)
	service := funcionarios.NewService(repo)
	handler := funcionarios.NewHandler(service)
	funcionarios.RegisterRoutes(mux, handler)

	// O token de configuração fica somente no ambiente do servidor e é usado uma vez para criar o gerente.
	authHandler := auth.NewHandler(auth.NewRepository(pool), os.Getenv("AUTH_SETUP_TOKEN"))
	auth.RegisterRoutes(mux, authHandler)

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
