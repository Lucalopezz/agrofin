// Testa a persistência PostgreSQL de credenciais e sessões.
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
		INSERT INTO users (name, login_name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, "Gerente de teste", loginName, passwordHash).Scan(&userID)
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
	otherSession, err := service.Login(ctx, loginName, password)
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

	profile, err := service.GetManagerProfile(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "Gerente de teste", profile.Name)
	assert.Equal(t, loginName, profile.LoginName)
	updatedProfile, err := service.UpdateManagerName(ctx, userID, "  Nome atualizado  ")
	require.NoError(t, err)
	assert.Equal(t, "Nome atualizado", updatedProfile.Name)
	assert.Equal(t, loginName, updatedProfile.LoginName, "alterar o nome não deve alterar o login")

	const newPassword = "OutraSenhaForteComMaisDe15Caracteres"
	require.NoError(t, service.ChangePassword(ctx, userID, password, newPassword))
	_, err = service.AuthenticateSession(ctx, created.Token)
	assert.ErrorIs(t, err, ErrInvalidSession, "trocar a senha deve revogar a sessão usada")
	_, err = service.AuthenticateSession(ctx, otherSession.Token)
	assert.ErrorIs(t, err, ErrInvalidSession, "a troca também revoga sessões abertas em outros navegadores")
	_, err = service.Login(ctx, loginName, password)
	assert.ErrorIs(t, err, ErrInvalidCredentials, "a senha anterior deixa de autenticar")
	newSession, err := service.Login(ctx, loginName, newPassword)
	require.NoError(t, err, "a nova senha deve autenticar")

	require.NoError(t, service.SetManagerActive(ctx, userID, false))
	var revoked bool
	err = pool.QueryRow(ctx, "SELECT revoked_at IS NOT NULL FROM sessions WHERE token_hash = $1", tokenHash[:]).Scan(&revoked)
	require.NoError(t, err)
	assert.True(t, revoked, "desativar a conta deve invalidar todas as sessões existentes")
	_, err = service.AuthenticateSession(ctx, created.Token)
	assert.ErrorIs(t, err, ErrInvalidSession)
	_, err = service.AuthenticateSession(ctx, newSession.Token)
	assert.ErrorIs(t, err, ErrInvalidSession, "desativar a conta também revoga as sessões criadas depois da troca")
	_, err = service.Login(ctx, loginName, password)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = service.Login(ctx, loginName, newPassword)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
