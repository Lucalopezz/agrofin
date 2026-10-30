package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type managerRegistrarFunc func(context.Context, string, string) error

func (f managerRegistrarFunc) RegisterManager(ctx context.Context, loginName, password string) error {
	return f(ctx, loginName, password)
}

func (f managerRegistrarFunc) Login(ctx context.Context, loginName, password string) (createdSession, error) {
	err := f(ctx, loginName, password)
	return createdSession{Token: "test-session-token", ExpiresAt: time.Now().Add(time.Hour)}, err
}

func (f managerRegistrarFunc) Logout(ctx context.Context, _ string) error {
	return f(ctx, "", "")
}

func TestRegisterHandler(t *testing.T) {
	const expectedToken = "token-de-configuracao"
	const validBody = `{"login_name":"Gerente_1","password":"UmaSenhaForteComMaisDe15Caracteres"}`
	tests := []struct {
		name             string
		body             string
		contentType      string
		token            string
		disableSetup     bool
		serviceErr       error
		expectedStatus   int
		expectedCalls    int
		expectedLogin    string
		expectedPassword string
	}{
		{name: "cadastro concluído", body: validBody, contentType: "application/json; charset=utf-8", token: expectedToken, expectedStatus: http.StatusCreated, expectedCalls: 1, expectedLogin: "Gerente_1", expectedPassword: "UmaSenhaForteComMaisDe15Caracteres"},
		{name: "token de configuração ausente no servidor", body: validBody, contentType: "application/json", token: expectedToken, disableSetup: true, expectedStatus: http.StatusServiceUnavailable},
		{name: "token não enviado", body: validBody, contentType: "application/json", expectedStatus: http.StatusForbidden},
		{name: "token inválido", body: validBody, contentType: "application/json", token: "outro-token", expectedStatus: http.StatusForbidden},
		{name: "content type ausente", body: validBody, token: expectedToken, expectedStatus: http.StatusUnsupportedMediaType},
		{name: "content type não suportado", body: validBody, contentType: "text/plain", token: expectedToken, expectedStatus: http.StatusUnsupportedMediaType},
		{name: "json inválido", body: `{"login_name":`, contentType: "application/json", token: expectedToken, expectedStatus: http.StatusBadRequest},
		{name: "json nulo", body: `null`, contentType: "application/json", token: expectedToken, expectedStatus: http.StatusBadRequest},
		{name: "campo desconhecido", body: `{"login_name":"gerente","password":"senha","admin":true}`, contentType: "application/json", token: expectedToken, expectedStatus: http.StatusBadRequest},
		{name: "mais de um valor json", body: validBody + ` {}`, contentType: "application/json", token: expectedToken, expectedStatus: http.StatusBadRequest},
		{name: "credenciais ausentes", body: `{"login_name":"","password":""}`, contentType: "application/json", token: expectedToken, expectedStatus: http.StatusBadRequest},
		{name: "corpo grande demais", body: `{"login_name":"gerente","password":"` + strings.Repeat("a", 4100) + `"}`, contentType: "application/json", token: expectedToken, expectedStatus: http.StatusRequestEntityTooLarge},
		{name: "credenciais recusadas pelo service", body: validBody, contentType: "application/json", token: expectedToken, serviceErr: invalidInput("senha fraca"), expectedStatus: http.StatusBadRequest, expectedCalls: 1},
		{name: "gerente já cadastrado", body: validBody, contentType: "application/json", token: expectedToken, serviceErr: ErrManagerAlreadyRegistered, expectedStatus: http.StatusConflict, expectedCalls: 1},
		{name: "erro interno não é exposto", body: validBody, contentType: "application/json", token: expectedToken, serviceErr: errors.New("detalhe privado do banco"), expectedStatus: http.StatusInternalServerError, expectedCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			calls := 0
			registrar := managerRegistrarFunc(func(gotCtx context.Context, loginName, password string) error {
				calls++
				assert.Same(t, ctx, gotCtx)
				if tt.expectedLogin != "" {
					assert.Equal(t, tt.expectedLogin, loginName)
					assert.Equal(t, tt.expectedPassword, password)
				}
				return tt.serviceErr
			})

			setupToken := expectedToken
			if tt.disableSetup {
				setupToken = ""
			}
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(registrar, HandlerConfig{SetupToken: setupToken}), func(next http.Handler) http.Handler { return next })
			request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(tt.body)).WithContext(ctx)
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			if tt.token != "" {
				request.Header.Set(setupTokenHeader, tt.token)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)

			assert.Equal(t, tt.expectedStatus, response.Code)
			assert.Equal(t, tt.expectedCalls, calls)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))

			var body map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			if tt.expectedStatus == http.StatusCreated {
				assert.Equal(t, map[string]any{"registered": true}, body)
				assert.NotContains(t, response.Body.String(), tt.expectedPassword)
			} else {
				assert.NotEmpty(t, body["error"])
			}
			if tt.serviceErr != nil && tt.expectedStatus == http.StatusInternalServerError {
				assert.NotContains(t, response.Body.String(), tt.serviceErr.Error())
			}
		})
	}
}

func TestMatchesSetupToken(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		want     bool
	}{
		{name: "correspondente", provided: "segredo", want: true},
		{name: "divergente", provided: "outro", want: false},
		{name: "vazio", provided: "", want: false},
		{name: "longo demais", provided: strings.Repeat("a", 257), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesSetupToken("segredo", tt.provided))
		})
	}
}

func TestLoginHandlerSetsHttpOnlySessionCookie(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registrar := managerRegistrarFunc(func(gotCtx context.Context, loginName, password string) error {
		assert.Same(t, ctx, gotCtx)
		assert.Equal(t, "Gerente_1", loginName)
		assert.Equal(t, "UmaSenhaForteComMaisDe15Caracteres", password)
		return nil
	})
	mux := http.NewServeMux()
	RegisterRoutes(mux, NewHandler(registrar, HandlerConfig{CookieSecure: true}), func(next http.Handler) http.Handler { return next })
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"login_name":"Gerente_1","password":"UmaSenhaForteComMaisDe15Caracteres"}`)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	assert.NotContains(t, response.Body.String(), "test-session-token")
	assert.NotContains(t, response.Body.String(), "UmaSenhaForteComMaisDe15Caracteres")
	var body sessionResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.True(t, body.Authenticated)
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, secureSessionCookieName, cookies[0].Name)
	assert.True(t, cookies[0].Secure)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
}

func TestLogoutHandlerRevokesAndClearsCookie(t *testing.T) {
	var gotToken string
	registrar := managerRegistrarFunc(func(_ context.Context, _, _ string) error { return nil })
	handler := NewHandler(registrar, HandlerConfig{})
	handler.service = logoutServiceStub{handlerService: handler.service, token: &gotToken}
	mux := http.NewServeMux()
	RegisterRoutes(mux, handler, func(next http.Handler) http.Handler { return next })
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: localSessionCookieName, Value: "opaque-session-token"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, "opaque-session-token", gotToken)
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, localSessionCookieName, cookies[0].Name)
	assert.Equal(t, -1, cookies[0].MaxAge)
	assert.True(t, cookies[0].HttpOnly)
}

type logoutServiceStub struct {
	handlerService
	token *string
}

func (s logoutServiceStub) Logout(_ context.Context, token string) error {
	*s.token = token
	return nil
}
