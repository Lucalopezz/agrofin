package funcionarios

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

// Funções não configuradas falham se o handler chamar um caso de uso inesperado.
type employeeServiceStub struct {
	create func(context.Context, string, string) (Employee, error)
	list   func(context.Context, string) ([]Employee, error)
}

func (s employeeServiceStub) Create(ctx context.Context, userID, name string) (Employee, error) {
	return s.create(ctx, userID, name)
}

func (s employeeServiceStub) List(ctx context.Context, userID string) ([]Employee, error) {
	return s.list(ctx, userID)
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
		{name: "sem gerente autenticado", body: `{"name":"Ana"}`, contentType: "application/json", unconfigured: true, expectedCode: 401},
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
			service := employeeServiceStub{create: func(gotCtx context.Context, userID, name string) (Employee, error) {
				calls++
				assert.Same(t, ctx, gotCtx)
				assert.Equal(t, "gerente-do-servidor", userID)
				return Employee{ID: "employee-id", UserID: userID, Name: name, Active: true}, tt.serviceErr
			}}
			userID := "gerente-do-servidor"
			if tt.unconfigured {
				userID = ""
			} else {
				ctx = auth.WithPrincipal(ctx, auth.Principal{UserID: userID})
			}
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(service), func(next http.Handler) http.Handler { return next })
			req := httptest.NewRequest(http.MethodPost, "/employees", strings.NewReader(tt.body)).WithContext(ctx)
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

func TestListEmployeesHandler(t *testing.T) {
	employees := []Employee{
		{ID: "employee-1", UserID: "gerente-da-sessao", Name: "Ana", Active: true},
		{ID: "employee-2", UserID: "gerente-da-sessao", Name: "João", Active: false},
	}
	tests := []struct {
		name          string
		unconfigured  bool
		employees     []Employee
		serviceErr    error
		expectedCode  int
		expectedCalls int
		expectedJSON  string
	}{
		{name: "lista com estados preservados", employees: employees, expectedCode: 200, expectedCalls: 1,
			expectedJSON: `[{"id":"employee-1","name":"Ana","active":true},{"id":"employee-2","name":"João","active":false}]`},
		{name: "slice nil vira array vazio", expectedCode: 200, expectedCalls: 1, expectedJSON: `[]`},
		{name: "lista vazia", employees: []Employee{}, expectedCode: 200, expectedCalls: 1, expectedJSON: `[]`},
		{name: "sem sessão", unconfigured: true, expectedCode: 401},
		{name: "falha interna não expõe lista parcial", employees: employees, serviceErr: errors.New("detalhe privado do banco"), expectedCode: 500, expectedCalls: 1},
		{name: "prazo esgotado", serviceErr: fmt.Errorf("consulta: %w", context.DeadlineExceeded), expectedCode: 504, expectedCalls: 1},
		{name: "cancelamento", serviceErr: fmt.Errorf("consulta: %w", context.Canceled), expectedCode: 408, expectedCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if !tt.unconfigured {
				ctx = auth.WithPrincipal(ctx, auth.Principal{UserID: "gerente-da-sessao"})
			}
			calls := 0
			service := employeeServiceStub{list: func(gotCtx context.Context, userID string) ([]Employee, error) {
				calls++
				assert.Same(t, ctx, gotCtx)
				assert.Equal(t, "gerente-da-sessao", userID)
				return tt.employees, tt.serviceErr
			}}
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(service), func(next http.Handler) http.Handler { return next })
			// Identidades fornecidas pelo cliente não podem substituir o principal validado.
			request := httptest.NewRequest(http.MethodGet, "/employees?user_id=outro-gerente", nil).WithContext(ctx)
			request.Header.Set("X-User-ID", "outro-gerente")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)

			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedCalls, calls)
			assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
			assert.NotContains(t, response.Body.String(), "user_id")
			assert.NotContains(t, response.Body.String(), "detalhe privado")
			if tt.expectedJSON != "" {
				assert.JSONEq(t, tt.expectedJSON, response.Body.String())
			} else {
				var body map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
				assert.Len(t, body, 1)
				assert.NotEmpty(t, body["error"])
			}
		})
	}
}

func TestEmployeeRoutesRequireSession(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			// Um principal artificial não pode contornar o middleware registrado na rota.
			// O serviço vazio também faz o teste falhar se o fluxo alcançar o handler.
			called := false
			requireSession := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called = true
					w.WriteHeader(http.StatusUnauthorized)
				})
			}
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(employeeServiceStub{}), requireSession)
			ctx := auth.WithPrincipal(context.Background(), auth.Principal{UserID: "gerente"})
			request := httptest.NewRequest(method, "/employees", nil).WithContext(ctx)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			assert.True(t, called)
			assert.Equal(t, http.StatusUnauthorized, response.Code)
		})
	}
}
