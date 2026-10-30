package funcionarios

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

type employeeServiceFunc func(context.Context, string, string) (Employee, error)

func (f employeeServiceFunc) Create(ctx context.Context, userID, name string) (Employee, error) {
	return f(ctx, userID, name)
}

func TestCreateEmployeeHandler(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		contentType   string
		unconfigured  bool
		serviceErr    error
		expectedCode  int
		expectedCalls int
	}{
		{name: "cadastro", body: `{"name":"Ana"}`, contentType: "application/json; charset=utf-8", expectedCode: 201, expectedCalls: 1},
		{name: "sem gerente autenticado", body: `{"name":"Ana"}`, contentType: "application/json", unconfigured: true, expectedCode: 503},
		{name: "sem content type", body: `{"name":"Ana"}`, expectedCode: 415},
		{name: "tipo não suportado", body: `{"name":"Ana"}`, contentType: "text/plain", expectedCode: 415},
		{name: "json inválido", body: `{"name":`, contentType: "application/json", expectedCode: 400},
		{name: "corpo vazio", contentType: "application/json", expectedCode: 400},
		{name: "null", body: `null`, contentType: "application/json", expectedCode: 400},
		{name: "array", body: `[]`, contentType: "application/json", expectedCode: 400},
		{name: "nome numérico", body: `{"name":123}`, contentType: "application/json", expectedCode: 400},
		// O navegador não pode escolher o vínculo técnico nem o estado inicial.
		{name: "user_id no corpo", body: `{"name":"Ana","user_id":"outro-gerente"}`, contentType: "application/json", expectedCode: 400},
		{name: "estado no corpo", body: `{"name":"Ana","active":false}`, contentType: "application/json", expectedCode: 400},
		{name: "mais de um json", body: `{"name":"Ana"} {}`, contentType: "application/json", expectedCode: 400},
		{name: "conteúdo após json", body: `{"name":"Ana"} lixo`, contentType: "application/json", expectedCode: 400},
		{name: "corpo muito grande", body: `{"name":"` + strings.Repeat("a", 4096) + `"}`, contentType: "application/json", expectedCode: 413},
		{name: "espaços extras excedem limite", body: `{"name":"Ana"}` + strings.Repeat(" ", 4096), contentType: "application/json", expectedCode: 413},
		{name: "nome rejeitado pelo serviço", body: `{"name":""}`, contentType: "application/json", serviceErr: ErrInvalidName, expectedCode: 400, expectedCalls: 1},
		{name: "falha interna", body: `{"name":"Ana"}`, contentType: "application/json", serviceErr: errors.New("detalhe privado do banco"), expectedCode: 500, expectedCalls: 1},
		{name: "prazo esgotado", body: `{"name":"Ana"}`, contentType: "application/json", serviceErr: context.DeadlineExceeded, expectedCode: 504, expectedCalls: 1},
		{name: "cancelamento", body: `{"name":"Ana"}`, contentType: "application/json", serviceErr: context.Canceled, expectedCode: 408, expectedCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			service := employeeServiceFunc(func(gotCtx context.Context, userID, name string) (Employee, error) {
				calls++
				assert.Same(t, ctx, gotCtx)
				assert.Equal(t, "gerente-do-servidor", userID)
				return Employee{ID: "employee-id", UserID: userID, Name: name, Active: true}, tt.serviceErr
			})
			userID := "gerente-do-servidor"
			if tt.unconfigured {
				userID = ""
			} else {
				ctx = auth.WithPrincipal(ctx, auth.Principal{UserID: userID})
			}
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(service))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/employees", strings.NewReader(tt.body)).WithContext(ctx)
			req.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req)

			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedCalls, calls)
			assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
			var body map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			assert.NotContains(t, body, "user_id")
			assert.NotContains(t, response.Body.String(), "detalhe privado")
			if tt.expectedCode == http.StatusCreated {
				assert.Equal(t, map[string]any{"id": "employee-id", "name": "Ana", "active": true}, body)
			} else {
				assert.NotEmpty(t, body["error"])
			}
		})
	}
}
