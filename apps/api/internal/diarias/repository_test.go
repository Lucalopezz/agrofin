package diarias

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyWorkRepositoryIntegration(t *testing.T) {
	// A integração exige um banco separado e migrado; DATABASE_URL nunca é usado aqui.
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("configure TEST_DATABASE_URL com um banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()

	userIDs := make([]string, 0, 2)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, err := pool.Exec(cleanupCtx, "DELETE FROM daily_work WHERE user_id::text = ANY($1)", userIDs)
		assert.NoError(t, err)
		_, err = pool.Exec(cleanupCtx, "DELETE FROM employees WHERE user_id::text = ANY($1)", userIDs)
		assert.NoError(t, err)
		_, err = pool.Exec(cleanupCtx, "DELETE FROM users WHERE id::text = ANY($1)", userIDs)
		assert.NoError(t, err)
	}()

	employeeIDs := make([]string, 0, 2)
	for range 2 {
		var userID, employeeID string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (name, login_name, password_hash)
			VALUES ('Gerente de teste', 'teste-diarias-' || gen_random_uuid()::text, 'hash-de-teste')
			RETURNING id::text
		`).Scan(&userID)
		require.NoError(t, err)
		userIDs = append(userIDs, userID)
		err = pool.QueryRow(ctx, `
			INSERT INTO employees (user_id, name, active)
			VALUES ($1, 'Funcionário de teste', true)
			RETURNING id::text
		`, userID).Scan(&employeeID)
		require.NoError(t, err)
		employeeIDs = append(employeeIDs, employeeID)
	}

	key := bytes.Repeat([]byte{7}, 32)
	keyring, err := financialcipher.NewKeyring("test-v1", map[string][]byte{"test-v1": key})
	require.NoError(t, err)
	cipher, err := NewCipher(keyring)
	require.NoError(t, err)
	repo := NewRepository(pool)
	service := NewService(repo, cipher)
	date := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	work, err := service.Create(ctx, userIDs[0], CreateDailyWorkInput{
		EmployeeID: employeeIDs[0], WorkDate: date, AmountCents: 8000, Notes: "Trabalho no talhão 2",
	})
	require.NoError(t, err)
	assert.Equal(t, date, work.WorkDate)
	secondDate := date.AddDate(0, 0, 1)
	_, err = service.Create(ctx, userIDs[0], CreateDailyWorkInput{
		EmployeeID: employeeIDs[0], WorkDate: secondDate, AmountCents: 2500,
	})
	require.NoError(t, err)
	_, err = service.Create(ctx, userIDs[1], CreateDailyWorkInput{
		EmployeeID: employeeIDs[1], WorkDate: date, AmountCents: 1000,
	})
	require.NoError(t, err)

	summaries, err := service.Summarize(ctx, userIDs[0], DailyWorkPeriod{StartDate: date, EndDate: secondDate})
	require.NoError(t, err)
	require.Len(t, summaries, 1, "o resumo não deve incluir diárias de outra conta")
	assert.Equal(t, employeeIDs[0], summaries[0].EmployeeID)
	assert.Equal(t, "Funcionário de teste", summaries[0].EmployeeName)
	assert.EqualValues(t, 10500, summaries[0].DailyWorkAmountCents)

	empty, err := service.Summarize(ctx, userIDs[0], DailyWorkPeriod{
		StartDate: secondDate.AddDate(0, 0, 1), EndDate: secondDate.AddDate(0, 0, 2),
	})
	require.NoError(t, err)
	assert.Empty(t, empty, "o filtro de data deve excluir lançamentos fora do período")

	var stored ProtectedDailyWork
	var notes *string
	err = pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, employee_id::text, work_date,
		       amount_ciphertext, amount_nonce, amount_key_version, notes
		FROM daily_work
		WHERE id = $1
	`, work.ID).Scan(
		&stored.ID, &stored.UserID, &stored.EmployeeID, &stored.WorkDate,
		&stored.Amount.Ciphertext, &stored.Amount.Nonce, &stored.Amount.KeyVersion, &notes,
	)
	require.NoError(t, err)
	assert.Equal(t, userIDs[0], stored.UserID)
	assert.Equal(t, employeeIDs[0], stored.EmployeeID)
	assert.Equal(t, date, stored.WorkDate)
	assert.Equal(t, "test-v1", stored.Amount.KeyVersion)
	assert.NotNil(t, notes)
	assert.Equal(t, "Trabalho no talhão 2", *notes)
	assert.Len(t, stored.Amount.Nonce, 12)
	assert.Len(t, stored.Amount.Ciphertext, 24)
	var plaintext [8]byte
	binary.BigEndian.PutUint64(plaintext[:], 8000)
	assert.False(t, bytes.Contains(stored.Amount.Ciphertext, plaintext[:]), "o banco não deve conter centavos em claro")

	// Uma instância nova consegue recuperar o envelope com a mesma versão histórica.
	recoveredKeys, err := financialcipher.NewKeyring("test-v1", map[string][]byte{"test-v1": key})
	require.NoError(t, err)
	recoveredCipher, err := NewCipher(recoveredKeys)
	require.NoError(t, err)
	gotCents, err := recoveredCipher.DecryptDailyWorkAmount(
		stored.UserID, stored.ID, stored.EmployeeID, stored.WorkDate, stored.Amount,
	)
	require.NoError(t, err)
	assert.EqualValues(t, 8000, gotCents)

	_, err = service.Create(ctx, userIDs[1], CreateDailyWorkInput{
		EmployeeID: employeeIDs[0], WorkDate: date, AmountCents: 500,
	})
	assert.ErrorIs(t, err, ErrEmployeeNotFound, "uma conta não pode lançar diária para funcionário alheio")
	_, err = pool.Exec(ctx, "UPDATE employees SET active = false, deactivated_at = now() WHERE id = $1", employeeIDs[0])
	require.NoError(t, err)
	_, err = service.Create(ctx, userIDs[0], CreateDailyWorkInput{
		EmployeeID: employeeIDs[0], WorkDate: date, AmountCents: 500,
	})
	assert.ErrorIs(t, err, ErrEmployeeInactive)
	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM daily_work WHERE user_id = $1", userIDs[0]).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "falhas de dono ou estado não podem inserir outra diária")
}
