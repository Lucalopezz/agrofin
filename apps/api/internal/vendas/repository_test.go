// Testa vendas, recebimentos e proteção financeira no PostgreSQL de teste.
package vendas

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSalesRepositoryIntegration(t *testing.T) {
	// Nunca usa o banco da aplicação: o endereço de teste deve ser explícito.
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("configure TEST_DATABASE_URL com um banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	defer pool.Close()
	var users []string
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, table := range []string{"sale_receipts", "sale_discounts", "sale_items", "sales", "users"} {
			column := "user_id"
			if table == "users" {
				column = "id"
			}
			_, err := pool.Exec(cleanup, "DELETE FROM "+table+" WHERE "+column+"::text = ANY($1)", users)
			assert.NoError(t, err)
		}
	}()
	for range 2 {
		var id string
		err := pool.QueryRow(ctx, `INSERT INTO users(name,login_name,password_hash) VALUES ('Teste vendas','teste-vendas-' || gen_random_uuid()::text,'hash') RETURNING id::text`).Scan(&id)
		require.NoError(t, err)
		users = append(users, id)
	}
	key := bytes.Repeat([]byte{7}, 32)
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": key})
	require.NoError(t, err)
	cipher, err := NewCipher(keys)
	require.NoError(t, err)
	service := NewService(NewRepository(pool), cipher)
	sale, err := service.Create(ctx, users[0], exampleSale())
	require.NoError(t, err)
	assert.EqualValues(t, 53000, sale.ReceivableCents)
	input := CreateReceiptInput{SaleID: sale.ID, AmountCents: 30000}
	_, err = service.Receive(ctx, users[0], input)
	require.NoError(t, err)
	input.AmountCents = 23001
	_, err = service.Receive(ctx, users[0], input)
	assert.ErrorIs(t, err, ErrReceiptExceedsBalance)
	_, err = service.Receive(ctx, users[1], input)
	assert.ErrorIs(t, err, ErrSaleNotFound)
	period := SalePeriod{StartDate: sale.SoldAt, EndDate: sale.SoldAt}
	list, err := service.List(ctx, users[0], period)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.EqualValues(t, 55000, list[0].GrossCents)
	assert.EqualValues(t, 53000, list[0].NetCents)
	assert.EqualValues(t, 30000, list[0].ReceivedCents)
	assert.EqualValues(t, 23000, list[0].ReceivableCents)
	other, err := service.List(ctx, users[1], period)
	require.NoError(t, err)
	assert.Empty(t, other)
	outside, err := service.List(ctx, users[0], SalePeriod{StartDate: sale.SoldAt.AddDate(0, 0, 1), EndDate: sale.SoldAt.AddDate(0, 0, 1)})
	require.NoError(t, err)
	assert.Empty(t, outside)

	// Verifica os três tipos de envelope persistido e a recuperação pela chave histórica.
	for _, check := range []struct {
		query string
		cents int64
	}{
		{`SELECT unit_price_ciphertext,unit_price_nonce,unit_price_key_version FROM sale_items WHERE sale_id=$1 AND sale_type='exportacao'`, 4000},
		{`SELECT amount_ciphertext,amount_nonce,amount_key_version FROM sale_discounts WHERE sale_id=$1`, 2000},
		{`SELECT amount_ciphertext,amount_nonce,amount_key_version FROM sale_receipts WHERE sale_id=$1`, 30000},
	} {
		var ciphertext, nonce []byte
		var version string
		require.NoError(t, pool.QueryRow(ctx, check.query, sale.ID).Scan(&ciphertext, &nonce, &version))
		var plain [8]byte
		binary.BigEndian.PutUint64(plain[:], uint64(check.cents))
		assert.NotContains(t, ciphertext, plain[:])
		assert.Len(t, nonce, 12)
		assert.Equal(t, "v1", version)
	}
	recoveredKeys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": key})
	require.NoError(t, err)
	recoveredCipher, err := NewCipher(recoveredKeys)
	require.NoError(t, err)
	recovered, err := NewService(NewRepository(pool), recoveredCipher).List(ctx, users[0], period)
	require.NoError(t, err)
	require.Len(t, recovered, 1)
	assert.EqualValues(t, 30000, recovered[0].ReceivedCents)

	// Duas tentativas diferentes disputam o saldo: apenas uma pode concluí-lo.
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := service.Receive(ctx, users[0], CreateReceiptInput{SaleID: sale.ID, AmountCents: 23000})
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	successes, conflicts := 0, 0
	for e := range errs {
		if e == nil {
			successes++
		} else if errors.Is(e, ErrReceiptExceedsBalance) {
			conflicts++
		} else {
			t.Errorf("erro inesperado: %v", e)
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, conflicts)
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sale_receipts WHERE sale_id=$1`, sale.ID).Scan(&count))
	assert.Equal(t, 2, count)

	// Envios iguais que cabem no saldo são eventos distintos.
	second, err := service.Create(ctx, users[0], exampleSale())
	require.NoError(t, err)
	firstReceipt, err := service.Receive(ctx, users[0], CreateReceiptInput{SaleID: second.ID, AmountCents: 10000})
	require.NoError(t, err)
	secondReceipt, err := service.Receive(ctx, users[0], CreateReceiptInput{SaleID: second.ID, AmountCents: 10000})
	require.NoError(t, err)
	assert.NotEqual(t, firstReceipt.ID, secondReceipt.ID)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sale_receipts WHERE sale_id=$1`, second.ID).Scan(&count))
	assert.Equal(t, 2, count)
}
