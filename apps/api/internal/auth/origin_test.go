package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSessionConfig(t *testing.T) {
	tests := []struct {
		name         string
		origin       string
		cookieSecure bool
		shouldError  bool
	}{
		{name: "http local sem Secure", origin: "http://localhost:5173"},
		{name: "https com Secure", origin: "https://agrofin.example", cookieSecure: true},
		{name: "https sem Secure", origin: "https://agrofin.example", shouldError: true},
		{name: "http com Secure", origin: "http://localhost:5173", cookieSecure: true, shouldError: true},
		{name: "origem com caminho", origin: "https://agrofin.example/app", cookieSecure: true, shouldError: true},
		{name: "origem ausente", shouldError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionConfig(tt.origin, tt.cookieSecure)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestRequireSameOrigin(t *testing.T) {
	tests := []struct {
		name         string
		publicOrigin string
		method       string
		origin       string
		fetchSite    string
		marker       string
		wantStatus   int
		wantNext     bool
	}{
		{name: "post da mesma origem", method: http.MethodPost, origin: "http://localhost:5173", fetchSite: "same-origin", marker: "1", wantStatus: http.StatusNoContent, wantNext: true},
		{name: "porta padrão normalizada", publicOrigin: "http://localhost:80", method: http.MethodPost, origin: "http://localhost", marker: "1", wantStatus: http.StatusNoContent, wantNext: true, fetchSite: "same-origin"},
		{name: "get público sem cabeçalhos de origem", method: http.MethodGet, wantStatus: http.StatusNoContent, wantNext: true},
		{name: "origin ausente", method: http.MethodPost, marker: "1", wantStatus: http.StatusForbidden},
		{name: "origem diferente", method: http.MethodPost, origin: "http://atacante.example", marker: "1", wantStatus: http.StatusForbidden},
		{name: "fetch cross-site", method: http.MethodPost, origin: "http://localhost:5173", fetchSite: "cross-site", marker: "1", wantStatus: http.StatusForbidden},
		{name: "cabeçalho de navegador ausente", method: http.MethodPost, origin: "http://localhost:5173", fetchSite: "same-origin", wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(tt.method, "/api/v1/auth/login", nil)
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			if tt.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", tt.fetchSite)
			}
			if tt.marker != "" {
				request.Header.Set(requestMarkerHeader, tt.marker)
			}
			response := httptest.NewRecorder()
			publicOrigin := tt.publicOrigin
			if publicOrigin == "" {
				publicOrigin = "http://localhost:5173"
			}
			RequireSameOrigin(publicOrigin, next).ServeHTTP(response, request)
			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantNext, called)
		})
	}
}
