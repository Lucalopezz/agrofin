package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/migrations"
	"github.com/pressly/goose/v3"

	// Registra "pgx" no database/sql; o goose recebe um *sql.DB.
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	// Só estes comandos podem consultar ou alterar o esquema.
	if len(args) != 1 || (args[0] != "up" && args[0] != "status" && args[0] != "down") {
		return errors.New("uso: go run ./cmd/migrate [up|status|down]")
	}

	// A URL vem do ambiente para manter a senha fora do código.
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL não configurada")
	}

	// Limita a conexão e a execução das migrações a dois minutos.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Cria o pool exigido pelo goose usando o driver PostgreSQL registrado acima.
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return errors.New("configuração do banco inválida")
	}
	defer db.Close()

	// sql.Open não confirma a conexão; Ping testa o acesso antes de executar SQL.
	if err := db.PingContext(ctx); err != nil {
		return errors.New("não foi possível conectar ao banco")
	}

	// Entrega ao goose o sistema de arquivos incorporado em migrations/embed.go.
	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configurar migrações: %w", err)
	}

	// O goose usa goose_db_version para decidir quais versões executar no comando up.
	if err := goose.RunContext(ctx, args[0], db, "."); err != nil {
		return fmt.Errorf("executar migração %s: %w", args[0], err)
	}
	return nil
}
