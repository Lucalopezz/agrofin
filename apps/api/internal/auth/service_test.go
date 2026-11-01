package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

type managerCreatorFunc func(context.Context, Manager) error

type authRepositoryStub struct {
	manager           Manager
	loginName         string
	findErr           error
	profile           ManagerProfile
	findProfileErr    error
	updatedManager    Manager
	updateProfileErr  error
	statusManager     Manager
	setStatusErr      error
	session           Session
	createSessionErr  error
	resolvedPrincipal Principal
	resolveSessionErr error
	resolvedTokenHash []byte
	revokedTokenHash  []byte
	revokeSessionErr  error
}

func (r *authRepositoryStub) CreateManager(context.Context, Manager) error { return nil }

func (r *authRepositoryStub) FindManager(_ context.Context, loginName string) (Manager, error) {
	r.loginName = loginName
	if r.findErr != nil {
		return Manager{}, r.findErr
	}
	return r.manager, nil
}

func (r *authRepositoryStub) FindManagerProfile(context.Context, string) (ManagerProfile, error) {
	return r.profile, r.findProfileErr
}

func (r *authRepositoryStub) UpdateManagerName(_ context.Context, manager Manager) (ManagerProfile, error) {
	r.updatedManager = manager
	return r.profile, r.updateProfileErr
}

func (r *authRepositoryStub) SetManagerActive(_ context.Context, manager Manager) error {
	r.statusManager = manager
	return r.setStatusErr
}

func (r *authRepositoryStub) CreateSession(_ context.Context, session Session) error {
	r.session = session
	return r.createSessionErr
}

func (r *authRepositoryStub) ResolveSession(_ context.Context, tokenHash []byte) (Principal, error) {
	r.resolvedTokenHash = append([]byte(nil), tokenHash...)
	return r.resolvedPrincipal, r.resolveSessionErr
}

func (r *authRepositoryStub) RevokeSession(_ context.Context, tokenHash []byte) error {
	r.revokedTokenHash = append([]byte(nil), tokenHash...)
	return r.revokeSessionErr
}

func (f managerCreatorFunc) CreateManager(ctx context.Context, manager Manager) error {
	return f(ctx, manager)
}

func (managerCreatorFunc) FindManager(context.Context, string) (Manager, error) {
	return Manager{}, errManagerNotFound
}

func (managerCreatorFunc) FindManagerProfile(context.Context, string) (ManagerProfile, error) {
	return ManagerProfile{}, errManagerNotFound
}

func (managerCreatorFunc) UpdateManagerName(context.Context, Manager) (ManagerProfile, error) {
	return ManagerProfile{}, errManagerNotFound
}

func (managerCreatorFunc) SetManagerActive(context.Context, Manager) error { return nil }

func (managerCreatorFunc) CreateSession(context.Context, Session) error { return nil }

func (managerCreatorFunc) ResolveSession(context.Context, []byte) (Principal, error) {
	return Principal{}, ErrInvalidSession
}

func (managerCreatorFunc) RevokeSession(context.Context, []byte) error { return nil }

func TestServiceRegisterManager(t *testing.T) {
	t.Run("normaliza credenciais e envia a entidade com hash", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const password = "UmaSenhaForteComMaisDe15Caracteres"
		called := false
		repo := managerCreatorFunc(func(gotCtx context.Context, manager Manager) error {
			called = true
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "Gerente da Produção", manager.Name)
			assert.Equal(t, "gerente_1", manager.LoginName)
			assert.NotEqual(t, password, manager.PasswordHash)
			assertPasswordHashMatches(t, manager.PasswordHash, password)
			return nil
		})

		err := NewService(repo).RegisterManager(ctx, "  Gerente da Produção  ", "  Gerente_1  ", password)
		require.NoError(t, err)
		assert.True(t, called, "o repository deve receber a entidade válida")
	})

	t.Run("credenciais inválidas não chegam ao repository", func(t *testing.T) {
		tests := []struct {
			name        string
			managerName string
			loginName   string
			password    string
		}{
			{name: "nome vazio", managerName: "   ", loginName: "gerente", password: "UmaSenhaForteComMaisDe15Caracteres"},
			{name: "login inválido", managerName: "Gerente", loginName: "x", password: "UmaSenhaForteComMaisDe15Caracteres"},
			{name: "senha fraca", managerName: "Gerente", loginName: "gerente", password: "passwordpassword"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				called := false
				repo := managerCreatorFunc(func(context.Context, Manager) error {
					called = true
					return nil
				})

				err := NewService(repo).RegisterManager(context.Background(), tt.managerName, tt.loginName, tt.password)
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

		err := NewService(repo).RegisterManager(context.Background(), "Gerente", "gerente", "UmaSenhaForteComMaisDe15Caracteres")
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestServiceManagerProfile(t *testing.T) {
	t.Run("atualiza somente o nome normalizado", func(t *testing.T) {
		repo := &authRepositoryStub{profile: ManagerProfile{ID: "manager-id", Name: "Novo Nome", LoginName: "login-fixo", Active: true}}
		profile, err := NewService(repo).UpdateManagerName(context.Background(), "manager-id", "  Novo Nome  ")
		require.NoError(t, err)
		assert.Equal(t, "manager-id", repo.updatedManager.ID)
		assert.Equal(t, "Novo Nome", repo.updatedManager.Name)
		assert.Empty(t, repo.updatedManager.LoginName, "login_name não participa da entidade de atualização")
		assert.Equal(t, "login-fixo", profile.LoginName)
	})

	t.Run("nome vazio não chega ao repository", func(t *testing.T) {
		repo := &authRepositoryStub{}
		_, err := NewService(repo).UpdateManagerName(context.Background(), "manager-id", "  ")
		assert.True(t, isValidationError(err))
		assert.Empty(t, repo.updatedManager.ID)
	})
}

func TestServiceSetManagerActive(t *testing.T) {
	repo := &authRepositoryStub{}
	service := NewService(repo)

	require.NoError(t, service.SetManagerActive(context.Background(), "manager-id", false))
	assert.Equal(t, "manager-id", repo.statusManager.ID)
	assert.False(t, repo.statusManager.Active)

	err := service.SetManagerActive(context.Background(), "manager-id", true)
	assert.ErrorIs(t, err, ErrActivationRequiresOperator)
	assert.False(t, repo.statusManager.Active, "reativação não deve chegar ao Repository")
}

func TestServiceLogin(t *testing.T) {
	const password = "UmaSenhaForteComMaisDe15Caracteres"
	passwordHash, err := createPasswordHash(password, "gerente")
	require.NoError(t, err)

	t.Run("credenciais válidas criam sessão e persistem somente o hash do token", func(t *testing.T) {
		repo := &authRepositoryStub{manager: Manager{ID: "manager-id", LoginName: "gerente", PasswordHash: passwordHash, Active: true}}
		ctx := context.Background()
		created, err := NewService(repo).Login(ctx, " GERENTE ", password)
		require.NoError(t, err)

		tokenBytes, err := base64.RawURLEncoding.DecodeString(created.Token)
		require.NoError(t, err)
		assert.Len(t, tokenBytes, 32)
		assert.True(t, created.ExpiresAt.After(time.Now()))
		assert.Equal(t, "gerente", repo.loginName)
		assert.Equal(t, "manager-id", repo.session.UserID)
		wantTokenHash := sha256.Sum256(tokenBytes)
		assert.Equal(t, wantTokenHash[:], repo.session.TokenHash)
		assert.True(t, repo.manager.Active)
	})

	t.Run("credenciais erradas e gerente ausente retornam o mesmo erro", func(t *testing.T) {
		tests := []struct {
			name    string
			manager Manager
			findErr error
		}{
			{name: "senha errada", manager: Manager{ID: "manager-id", LoginName: "gerente", PasswordHash: passwordHash, Active: true}},
			{name: "login inexistente", findErr: errManagerNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &authRepositoryStub{manager: tt.manager, findErr: tt.findErr}
				_, err := NewService(repo).Login(context.Background(), "gerente", "OutraSenhaLongaESuficiente")
				assert.ErrorIs(t, err, ErrInvalidCredentials)
				assert.Empty(t, repo.session.UserID)
			})
		}
	})

	t.Run("usuário inativo não autentica", func(t *testing.T) {
		repo := &authRepositoryStub{manager: Manager{ID: "manager-id", LoginName: "gerente", PasswordHash: passwordHash, Active: false}}
		_, err := NewService(repo).Login(context.Background(), "gerente", password)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, repo.session.UserID)
	})
}

func TestServiceSessionTokenMethods(t *testing.T) {
	tokenBytes := []byte("01234567890123456789012345678901")
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	wantTokenHash := sha256.Sum256(tokenBytes)
	repo := &authRepositoryStub{resolvedPrincipal: Principal{UserID: "manager-id"}}
	service := NewService(repo)

	principal, err := service.AuthenticateSession(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, Principal{UserID: "manager-id"}, principal)
	assert.Equal(t, wantTokenHash[:], repo.resolvedTokenHash)

	assert.ErrorIs(t, func() error {
		_, err := service.AuthenticateSession(context.Background(), "malformed")
		return err
	}(), ErrInvalidSession)

	require.NoError(t, service.Logout(context.Background(), token))
	assert.Equal(t, wantTokenHash[:], repo.revokedTokenHash)
	require.NoError(t, service.Logout(context.Background(), "malformed"))
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
