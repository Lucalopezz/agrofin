// Testa o contrato HTTP de criação de diárias.
package diarias

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dailyWorkServiceStub struct {
	create func(context.Context, string, CreateDailyWorkInput) (DailyWork, error)
}

func (s dailyWorkServiceStub) Create(ctx context.Context, userID string, input CreateDailyWorkInput) (DailyWork, error) {
	return s.create(ctx, userID, input)
}

func TestCreateDailyWorkHandler(t *testing.T) {
	date := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		body          string
		contentType   string
		noSession     bool
		serviceErr    error
		expectedCode  int
		expectedCalls int
	}{
		{name: "cria diária", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":8000,"notes":"Trabalho"}`, contentType: "application/json", expectedCode: http.StatusCreated, expectedCalls: 1},
		{name: "sem sessão", body: `{}`, contentType: "application/json", noSession: true, expectedCode: http.StatusUnauthorized},
		{name: "sem content type", body: `{}`, expectedCode: http.StatusUnsupportedMediaType},
		{name: "json inválido", body: `{"employee_id":`, contentType: "application/json", expectedCode: http.StatusBadRequest},
		{name: "corpo nulo", body: `null`, contentType: "application/json", expectedCode: http.StatusBadRequest},
		{name: "campo desconhecido", body: `{"employee_id":"` + validEmployeeID + `","user_id":"outro"}`, contentType: "application/json", expectedCode: http.StatusBadRequest},
		{name: "mais de um objeto", body: `{} {}`, contentType: "application/json", expectedCode: http.StatusBadRequest},
		{name: "uuid inválido", body: `{"employee_id":"x","work_date":"2026-11-04","amount_cents":1}`, contentType: "application/json", expectedCode: http.StatusBadRequest},
		{name: "data inválida", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-02-30","amount_cents":1}`, contentType: "application/json", expectedCode: http.StatusBadRequest},
		{name: "data com formato diferente", body: `{"employee_id":"` + validEmployeeID + `","work_date":"04-11-2026","amount_cents":1}`, contentType: "application/json", expectedCode: http.StatusBadRequest},
		{name: "corpo excede limite", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":1,"notes":"` + strings.Repeat("a", 4096) + `"}`, contentType: "application/json", expectedCode: http.StatusRequestEntityTooLarge},
		{name: "valor inválido pelo domínio", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":0}`, contentType: "application/json", serviceErr: ErrInvalidDailyWorkAmount, expectedCode: http.StatusBadRequest, expectedCalls: 1},
		{name: "funcionário inexistente", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":1}`, contentType: "application/json", serviceErr: ErrEmployeeNotFound, expectedCode: http.StatusNotFound, expectedCalls: 1},
		{name: "funcionário inativo", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":1}`, contentType: "application/json", serviceErr: ErrEmployeeInactive, expectedCode: http.StatusConflict, expectedCalls: 1},
		{name: "falha interna sem detalhe", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":1}`, contentType: "application/json", serviceErr: errors.New("detalhe privado"), expectedCode: http.StatusInternalServerError, expectedCalls: 1},
		{name: "prazo esgotado", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":1}`, contentType: "application/json", serviceErr: context.DeadlineExceeded, expectedCode: http.StatusGatewayTimeout, expectedCalls: 1},
		{name: "cancelada", body: `{"employee_id":"` + validEmployeeID + `","work_date":"2026-11-04","amount_cents":1}`, contentType: "application/json", serviceErr: context.Canceled, expectedCode: http.StatusRequestTimeout, expectedCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if !tt.noSession {
				ctx = auth.WithPrincipal(ctx, auth.Principal{UserID: "user-from-session"})
			}
			calls := 0
			service := dailyWorkServiceStub{create: func(gotCtx context.Context, userID string, input CreateDailyWorkInput) (DailyWork, error) {
				calls++
				assert.Equal(t, "user-from-session", userID)
				if tt.expectedCode == http.StatusCreated {
					assert.Equal(t, validEmployeeID, input.EmployeeID)
					assert.Equal(t, date, input.WorkDate)
					assert.EqualValues(t, 8000, input.AmountCents)
					assert.Equal(t, "Trabalho", input.Notes)
				}
				return DailyWork{
					ID: "daily-id", UserID: userID, EmployeeID: input.EmployeeID,
					WorkDate: date, AmountCents: input.AmountCents, Notes: input.Notes,
				}, tt.serviceErr
			}}
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(service), func(next http.Handler) http.Handler { return next })
			request := httptest.NewRequest(http.MethodPost, "/daily-work", strings.NewReader(tt.body)).WithContext(ctx)
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)

			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedCalls, calls)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
			assert.NotContains(t, response.Body.String(), "detalhe privado")
			if tt.expectedCode == http.StatusCreated {
				var body dailyWorkResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
				assert.Equal(t, dailyWorkResponse{
					ID: "daily-id", EmployeeID: validEmployeeID,
					WorkDate: "2026-11-04", AmountCents: 8000, Notes: "Trabalho",
				}, body)
			}
		})
	}
}
