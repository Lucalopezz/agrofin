package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

	accountID, err := developmentAccountID(ctx, pool, os.Getenv("APP_ENV"), os.Getenv("DEV_ACCOUNT_ID"))
	if err != nil {
		return err
	}

	// O main compõe as rotas; os handlers de negócio ficam nos pacotes das funcionalidades.
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", health)

	// O main monta as dependências; HTTP, domínio e SQL continuam na feature.
	repo := funcionarios.NewRepository(pool)
	service := funcionarios.NewService(repo)
	handler := funcionarios.NewHandler(service, accountID)
	funcionarios.RegisterRoutes(mux, handler)

	log.Println("api em http://localhost:8080")
	return http.ListenAndServe(":8080", mux)
}

func developmentAccountID(ctx context.Context, pool *pgxpool.Pool, appEnv, accountID string) (string, error) {
	if accountID == "" {
		return "", nil
	}
	// Este caminho temporário não autentica o cliente. Exigir o modo explícito
	// evita habilitar a escrita por configuração acidental em outro ambiente.
	if appEnv != "development" {
		return "", errors.New("DEV_ACCOUNT_ID só pode ser usado com APP_ENV=development")
	}
	var id string
	if err := pool.QueryRow(ctx, "SELECT id::text FROM accounts WHERE id = $1::uuid", accountID).Scan(&id); err != nil {
		return "", errors.New("DEV_ACCOUNT_ID deve ser o UUID de uma conta existente e o banco deve estar acessível")
	}
	return id, nil
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// A rota de saúde fica independente de banco e outras funcionalidades para indicar que o servidor responde.
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
