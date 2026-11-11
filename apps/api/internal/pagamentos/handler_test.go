// Testa autenticação, entradas e respostas HTTP do resumo da folha.
package pagamentos

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

type payrollSummaryServiceFunc func(context.Context, string, Period) (PeriodSummary, error)

func (f payrollSummaryServiceFunc) Summarize(ctx context.Context, userID string, period Period) (PeriodSummary, error) {
	return f(ctx, userID, period)
}

func TestPayrollSummaryHandler(t *testing.T) {
	start := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 6)
	path := "/payroll-periods/summary?start_date=2026-11-02&end_date=2026-11-08"
	tests := []struct {
		name        string
		path        string
		noSession   bool
		serviceErr  error
		emptyResult bool
		wantStatus  int
		wantCalls   int
	}{
		{name: "retorna resumo com datas e componentes separados", path: path, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "período vazio retorna lista vazia", path: path, emptyResult: true, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "sem sessão", path: path, noSession: true, wantStatus: http.StatusUnauthorized},
		{name: "sem data inicial", path: "/payroll-periods/summary?end_date=2026-11-08", wantStatus: http.StatusBadRequest},
		{name: "parâmetro repetido", path: path + "&start_date=2026-11-03", wantStatus: http.StatusBadRequest},
		{name: "parâmetro extra", path: path + "&user_id=outro", wantStatus: http.StatusBadRequest},
		{name: "query malformada", path: "/payroll-periods/summary?start_date=%zz&end_date=2026-11-08", wantStatus: http.StatusBadRequest},
		{name: "data inválida", path: "/payroll-periods/summary?start_date=2026-02-30&end_date=2026-11-08", wantStatus: http.StatusBadRequest},
		{name: "período invertido", path: "/payroll-periods/summary?start_date=2026-11-08&end_date=2026-11-02", serviceErr: ErrInvalidPayrollPeriod, wantStatus: http.StatusBadRequest, wantCalls: 1},
		{name: "prazo esgotado", path: path, serviceErr: context.DeadlineExceeded, wantStatus: http.StatusGatewayTimeout, wantCalls: 1},
		{name: "falha interna sem detalhe", path: path, serviceErr: errors.New("valor protegido privado"), wantStatus: http.StatusInternalServerError, wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if !tt.noSession {
				ctx = auth.WithPrincipal(ctx, auth.Principal{UserID: "user-from-session"})
			}
			calls := 0
			service := payrollSummaryServiceFunc(func(gotCtx context.Context, userID string, period Period) (PeriodSummary, error) {
				calls++
				assert.Equal(t, "user-from-session", userID)
				if tt.wantCalls > 0 && tt.serviceErr != nil {
					return PeriodSummary{}, tt.serviceErr
				}
				assert.Equal(t, Period{StartDate: start, EndDate: end}, period)
				summary := PeriodSummary{Period: period}
				if !tt.emptyResult {
					summary.Employees = []EmployeeSummary{{
						EmployeeID: "employee-a", EmployeeName: "João", NormalBoxes: 10,
						MillingBoxes: 4, NormalAmountCents: 5000, MillingAmountCents: 800,
						HarvestAmountCents: 5800, DailyWorkAmountCents: 8000, DueAmountCents: 13800,
					}}
				}
				return summary, nil
			})
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(service), func(next http.Handler) http.Handler { return next })
			request := httptest.NewRequest(http.MethodGet, tt.path, nil).WithContext(ctx)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantCalls, calls)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
			assert.NotContains(t, response.Body.String(), "valor protegido privado")
			if tt.wantStatus == http.StatusOK {
				var body periodSummaryResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
				assert.Equal(t, periodResponse{StartDate: "2026-11-02", EndDate: "2026-11-08"}, body.Period)
				if tt.emptyResult {
					assert.NotNil(t, body.Employees)
					assert.Empty(t, body.Employees)
				} else {
					require.Len(t, body.Employees, 1)
					assert.Equal(t, employeeResponse{ID: "employee-a", Name: "João"}, body.Employees[0].Employee)
					assert.EqualValues(t, 13800, body.Employees[0].DueAmountCents)
					assert.EqualValues(t, 8000, body.Employees[0].DailyWorkAmountCents)
				}
			}
			if tt.wantStatus == http.StatusBadRequest {
				assert.True(t, strings.Contains(response.Body.String(), `"error"`))
			}
		})
	}
}
