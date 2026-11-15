// Testa cadastro, consulta, edição, exclusão e proteção de despesas no PostgreSQL.
package despesas

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

func TestExpenseIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("configure TEST_DATABASE_URL com banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	defer pool.Close()
	users := []string{}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, table := range []string{"expenses", "users"} {
			col := "user_id"
			if table == "users" {
				col = "id"
			}
			_, err := pool.Exec(c, "DELETE FROM "+table+" WHERE "+col+"::text=ANY($1)", users)
			assert.NoError(t, err)
		}
	}()
	for range 2 {
		var id string
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO users(name,login_name,password_hash) VALUES('Teste','despesa-'||gen_random_uuid()::text,'hash') RETURNING id::text`).Scan(&id))
		users = append(users, id)
	}
	key := bytes.Repeat([]byte{3}, 32)
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": key})
	require.NoError(t, err)
	cipher, err := NewCipher(keys)
	require.NoError(t, err)
	service := NewService(NewRepository(pool), cipher)
	date := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	in := Input{PaidAt: date, Category: "Energia", Description: "Irrigação", AmountCents: 12345}
	first, err := service.Create(ctx, users[0], in)
	require.NoError(t, err)
	id := first.ID
	// Dois envios iguais são dois lançamentos independentes, conforme o contrato.
	duplicate, err := service.Create(ctx, users[0], in)
	require.NoError(t, err)
	assert.NotEqual(t, id, duplicate.ID)
	require.NoError(t, service.Delete(ctx, users[0], duplicate.ID))
	var raw, nonce []byte
	var version string
	require.NoError(t, pool.QueryRow(ctx, `SELECT amount_ciphertext,amount_nonce,amount_key_version FROM expenses WHERE id=$1`, id).Scan(&raw, &nonce, &version))
	var plain [8]byte
	binary.BigEndian.PutUint64(plain[:], 12345)
	assert.NotContains(t, raw, plain[:])
	assert.Len(t, nonce, 12)
	assert.Equal(t, "v1", version)
	list, err := service.List(ctx, users[0], date, date)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.EqualValues(t, 12345, list[0].AmountCents)
	assert.Equal(t, "Energia", list[0].Category)
	list, err = service.List(ctx, users[1], date, date)
	require.NoError(t, err)
	assert.Empty(t, list)
	_, err = service.Update(ctx, users[1], id, in)
	assert.ErrorIs(t, err, ErrNotFound)
	changed := Input{PaidAt: date.AddDate(0, 0, 1), Category: "Energia", Description: "Corrigida", AmountCents: 20000}
	updated, err := service.Update(ctx, users[0], id, changed)
	require.NoError(t, err)
	assert.EqualValues(t, 20000, updated.AmountCents)
	old, err := service.List(ctx, users[0], date, date)
	require.NoError(t, err)
	assert.Empty(t, old)
	recoveredKeys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": key})
	require.NoError(t, err)
	recoveredCipher, err := NewCipher(recoveredKeys)
	require.NoError(t, err)
	recovered := NewService(NewRepository(pool), recoveredCipher)
	list, err = recovered.List(ctx, users[0], changed.PaidAt, changed.PaidAt)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.EqualValues(t, 20000, list[0].AmountCents)
	assert.ErrorIs(t, service.Delete(ctx, users[1], id), ErrNotFound)
	require.NoError(t, service.Delete(ctx, users[0], id))
	assert.ErrorIs(t, service.Delete(ctx, users[0], id), ErrNotFound)
	list, err = service.List(ctx, users[0], changed.PaidAt, changed.PaidAt)
	require.NoError(t, err)
	assert.Empty(t, list)

}
