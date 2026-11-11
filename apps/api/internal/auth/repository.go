// Persiste contas, credenciais e sessões no PostgreSQL.
package auth

import (
	"context"
	"errors"
	"time"

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
// A unicidade do login protege contra cadastros concorrentes com o mesmo acesso.
func (r *Repository) CreateManager(ctx context.Context, manager Manager) error {
	// O Repository recebe a entidade já normalizada e com hash; ele não decide
	// regras de senha nem transforma dados de entrada.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (name, login_name, password_hash, active)
		VALUES ($1, $2, $3, $4)
	`, manager.Name, manager.LoginName, manager.PasswordHash, manager.Active)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" && databaseError.ConstraintName == "users_login_name_key" {
			// Só o conflito do login vira 409; outras restrições não significam
			// que o nome está ocupado. O banco decide a disputa entre cadastros.
			return ErrLoginNameTaken
		}
		return errors.New("não foi possível cadastrar o gerente")
	}
	return nil
}

// FindManager lê a credencial persistida para que o Service faça a verificação da senha.
func (r *Repository) FindManager(ctx context.Context, loginName string) (Manager, error) {
	var manager Manager
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, login_name, password_hash, active
		FROM users
		WHERE login_name = $1
	`, loginName).Scan(&manager.ID, &manager.Name, &manager.LoginName, &manager.PasswordHash, &manager.Active)
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

// FindManagerByID carrega a credencial da conta identificada pelo principal da sessão.
func (r *Repository) FindManagerByID(ctx context.Context, userID string) (Manager, error) {
	var manager Manager
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, login_name, password_hash, active
		FROM users
		WHERE id = $1
	`, userID).Scan(&manager.ID, &manager.Name, &manager.LoginName, &manager.PasswordHash, &manager.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return Manager{}, errManagerNotFound
	}
	if err != nil {
		return Manager{}, errors.New("não foi possível consultar a credencial do gerente")
	}
	return manager, nil
}

// FindManagerProfile lê somente dados públicos para /auth/me, sem carregar o hash da senha.
func (r *Repository) FindManagerProfile(ctx context.Context, userID string) (ManagerProfile, error) {
	var profile ManagerProfile
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, login_name, active
		FROM users
		WHERE id = $1
	`, userID).Scan(&profile.ID, &profile.Name, &profile.LoginName, &profile.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ManagerProfile{}, errManagerNotFound
	}
	if err != nil {
		return ManagerProfile{}, errors.New("não foi possível consultar o perfil do gerente")
	}
	return profile, nil
}

// UpdateManagerName altera somente name e devolve a projeção pública na mesma consulta.
func (r *Repository) UpdateManagerName(ctx context.Context, manager Manager) (ManagerProfile, error) {
	var profile ManagerProfile
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET name = $1
		WHERE id = $2
		RETURNING id::text, name, login_name, active
	`, manager.Name, manager.ID).Scan(&profile.ID, &profile.Name, &profile.LoginName, &profile.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ManagerProfile{}, errManagerNotFound
	}
	if err != nil {
		return ManagerProfile{}, errors.New("não foi possível atualizar o nome do gerente")
	}
	return profile, nil
}

// UpdateManagerPassword atualiza a senha e revoga as sessões na mesma transação.
// Assim, um cookie emitido antes da troca nunca continua válido com a senha nova.
func (r *Repository) UpdateManagerPassword(ctx context.Context, change ManagerPasswordChange) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.New("não foi possível iniciar a troca de senha")
	}
	defer func() {
		// O rollback precisa terminar mesmo se o contexto HTTP já tiver sido cancelado.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	result, err := tx.Exec(ctx, `
		UPDATE users
		SET password_hash = $1
		WHERE id = $2 AND password_hash = $3 AND active = true
	`, change.NewPasswordHash, change.UserID, change.PreviousPasswordHash)
	if err != nil {
		return errors.New("não foi possível atualizar a senha")
	}
	if result.RowsAffected() != 1 {
		// Pode significar que outra troca venceu a corrida ou que a conta foi
		// desativada após o middleware validar esta requisição.
		return ErrPasswordChangeConflict
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, change.UserID); err != nil {
		return errors.New("não foi possível revogar as sessões após a troca de senha")
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("não foi possível finalizar a troca de senha")
	}
	return nil
}

// SetManagerActive grava o estado e revoga sessões na mesma transação para que
// tokens antigos não voltem a funcionar após uma reativação manual.
func (r *Repository) SetManagerActive(ctx context.Context, manager Manager) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.New("não foi possível iniciar a atualização da conta")
	}
	defer func() {
		// O rollback precisa terminar mesmo se o contexto HTTP já tiver sido cancelado.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		// cria um contexto que preserva os valores do contexto original, mas não herda seu cancelamento nem seu prazo de expiração.
		// assim, mesmo que o usuário feche o navegador, o backend ainda pode tentar encerrar a transação corretamente.
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE users
		SET active = $1
		WHERE id = $2
		RETURNING id::text
	`, manager.Active, manager.ID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errManagerNotFound
	}
	if err != nil {
		return errors.New("não foi possível atualizar o estado da conta")
	}

	if !manager.Active {
		if _, err := tx.Exec(ctx, `
			UPDATE sessions
			SET revoked_at = now()
			WHERE user_id = $1 AND revoked_at IS NULL
		`, userID); err != nil {
			return errors.New("não foi possível revogar as sessões do gerente")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("não foi possível finalizar a atualização da conta")
	}
	return nil
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
