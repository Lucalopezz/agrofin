package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
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
	// A flag vem do ambiente para diferenciar HTTP local de HTTPS publicado; a
	// validação abaixo impede combinações que fariam o navegador ignorar o cookie.
	cookieSecure, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE")))
	if err != nil {
		return fmt.Errorf("SESSION_COOKIE_SECURE deve ser true ou false: %w", err)
	}
	publicOrigin := strings.TrimSpace(os.Getenv("PUBLIC_ORIGIN"))
	// A origem é uma allowlist única para o navegador. Não inferimos isso de
	// Host nem de X-Forwarded-* porque esses cabeçalhos dependem da borda confiável.
	if err := auth.ValidateSessionConfig(publicOrigin, cookieSecure); err != nil {
		return err
	}

	// Só inicia HTTP após confirmar o banco; o prazo limita a espera na inicialização.
	// Este contexto pertence à inicialização (ponto de entrada), não a uma
	// requisição. As requisições recebem seus próprios contextos via net/http.
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

	// O main é o ponto de composição: constrói dependências concretas e as conecta.
	// HTTP, regras de domínio e SQL continuam nos pacotes de suas funcionalidades.
	repo := funcionarios.NewRepository(pool)
	service := funcionarios.NewService(repo)
	handler := funcionarios.NewHandler(service)

	// Uma única instância de Service compartilha o limite local de trabalho Argon2id;
	// as tentativas de login e sessões continuam compartilhadas no PostgreSQL.
	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(authRepo)
	authHandler := auth.NewHandler(authService, auth.HandlerConfig{
		SetupToken:   os.Getenv("AUTH_SETUP_TOKEN"),
		CookieSecure: cookieSecure,
	})
	sessionMiddleware := auth.NewMiddleware(authService, cookieSecure)
	// Os pacotes declaram caminhos sem prefixo público; aqui o mesmo middleware de
	// sessão é entregue às rotas privadas de auth e funcionários.
	auth.RegisterRoutes(apiMux, authHandler, sessionMiddleware.RequireSession)
	funcionarios.RegisterRoutes(apiMux, handler, sessionMiddleware.RequireSession)

	mux := http.NewServeMux()
	mux.Handle(apiPrefix+"/", http.StripPrefix(apiPrefix, apiMux))

	log.Printf("api em http://localhost:8080%s", apiPrefix)
	server := &http.Server{
		Addr: ":8080",
		// A proteção de origem fica fora do mux para cobrir toda escrita, inclusive
		// login e logout, antes que qualquer handler execute.
		Handler: auth.RequireSameOrigin(publicOrigin, mux),
		// Timeouts e limite de cabeçalhos reduzem o impacto de clientes que enviam
		// requisições lentamente ou tentam ocupar conexões sem concluir o HTTP.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return server.ListenAndServe()
}

func apiPrefixFromEnv() (string, error) {
	apiPrefix := strings.TrimSpace(os.Getenv("API_PREFIX"))
	if apiPrefix == "" {
		apiPrefix = defaultAPIPrefix
	}
	// O prefixo é usado como caminho e também passado ao StripPrefix; rejeitar
	// caminhos ambíguos evita que a rota pública e a rota interna se desencontrem.
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

	// Esta rota indica que o processo HTTP responde; ela não consulta tabelas nem
	// comprova que cada fluxo de negócio está saudável.
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
