// Abre e valida o pool PostgreSQL compartilhado pela API.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Open cria o pool compartilhado da API e confirma a conexão antes de servir requisições.
// Erros de conexão não incluem a URL porque ela contém a senha do banco.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL não configurada")
	}

	// Um pool é compartilhado pelas futuras consultas da API.
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, errors.New("configuração do banco inválida")
	}

	// Confirma que o banco responde antes de liberar o servidor HTTP.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("não foi possível conectar ao banco")
	}

	return pool, nil
}

// Executor permite executar as mesmas consultas no pool ou na transação de leitura
// que mantém todas as fontes de um relatório na mesma visão do banco.
type Executor interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
