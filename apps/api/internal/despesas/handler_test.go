// Testa o contrato HTTP das operações de despesas.
package despesas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/stretchr/testify/assert"
)

type handlerStub struct {
	err   error
	owner string
}

func (s *handlerStub) Create(_ context.Context, owner string, in Input) (Expense, error) {
	s.owner = owner
	return Expense{ID: "despesa", PaidAt: in.PaidAt, Category: in.Category, AmountCents: in.AmountCents}, s.err
}
func (s *handlerStub) List(_ context.Context, owner string, _, _ time.Time) ([]Expense, error) {
	s.owner = owner
	return nil, s.err
}
func (s *handlerStub) Update(_ context.Context, owner, id string, in Input) (Expense, error) {
	s.owner = owner
	return Expense{ID: id, PaidAt: in.PaidAt, Category: in.Category, AmountCents: in.AmountCents}, s.err
}
func (s *handlerStub) Delete(_ context.Context, owner, id string) error {
	s.owner = owner
	return s.err
}
func TestExpenseHTTP(t *testing.T) {
	const valid = `{"paid_at":"2026-11-15","category":"Energia","amount_cents":12345,"description":"Irrigação"}`
	tests := []struct {
		name, method, path, body, media string
		auth                            bool
		serviceErr                      error
		status                          int
	}{
		{"sem sessão", "POST", "/expenses", valid, "application/json", false, nil, 401},
		{"criar sem chave", "POST", "/expenses", valid, "application/json", true, nil, 201},
		{"campo desconhecido", "POST", "/expenses", `{"paid_at":"2026-11-15","user_id":"outra"}`, "application/json", true, nil, 400},
		{"sem tipo de mídia", "POST", "/expenses", valid, "", true, nil, 415},
		{"corpo grande", "POST", "/expenses", strings.Repeat(" ", 5000), "application/json", true, nil, 413},
		{"lista vazia", "GET", "/expenses?start_date=2026-11-15&end_date=2026-11-15", "", "", true, nil, 200},
		{"período repetido", "GET", "/expenses?start_date=2026-11-15&start_date=2026-11-16&end_date=2026-11-16", "", "", true, nil, 400},
		{"edição", "PUT", "/expenses/id", valid, "application/json", true, nil, 200},
		{"exclusão", "DELETE", "/expenses/id", "", "", true, nil, 204},
		{"despesa alheia", "DELETE", "/expenses/id", "", "", true, ErrNotFound, 404},
		{"dados inválidos", "POST", "/expenses", valid, "application/json", true, ErrInvalid, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &handlerStub{err: tt.serviceErr}
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(stub), func(h http.Handler) http.Handler { return h })
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.media)
			if tt.auth {
				req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "conta-autenticada"}))
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code, w.Body.String())
			if tt.name == "lista vazia" {
				assert.JSONEq(t, `[]`, w.Body.String())
			}
			if tt.name == "criar sem chave" {
				assert.Equal(t, "conta-autenticada", stub.owner)
			}
		})
	}
}
