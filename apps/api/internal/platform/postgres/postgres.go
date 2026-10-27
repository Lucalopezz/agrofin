package postgres

import (
	"context"
	"errors"

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
