package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sessionResolverFunc func(context.Context, string) (Principal, error)

func (f sessionResolverFunc) AuthenticateSession(ctx context.Context, token string) (Principal, error) {
	return f(ctx, token)
}

func TestRequireSession(t *testing.T) {
	tests := []struct {
		name           string
		cookie         *http.Cookie
		resolverErr    error
		expectedStatus int
		expectedNext   bool
	}{
		{name: "sessão válida", cookie: &http.Cookie{Name: localSessionCookieName, Value: "token"}, expectedStatus: http.StatusNoContent, expectedNext: true},
		{name: "cookie ausente", expectedStatus: http.StatusUnauthorized},
		{name: "sessão revogada", cookie: &http.Cookie{Name: localSessionCookieName, Value: "token"}, resolverErr: ErrInvalidSession, expectedStatus: http.StatusUnauthorized},
		{name: "falha ao consultar sessão", cookie: &http.Cookie{Name: localSessionCookieName, Value: "token"}, resolverErr: errors.New("erro privado"), expectedStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			resolver := sessionResolverFunc(func(_ context.Context, token string) (Principal, error) {
				if tt.cookie != nil {
					assert.Equal(t, tt.cookie.Value, token)
				}
				if tt.resolverErr != nil {
					return Principal{}, tt.resolverErr
				}
				return Principal{UserID: "gerente-id"}, nil
			})
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				principal, ok := PrincipalFromContext(r.Context())
				assert.True(t, ok)
				assert.Equal(t, "gerente-id", principal.UserID)
				w.WriteHeader(http.StatusNoContent)
			})
			middleware := NewMiddleware(resolver, false)
			request := httptest.NewRequest(http.MethodGet, "/employees", nil)
			if tt.cookie != nil {
				request.AddCookie(tt.cookie)
			}
			response := httptest.NewRecorder()
			middleware.RequireSession(next).ServeHTTP(response, request)

			assert.Equal(t, tt.expectedStatus, response.Code)
			assert.Equal(t, tt.expectedNext, called)
			if tt.expectedStatus != http.StatusNoContent {
				assert.NotContains(t, response.Body.String(), "erro privado")
			}
		})
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "desenvolvimento local", true: "https"}[secure], func(t *testing.T) {
			response := httptest.NewRecorder()
			setSessionCookie(response, "opaque-token", time.Now().Add(time.Hour), secure)
			cookies := response.Result().Cookies()
			require.Len(t, cookies, 1)
			cookie := cookies[0]
			assert.Equal(t, sessionCookieName(secure), cookie.Name)
			assert.Equal(t, "/", cookie.Path)
			assert.Empty(t, cookie.Domain)
			assert.True(t, cookie.HttpOnly)
			assert.Equal(t, secure, cookie.Secure)
			assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
			assert.Greater(t, cookie.MaxAge, 0)
		})
	}
}
