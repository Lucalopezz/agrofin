package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

type managerCreatorFunc func(context.Context, Manager) error

func (f managerCreatorFunc) CreateManager(ctx context.Context, manager Manager) error {
	return f(ctx, manager)
}

func TestServiceRegisterManager(t *testing.T) {
	t.Run("normaliza credenciais e envia a entidade com hash", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const password = "UmaSenhaForteComMaisDe15Caracteres"
		called := false
		repo := managerCreatorFunc(func(gotCtx context.Context, manager Manager) error {
			called = true
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "gerente_1", manager.LoginName)
			assert.NotEqual(t, password, manager.PasswordHash)
			assertPasswordHashMatches(t, manager.PasswordHash, password)
			return nil
		})

		err := NewService(repo).RegisterManager(ctx, "  Gerente_1  ", password)
		require.NoError(t, err)
		assert.True(t, called, "o repository deve receber a entidade válida")
	})

	t.Run("credenciais inválidas não chegam ao repository", func(t *testing.T) {
		tests := []struct {
			name      string
			loginName string
			password  string
		}{
			{name: "login inválido", loginName: "x", password: "UmaSenhaForteComMaisDe15Caracteres"},
			{name: "senha fraca", loginName: "gerente", password: "passwordpassword"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				called := false
				repo := managerCreatorFunc(func(context.Context, Manager) error {
					called = true
					return nil
				})

				err := NewService(repo).RegisterManager(context.Background(), tt.loginName, tt.password)
				assert.True(t, isValidationError(err))
				assert.False(t, called)
			})
		}
	})

	t.Run("preserva erro do repository", func(t *testing.T) {
		wantErr := errors.New("falha de persistência")
		repo := managerCreatorFunc(func(context.Context, Manager) error {
			return wantErr
		})

		err := NewService(repo).RegisterManager(context.Background(), "gerente", "UmaSenhaForteComMaisDe15Caracteres")
		assert.ErrorIs(t, err, wantErr)
	})
}

func assertPasswordHashMatches(t *testing.T, encodedHash, password string) {
	t.Helper()
	parts := strings.Split(encodedHash, "$")
	require.Len(t, parts, 6)
	assert.Equal(t, "argon2id", parts[1])
	assert.Equal(t, "v=19", parts[2])
	assert.Equal(t, "m=19456,t=2,p=1", parts[3])

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	require.NoError(t, err)
	digest, err := base64.RawStdEncoding.DecodeString(parts[5])
	require.NoError(t, err)

	// Recalcular o digest garante que o valor persistido corresponde à senha e aos parâmetros declarados no formato PHC.
	expected := argon2.IDKey([]byte(password), salt, passwordTime, passwordMemory, passwordThreads, passwordKeyBytes)
	assert.Equal(t, expected, digest)
}
