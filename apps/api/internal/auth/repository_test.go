package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("configure TEST_DATABASE_URL com um banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err, "configurar o banco de teste")
	defer pool.Close()
	unlockIntegrationDatabase := acquireAuthIntegrationDatabaseLock(t, ctx, pool)
	defer unlockIntegrationDatabase()

	const loginName = "teste-auth-sessoes"
	const password = "UmaSenhaForteComMaisDe15Caracteres"
	passwordHash, err := createPasswordHash(password, loginName)
	require.NoError(t, err, "gerar hash para o gerente de teste")
	var userID string
	_, err = pool.Exec(ctx, `
		DELETE FROM sessions
		WHERE user_id IN (SELECT id FROM users WHERE login_name = $1)
	`, loginName)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM users WHERE login_name = $1", loginName)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `
		INSERT INTO users (login_name, password_hash)
		VALUES ($1, $2)
		RETURNING id::text
	`, loginName, passwordHash).Scan(&userID)
	require.NoError(t, err, "criar gerente de teste")
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, err := pool.Exec(cleanupCtx, "DELETE FROM sessions WHERE user_id = $1", userID)
		assert.NoError(t, err, "limpar sessões de teste")
		_, err = pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", userID)
		assert.NoError(t, err, "limpar gerente de teste")
	}()

	repository := NewRepository(pool)
	service := NewService(repository)
	created, err := service.Login(ctx, loginName, password)
	require.NoError(t, err)
	tokenBytes, err := base64.RawURLEncoding.DecodeString(created.Token)
	require.NoError(t, err)
	tokenHash := sha256.Sum256(tokenBytes)
	var persistedTokenHash []byte
	err = pool.QueryRow(ctx, "SELECT token_hash FROM sessions WHERE token_hash = $1", tokenHash[:]).Scan(&persistedTokenHash)
	require.NoError(t, err)
	assert.Equal(t, tokenHash[:], persistedTokenHash)

	principal, err := service.AuthenticateSession(ctx, created.Token)
	require.NoError(t, err)
	assert.Equal(t, userID, principal.UserID)

	require.NoError(t, service.Logout(ctx, created.Token))
	_, err = service.AuthenticateSession(ctx, created.Token)
	assert.ErrorIs(t, err, ErrInvalidSession)
}

func acquireAuthIntegrationDatabaseLock(t *testing.T, ctx context.Context, pool *pgxpool.Pool) func() {
	t.Helper()
	connection, err := pool.Acquire(ctx)
	require.NoError(t, err, "reservar conexão para serializar integrações")
	const lockKey int64 = 2026103001
	_, err = connection.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey)
	require.NoError(t, err, "serializar testes que usam o gerente único")
	return func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := connection.Exec(cleanupCtx, "SELECT pg_advisory_unlock($1)", lockKey)
		assert.NoError(t, err, "liberar trava de integração")
		connection.Release()
	}
}
