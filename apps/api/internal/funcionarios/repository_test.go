package funcionarios

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
)

func TestEmployeesIntegration(t *testing.T) {
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

	const origin = "http://localhost:5173"
	const password = "UmaSenhaForteComMaisDe15Caracteres"
	logins := []string{"teste-funcionarios", "teste-funcionarios-outro"}
	defer func() {
		// A limpeza só atinge as contas deste teste e tem prazo próprio, mesmo
		// quando uma falha interrompe o fluxo antes de todas as contas existirem.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, err := pool.Exec(cleanupCtx, "DELETE FROM employees WHERE user_id IN (SELECT id FROM users WHERE login_name = ANY($1))", logins)
		assert.NoError(t, err, "limpar funcionários de teste")
		// As sessões são removidas pelo ON DELETE CASCADE do usuário.
		_, err = pool.Exec(cleanupCtx, "DELETE FROM users WHERE login_name = ANY($1)", logins)
		assert.NoError(t, err, "limpar gerentes de teste")
	}()

	service := NewService(NewRepository(pool))
	authService := auth.NewService(auth.NewRepository(pool))
	middleware := auth.NewMiddleware(authService, false)
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux, auth.NewHandler(authService, auth.HandlerConfig{}), middleware.RequireSession)
	RegisterRoutes(mux, NewHandler(service), middleware.RequireSession)
	// Servidor e cookies reais exercitam cadastro público, sessão e isolamento
	// juntos: o teste não injeta mais um principal para simular autenticação.
	server := httptest.NewServer(auth.RequireSameOrigin(origin, mux))
	defer server.Close()
	request := func(client *http.Client, method, path, body string, expectedStatus int) *http.Response {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, method, server.URL+path, strings.NewReader(body))
		require.NoError(t, err)
		if method != http.MethodGet {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", origin)
			req.Header.Set("X-Agrofin-Request", "1")
		}
		response, err := client.Do(req)
		require.NoError(t, err)
		if response.StatusCode != expectedStatus {
			response.Body.Close()
			t.Fatalf("%s %s: status %d, esperado %d", method, path, response.StatusCode, expectedStatus)
		}
		return response
	}
	newManager := func(loginName string) (*http.Client, string) {
		t.Helper()
		jar, err := cookiejar.New(nil)
		require.NoError(t, err)
		// Cookie jars separados representam navegadores de clientes diferentes.
		client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
		body := `{"name":"Gerente de teste","login_name":"` + loginName + `","password":"` + password + `"}`
		response := request(client, http.MethodPost, "/auth/register", body, http.StatusCreated)
		response.Body.Close()
		response = request(client, http.MethodPost, "/auth/login", `{"login_name":"`+loginName+`","password":"`+password+`"}`, http.StatusOK)
		response.Body.Close()
		response = request(client, http.MethodGet, "/auth/me", "", http.StatusOK)
		defer response.Body.Close()
		var profile struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&profile))
		require.NotEmpty(t, profile.User.ID)
		return client, profile.User.ID
	}
	client, userID := newManager(logins[0])
	otherClient, otherUserID := newManager(logins[1])
	assert.NotEqual(t, userID, otherUserID)

	// O login é normalizado antes de persistir: diferenças de maiúsculas e
	// espaços não podem criar uma segunda conta com a mesma credencial.
	response := request(otherClient, http.MethodPost, "/auth/register", `{"name":"Outro nome","login_name":" TESTE-FUNCIONARIOS ","password":"`+password+`"}`, http.StatusConflict)
	response.Body.Close()
	response = request(server.Client(), http.MethodGet, "/employees", "", http.StatusUnauthorized)
	response.Body.Close()
	list := func(client *http.Client) []employeeResponse {
		t.Helper()
		response, err := client.Get(server.URL + "/employees")
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
		var employees []employeeResponse
		require.NoError(t, json.NewDecoder(response.Body).Decode(&employees))
		// Decodificar null não gera erro, mas quebraria o contrato de coleção da API.
		require.NotNil(t, employees)
		return employees
	}
	assert.Empty(t, list(client))
	create := func(client *http.Client, body string) employeeResponse {
		t.Helper()
		response := request(client, http.MethodPost, "/employees", body, http.StatusCreated)
		defer response.Body.Close()
		// Só lemos o funcionário quando a criação HTTP terminou com sucesso.
		require.Equal(t, http.StatusCreated, response.StatusCode)
		var employee employeeResponse
		require.NoError(t, json.NewDecoder(response.Body).Decode(&employee))
		return employee
	}
	first := create(client, `{"name":"  João D'Ávila  "}`)
	second := create(client, `{"name":"João D'Ávila"}`)
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
		FROM employees WHERE user_id = $1 AND id = $2
	`, userID, first.ID).Scan(&name, &active, &deactivated)
	require.NoError(t, err, "consultar funcionário persistido")
	assert.Equal(t, "João D'Ávila", name)
	assert.Equal(t, first.Name, name)
	assert.True(t, active)
	assert.False(t, deactivated)

	response = request(client, http.MethodPost, "/employees", `{"name":"   "}`, http.StatusBadRequest)
	response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	// O cancelamento deve chegar ao driver e continuar identificável mesmo
	// depois de o repositório acrescentar contexto ao erro com %w.
	canceledCtx, canceled := context.WithCancel(ctx)
	canceled()
	_, err = service.Create(canceledCtx, userID, "Ana")
	assert.ErrorIs(t, err, context.Canceled)
	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM employees WHERE user_id = $1", userID).Scan(&count)
	require.NoError(t, err, "contar funcionários após cancelamento")
	assert.Equal(t, 2, count)

	ana := create(client, `{"name":"Ana"}`)
	// A listagem do cadastro preserva o estado dos inativos. Excluí-los da seleção
	// de novos lançamentos pertence ao fluxo de colheita, ainda não implementado.
	_, err = pool.Exec(ctx, "UPDATE employees SET active = false, deactivated_at = now() WHERE id = $1", second.ID)
	require.NoError(t, err)
	second.Active = false
	expected := []employeeResponse{ana, first, second}
	// Os dois nomes iguais são desempatados pelo ID, sem deduplicar pessoas.
	if second.ID < first.ID {
		expected[1], expected[2] = second, first
	}
	assert.Equal(t, expected, list(client))

	// As duas contas têm funcionários reais. Cada sessão deve enxergar somente
	// os seus, mesmo quando o cliente tenta escolher outro dono na requisição.
	assert.Empty(t, list(otherClient))
	otherEmployee := create(otherClient, `{"name":"Equipe do outro gerente"}`)
	assert.Equal(t, []employeeResponse{otherEmployee}, list(otherClient))
	assert.Equal(t, expected, list(client))
	response = request(client, http.MethodPost, "/employees", `{"name":"Tentativa","user_id":"`+otherUserID+`"}`, http.StatusBadRequest)
	response.Body.Close()
	response = request(client, http.MethodGet, "/employees?user_id="+otherUserID, "", http.StatusOK)
	var filtered []employeeResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&filtered))
	response.Body.Close()
	assert.Equal(t, expected, filtered, "a query não pode substituir o dono derivado da sessão")

	rename := func(employee employeeResponse, newName string) employeeResponse {
		t.Helper()
		body, err := json.Marshal(employeeNameRequest{Name: newName})
		require.NoError(t, err)
		response := request(client, http.MethodPatch, "/employees/"+employee.ID, string(body), http.StatusOK)
		defer response.Body.Close()
		var renamed employeeResponse
		require.NoError(t, json.NewDecoder(response.Body).Decode(&renamed))
		assert.Equal(t, employee.ID, renamed.ID)
		assert.Equal(t, employee.Active, renamed.Active)
		return renamed
	}
	response = request(server.Client(), http.MethodPatch, "/employees/"+first.ID, `{"name":"Tentativa"}`, http.StatusUnauthorized)
	response.Body.Close()

	// A rota herda a proteção de escrita: nem cookie válido dispensa origem e marcador.
	for _, missingHeader := range []string{"Origin", "X-Agrofin-Request"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, server.URL+"/employees/"+first.ID, strings.NewReader(`{"name":"Tentativa"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		req.Header.Set("X-Agrofin-Request", "1")
		req.Header.Del(missingHeader)
		response, err := client.Do(req)
		require.NoError(t, err)
		response.Body.Close()
		assert.Equal(t, http.StatusForbidden, response.StatusCode, missingHeader)
	}
	renamed := rename(first, " \tJoão  D'Ávila atualizado\n")
	assert.Equal(t, "João  D'Ávila atualizado", renamed.Name)
	assert.Equal(t, renamed, rename(renamed, renamed.Name), "repetir o mesmo nome continua sendo sucesso")
	// Nomes não são identificadores: renomear para um nome já usado é permitido.
	first = rename(first, ana.Name)
	assert.Equal(t, ana.Name, first.Name)

	var deactivatedAt time.Time
	err = pool.QueryRow(ctx, "SELECT deactivated_at FROM employees WHERE id = $1", second.ID).Scan(&deactivatedAt)
	require.NoError(t, err)
	second = rename(second, "  Funcionário inativo renomeado  ")
	assert.Equal(t, "Funcionário inativo renomeado", second.Name)
	var persistedName, persistedUserID string
	var persistedDeactivatedAt time.Time
	err = pool.QueryRow(ctx, "SELECT name, user_id::text, active, deactivated_at FROM employees WHERE id = $1", second.ID).
		Scan(&persistedName, &persistedUserID, &active, &persistedDeactivatedAt)
	require.NoError(t, err)
	assert.Equal(t, second.Name, persistedName)
	assert.Equal(t, userID, persistedUserID)
	assert.False(t, active)
	assert.True(t, deactivatedAt.Equal(persistedDeactivatedAt), "renomear preserva a data de desativação")

	// O mesmo 404 cobre ausência e registro alheio, inclusive com user_id forjado.
	for _, path := range []string{
		"/employees/00000000-0000-0000-0000-000000000001",
		"/employees/" + otherEmployee.ID + "?user_id=" + otherUserID,
	} {
		response = request(client, http.MethodPatch, path, `{"name":"Tentativa"}`, http.StatusNotFound)
		var body errorResponse
		require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
		response.Body.Close()
		assert.Equal(t, "Funcionário não encontrado", body.Error)
	}
	assert.Equal(t, []employeeResponse{otherEmployee}, list(otherClient))
	response = request(client, http.MethodPatch, "/employees/nao-e-uuid", `{"name":"Tentativa"}`, http.StatusBadRequest)
	response.Body.Close()
	response = request(client, http.MethodPatch, "/employees/"+first.ID, `{"name":"   "}`, http.StatusBadRequest)
	response.Body.Close()
	response = request(client, http.MethodPatch, "/employees/"+first.ID, `{"name":"Tentativa","user_id":"`+otherUserID+`"}`, http.StatusBadRequest)
	response.Body.Close()
	_, err = service.Rename(canceledCtx, userID, first.ID, "Tentativa cancelada")
	assert.ErrorIs(t, err, context.Canceled)
	// Ler novamente comprova persistência e que as tentativas rejeitadas não gravaram.
	assert.ElementsMatch(t, []employeeResponse{ana, first, second}, list(client))

	// Desativar a própria conta não deve revogar sessões de outros clientes.
	response = request(client, http.MethodPatch, "/auth/me/status", `{"active":false}`, http.StatusOK)
	response.Body.Close()
	response = request(client, http.MethodGet, "/employees", "", http.StatusUnauthorized)
	response.Body.Close()
	assert.Len(t, list(otherClient), 1)
	_, err = service.List(canceledCtx, userID)
	assert.ErrorIs(t, err, context.Canceled)
}
