// Testa o contrato HTTP das vendas e dos recebimentos.
package vendas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/stretchr/testify/assert"
)

type salesHandlerStub struct {
	err error
}

func (s salesHandlerStub) Create(_ context.Context, _ string, in CreateSaleInput) (Sale, error) {
	return Sale{SoldAt: in.SoldAt}, s.err
}
func (s salesHandlerStub) List(context.Context, string, SalePeriod) ([]Sale, error) {
	return nil, s.err
}
func (s salesHandlerStub) Receive(context.Context, string, CreateReceiptInput) (Receipt, error) {
	return Receipt{}, s.err
}

func TestSalesHTTP(t *testing.T) {
	cases := []struct {
		name, method, path, body, contentType string
		authenticated                         bool
		err                                   error
		status                                int
	}{
		{name: "sem sessão", method: "GET", path: "/sales?start_date=2026-11-01&end_date=2026-11-30", status: 401},
		{name: "lista vazia", method: "GET", path: "/sales?start_date=2026-11-01&end_date=2026-11-30", authenticated: true, status: 200},
		{name: "datas repetidas", method: "GET", path: "/sales?start_date=2026-11-01&start_date=2026-11-02&end_date=2026-11-30", authenticated: true, status: 400},
		{name: "data inválida", method: "GET", path: "/sales?start_date=erro&end_date=2026-11-30", authenticated: true, status: 400},
		{name: "criar venda", method: "POST", path: "/sales", body: `{"sold_at":"2026-11-12","items":[]}`, contentType: "application/json", authenticated: true, status: 201},
		{name: "campo de dono rejeitado", method: "POST", path: "/sales", body: `{"sold_at":"2026-11-12","user_id":"outro"}`, contentType: "application/json", authenticated: true, status: 400},
		{name: "tipo de mídia", method: "POST", path: "/sales", body: `{}`, authenticated: true, status: 415},
		{name: "falha de domínio", method: "POST", path: "/sales", body: `{"sold_at":"2026-11-12"}`, contentType: "application/json", authenticated: true, err: ErrInvalidSaleItems, status: 400},
		{name: "id inválido", method: "POST", path: "/sales/erro/receipts", body: `{}`, contentType: "application/json", authenticated: true, status: 400},
		{name: "recebimento criado", method: "POST", path: "/sales/11111111-1111-4111-8111-111111111111/receipts", body: `{"amount_cents":100}`, contentType: "application/json", authenticated: true, status: 201},
		{name: "saldo insuficiente", method: "POST", path: "/sales/11111111-1111-4111-8111-111111111111/receipts", body: `{"amount_cents":100}`, contentType: "application/json", authenticated: true, err: ErrReceiptExceedsBalance, status: 409},
		{name: "venda não encontrada", method: "POST", path: "/sales/11111111-1111-4111-8111-111111111111/receipts", body: `{"amount_cents":100}`, contentType: "application/json", authenticated: true, err: ErrSaleNotFound, status: 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(salesHandlerStub{err: tc.err})
			mux := http.NewServeMux()
			RegisterRoutes(mux, h, func(next http.Handler) http.Handler { return next })
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			if tc.authenticated {
				req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: testOwner}))
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assert.Equal(t, tc.status, rec.Code, rec.Body.String())
			if tc.name == "lista vazia" {
				assert.JSONEq(t, `[]`, rec.Body.String())
			}
		})
	}
}
