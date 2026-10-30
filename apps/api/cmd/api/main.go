package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/lucalopezz/agrofin/apps/api/internal/funcionarios"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/postgres"
)

type healthResponse struct {
	Status string `json:"status"`
}

const defaultAPIPrefix = "/api/v1"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	apiPrefix, err := apiPrefixFromEnv()
	if err != nil {
		return err
	}

	// Só inicia HTTP após confirmar o banco; o prazo limita a espera na inicialização.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()

	// Os pacotes registram caminhos locais; somente este ponto aplica o prefixo público comum.
	apiMux := http.NewServeMux()

	apiMux.HandleFunc("GET /health", health)

	// O main monta as dependências; HTTP, domínio e SQL continuam na feature.
	repo := funcionarios.NewRepository(pool)
	service := funcionarios.NewService(repo)
	handler := funcionarios.NewHandler(service)
	funcionarios.RegisterRoutes(apiMux, handler)

	// O token de configuração fica somente no ambiente do servidor e é usado uma vez para criar o gerente.
	authHandler := auth.NewHandler(auth.NewRepository(pool), os.Getenv("AUTH_SETUP_TOKEN"))
	auth.RegisterRoutes(apiMux, authHandler)

	mux := http.NewServeMux()
	mux.Handle(apiPrefix+"/", http.StripPrefix(apiPrefix, apiMux))

	log.Printf("api em http://localhost:8080%s", apiPrefix)
	return http.ListenAndServe(":8080", mux)
}

func apiPrefixFromEnv() (string, error) {
	apiPrefix := strings.TrimSpace(os.Getenv("API_PREFIX"))
	if apiPrefix == "" {
		apiPrefix = defaultAPIPrefix
	}
	if !strings.HasPrefix(apiPrefix, "/") || strings.Contains(apiPrefix, "//") || strings.ContainsAny(apiPrefix, "?# \t\r\n") {
		return "", fmt.Errorf("API_PREFIX inválido: use um caminho absoluto como %s", defaultAPIPrefix)
	}

	apiPrefix = strings.TrimSuffix(apiPrefix, "/")
	if apiPrefix == "" || path.Clean(apiPrefix) != apiPrefix {
		return "", fmt.Errorf("API_PREFIX inválido: use um caminho absoluto como %s", defaultAPIPrefix)
	}
	return apiPrefix, nil
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// A rota de saúde fica independente de banco e outras funcionalidades para indicar que o servidor responde.
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
