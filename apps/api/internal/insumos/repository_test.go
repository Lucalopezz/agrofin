// Verifica o cadastro HTTP persistido e o isolamento entre contas.
package insumos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

func TestProductsIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("configure TEST_DATABASE_URL com um banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	const origin = "http://localhost:5173"
	const path = "/input-products"
	suffix := uuid.NewString()[:8]
	logins := []string{"insumos-a-" + suffix, "insumos-b-" + suffix}
	t.Cleanup(func() {
		// O prazo de limpeza independe da requisição que falhou.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, err := pool.Exec(cleanupCtx, "DELETE FROM input_products WHERE user_id IN (SELECT id FROM users WHERE login_name = ANY($1))", logins)
		assert.NoError(t, err)
		_, err = pool.Exec(cleanupCtx, "DELETE FROM users WHERE login_name = ANY($1)", logins)
		assert.NoError(t, err)
	})

	authService := auth.NewService(auth.NewRepository(pool))
	middleware := auth.NewMiddleware(authService, false)
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux, auth.NewHandler(authService, auth.HandlerConfig{}), middleware.RequireSession)
	RegisterRoutes(mux, NewHandler(NewService(NewRepository(pool))), middleware.RequireSession)
	server := auth.RequireSameOrigin(origin, mux)

	request := func(method, path, body string, cookie *http.Cookie, sendMarker bool, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", origin)
			if sendMarker {
				req.Header.Set("X-Agrofin-Request", "1")
			}
		}
		response := httptest.NewRecorder()
		server.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, "%s %s: %s", method, path, response.Body)
		return response
	}
	login := func(loginName string) *http.Cookie {
		t.Helper()
		const password = "senha-de-teste-insumos"
		require.NoError(t, authService.RegisterManager(ctx, "Cliente de teste", loginName, password))
		response := request(http.MethodPost, "/auth/login",
			`{"login_name":"`+loginName+`","password":"`+password+`"}`,
			nil, true, http.StatusOK,
		)
		result := response.Result()
		defer result.Body.Close()
		cookies := result.Cookies()
		require.Len(t, cookies, 1)
		return cookies[0]
	}
	create := func(cookie *http.Cookie, body string) productResponse {
		t.Helper()
		response := request(http.MethodPost, path, body, cookie, true, http.StatusCreated)
		assert.NotContains(t, response.Body.String(), `"user_id"`)
		var product productResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &product))
		assert.NoError(t, uuid.Validate(product.ID))
		assert.Equal(t, "pending", product.StockControlState)
		return product
	}

	const fertilizer = `{"name":"  Adubo orgânico  ","category":"fertilizante","default_unit":" kg ","notes":" Aplicação foliar "}`
	request(http.MethodGet, path, "", nil, false, http.StatusUnauthorized)
	request(http.MethodPost, path, fertilizer, nil, true, http.StatusUnauthorized)
	first := login(logins[0])
	second := login(logins[1])
	request(http.MethodPost, path, fertilizer, first, false, http.StatusForbidden)
	request(http.MethodPost, path, `{"name":"Adubo","category":"fertilizante","default_unit":"kg","user_id":"outro"}`, first, true, http.StatusBadRequest)

	pesticide := create(first, `{"name":"Defensivo de teste","category":"defensivo","default_unit":"L"}`)
	product := create(first, fertilizer)
	assert.Equal(t, "Adubo orgânico", product.Name)
	assert.Equal(t, CategoryFertilizer, product.Category)
	assert.Equal(t, "kg", product.DefaultUnit)
	assert.Equal(t, "Aplicação foliar", product.Notes)

	empty := request(http.MethodGet, path, "", second, false, http.StatusOK)
	assert.JSONEq(t, "[]", empty.Body.String())
	otherProduct := create(second, `{"name":"Outro adubo","category":"fertilizante","default_unit":"g"}`)
	for _, tt := range []struct {
		cookie   *http.Cookie
		expected []productResponse
	}{
		{first, []productResponse{product, pesticide}},
		{second, []productResponse{otherProduct}},
	} {
		response := request(http.MethodGet, path, "", tt.cookie, false, http.StatusOK)
		var products []productResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &products))
		assert.Equal(t, tt.expected, products)
	}

	// O cadastro pendente preserva a ausência de um saldo de abertura.
	var movements int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM stock_movements WHERE product_id = $1",
		product.ID,
	).Scan(&movements))
	assert.Zero(t, movements)
}
