package funcionarios

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEmployeeIntegration(t *testing.T) {
	// A integração exige um banco de teste com as migrações aplicadas.
	// Não usamos DATABASE_URL para evitar gravar no banco de desenvolvimento.
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("configure TEST_DATABASE_URL com um banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err, "configurar o banco de teste")
	defer pool.Close()

	var accountID string
	err = pool.QueryRow(ctx, "INSERT INTO accounts (name) VALUES ($1) RETURNING id::text", "teste de funcionários").Scan(&accountID)
	require.NoError(t, err, "criar a conta de teste")
	defer func() {
		// A limpeza tem prazo próprio para funcionar mesmo se o teste cancelar
		// seu contexto. Só removemos registros vinculados à conta desta execução.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		// assert permite tentar a segunda limpeza mesmo se a primeira falhar.
		_, err := pool.Exec(cleanupCtx, "DELETE FROM employees WHERE account_id = $1", accountID)
		assert.NoError(t, err, "limpar os funcionários de teste")
		_, err = pool.Exec(cleanupCtx, "DELETE FROM accounts WHERE id = $1", accountID)
		assert.NoError(t, err, "limpar a conta de teste")
	}()

	service := NewService(NewRepository(pool))
	mux := http.NewServeMux()
	RegisterRoutes(mux, NewHandler(service, accountID))
	// Uma conexão HTTP real exercita rota, JSON, serviço e SQL juntos.
	server := httptest.NewServer(mux)
	defer server.Close()
	client := server.Client()
	client.Timeout = 5 * time.Second
	create := func(body string) employeeResponse {
		t.Helper()
		response, err := client.Post(server.URL+"/api/v1/employees", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer response.Body.Close()
		// Só lemos o funcionário quando a criação HTTP terminou com sucesso.
		require.Equal(t, http.StatusCreated, response.StatusCode)
		var employee employeeResponse
		require.NoError(t, json.NewDecoder(response.Body).Decode(&employee))
		return employee
	}
	first := create(`{"name":"  João D'Ávila  "}`)
	second := create(`{"name":"João D'Ávila"}`)
	assert.NotEmpty(t, first.ID)
	assert.NotEmpty(t, second.ID)
	assert.NotEqual(t, first.ID, second.ID)
	assert.True(t, first.Active)
	assert.True(t, second.Active)

	var name string
	var active bool
	var deactivated bool
	err = pool.QueryRow(ctx, `
		SELECT name, active, deactivated_at IS NOT NULL
		FROM employees WHERE account_id = $1 AND id = $2
	`, accountID, first.ID).Scan(&name, &active, &deactivated)
	require.NoError(t, err, "consultar funcionário persistido")
	assert.Equal(t, "João D'Ávila", name)
	assert.Equal(t, first.Name, name)
	assert.True(t, active)
	assert.False(t, deactivated)

	response, err := client.Post(server.URL+"/api/v1/employees", "application/json", strings.NewReader(`{"name":"   "}`))
	require.NoError(t, err)
	response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	// O cancelamento deve chegar ao driver e continuar identificável mesmo
	// depois de o repositório acrescentar contexto ao erro com %w.
	canceledCtx, canceled := context.WithCancel(ctx)
	canceled()
	_, err = service.Create(canceledCtx, accountID, "Ana")
	assert.ErrorIs(t, err, context.Canceled)
	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM employees WHERE account_id = $1", accountID).Scan(&count)
	require.NoError(t, err, "contar funcionários após cancelamento")
	assert.Equal(t, 2, count)
}
