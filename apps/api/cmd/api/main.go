// Compõe os serviços, registra as rotas protegidas e inicia o servidor HTTP da API.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	"github.com/lucalopezz/agrofin/apps/api/internal/funcionarios"
	"github.com/lucalopezz/agrofin/apps/api/internal/health"
	"github.com/lucalopezz/agrofin/apps/api/internal/pagamentos"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/postgres"
)

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
	financialKeyring, err := financialKeyringFromEnv()
	if err != nil {
		return err
	}
	rateCipher, err := colheitas.NewRateCipherWithKeyring(financialKeyring)
	if err != nil {
		return err
	}
	dailyWorkCipher, err := diarias.NewCipher(financialKeyring)
	if err != nil {
		return err
	}
	paymentCipher, err := pagamentos.NewCipher(financialKeyring)
	if err != nil {
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
	health.RegisterRoutes(apiMux)

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
		CookieSecure: cookieSecure,
	})
	sessionMiddleware := auth.NewMiddleware(authService, cookieSecure)
	// Os pacotes declaram caminhos sem prefixo público; aqui o mesmo middleware de
	// sessão é entregue às rotas privadas de auth, funcionários e tarifas.
	auth.RegisterRoutes(apiMux, authHandler, sessionMiddleware.RequireSession)
	funcionarios.RegisterRoutes(apiMux, handler, sessionMiddleware.RequireSession)

	rateRepository := colheitas.NewRepository(pool)
	rateService := colheitas.NewService(rateRepository, rateCipher)
	harvestService := colheitas.NewHarvestService(rateRepository, rateCipher)
	rateHandler := colheitas.NewHandler(rateService, harvestService)
	colheitas.RegisterRoutes(apiMux, rateHandler, sessionMiddleware.RequireSession)

	dailyWorkRepository := diarias.NewRepository(pool)
	dailyWorkService := diarias.NewService(dailyWorkRepository, dailyWorkCipher)
	dailyWorkHandler := diarias.NewHandler(dailyWorkService)
	diarias.RegisterRoutes(apiMux, dailyWorkHandler, sessionMiddleware.RequireSession)

	// Um único PayrollService coordena a criação e o resumo, reutilizando as projeções
	// de colheitas e diárias sem transferir para ele a persistência desses domínios.
	paymentRepository := pagamentos.NewRepository(pool)
	payrollService := pagamentos.NewPayrollService(harvestService, dailyWorkService, paymentRepository, paymentCipher)
	payrollHandler := pagamentos.NewHandler(payrollService)
	pagamentos.RegisterRoutes(apiMux, payrollHandler, sessionMiddleware.RequireSession)

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
