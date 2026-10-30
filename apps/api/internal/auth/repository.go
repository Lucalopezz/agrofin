package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persiste os dados de identidade usando o pool compartilhado da API.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository recebe um pool já aberto; o ciclo de vida continua com a aplicação.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateManager persiste a entidade validada e preparada pelo Service.
// O índice único da instalação protege contra cadastros concorrentes.
func (r *Repository) CreateManager(ctx context.Context, manager Manager) error {
	// O Repository recebe a entidade já normalizada e com hash; ele não decide
	// regras de senha nem transforma dados de entrada.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (login_name, password_hash, active)
		VALUES ($1, $2, $3)
	`, manager.LoginName, manager.PasswordHash, manager.Active)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			// A unicidade é garantida pelo banco, inclusive se dois cadastros
			// iniciais chegarem juntos. O SQLSTATE vira um erro do domínio.
			return ErrManagerAlreadyRegistered
		}
		return errors.New("não foi possível cadastrar o gerente")
	}
	return nil
}

// FindManager lê a credencial persistida para que o Service faça a verificação da senha.
func (r *Repository) FindManager(ctx context.Context, loginName string) (Manager, error) {
	var manager Manager
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, login_name, password_hash, active
		FROM users
		WHERE login_name = $1
	`, loginName).Scan(&manager.ID, &manager.LoginName, &manager.PasswordHash, &manager.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		// O Service converte este caso em ErrInvalidCredentials, igualando a
		// resposta pública de login inexistente e senha incorreta.
		return Manager{}, errManagerNotFound
	}
	if err != nil {
		return Manager{}, errors.New("não foi possível consultar as credenciais")
	}
	return manager, nil
}

// CreateSession grava somente o hash do token; o original é devolvido uma vez ao Handler.
func (r *Repository) CreateSession(ctx context.Context, session Session) error {
	// A limpeza oportunista evita manter indefinidamente linhas vencidas sem
	// precisar de um job separado. Ela não substitui a validação abaixo no login.
	if _, err := r.pool.Exec(ctx, `
		DELETE FROM sessions
		WHERE expires_at <= now() OR revoked_at < now() - interval '30 days'
	`); err != nil {
		return errors.New("não foi possível limpar sessões expiradas")
	}
	// A sessão já chega pronta do Service. O banco guarda o hash do token e associa
	// o user_id confirmado pela consulta de credenciais.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, session.UserID, session.TokenHash, session.ExpiresAt)
	if err != nil {
		return errors.New("não foi possível criar a sessão")
	}
	return nil
}

// ResolveSession atualiza a atividade e retorna a identidade somente para uma sessão ativa e não ociosa.
func (r *Repository) ResolveSession(ctx context.Context, tokenHash []byte) (Principal, error) {
	var principal Principal
	// UPDATE ... RETURNING valida e registra atividade numa única instrução. Só
	// retorna o principal se a sessão ainda estiver ativa, dentro dos dois prazos,
	// sem revogação e ligada a um gerente ativo.
	err := r.pool.QueryRow(ctx, `
		UPDATE sessions AS s
		SET last_seen_at = now()
	FROM users AS u
	WHERE s.user_id = u.id
	  AND s.token_hash = $1
	  AND s.revoked_at IS NULL
	  AND s.expires_at > now()
	  AND s.last_seen_at > now() - interval '7 days'
	  AND u.active
	RETURNING s.user_id::text
	`, tokenHash).Scan(&principal.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrInvalidSession
	}
	if err != nil {
		return Principal{}, errors.New("não foi possível validar a sessão")
	}
	return principal, nil
}

// RevokeSession invalida no servidor o token que o navegador está prestes a descartar.
func (r *Repository) RevokeSession(ctx context.Context, tokenHash []byte) error {
	// Em vez de apagar a linha, marcamos revoked_at para que o estado de revogação
	// seja explícito e para permitir diagnóstico/limpeza posterior.
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return errors.New("não foi possível encerrar a sessão")
	}
	return nil
}
