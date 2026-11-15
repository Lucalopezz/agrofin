// Testa pagamentos repetidos, saldo e concorrência no PostgreSQL.
package pagamentos

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentsWithoutIdempotencyIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("configure TEST_DATABASE_URL com banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	defer pool.Close()
	var owner, employee string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO users(name,login_name,password_hash) VALUES ('Teste','pagamentos-'||gen_random_uuid()::text,'hash') RETURNING id::text`).Scan(&owner))
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, q := range []string{`DELETE FROM employee_payments WHERE user_id=$1`, `DELETE FROM employees WHERE user_id=$1`, `DELETE FROM users WHERE id=$1`} {
			_, err := pool.Exec(c, q, owner)
			assert.NoError(t, err)
		}
	}()
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO employees(user_id,name,active) VALUES($1,'Teste',true) RETURNING id::text`, owner).Scan(&employee))
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{7}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keys)
	require.NoError(t, err)
	harvest := harvestSummarizerFunc(func(context.Context, string, colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
		return []colheitas.HarvestSummary{{EmployeeID: employee, EmployeeName: "Teste", NormalAmountCents: 40000, HarvestAmountCents: 40000}}, nil
	})
	daily := dailyWorkSummarizerFunc(func(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
		return nil, nil
	})
	service := NewPayrollService(harvest, daily, NewRepository(pool), cipher)
	date := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	in := CreatePaymentInput{EmployeeID: employee, PeriodStart: date, PeriodEnd: date, AmountCents: 10000}
	first, err := service.Create(ctx, owner, in)
	require.NoError(t, err)
	second, err := service.Create(ctx, owner, in)
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, second.ID)
	summary, err := service.Summarize(ctx, owner, Period{StartDate: date, EndDate: date})
	require.NoError(t, err)
	require.Len(t, summary.Employees, 1)
	assert.EqualValues(t, 20000, summary.Employees[0].PaidAmountCents)
	// Duas requisições que disputam o restante não podem ultrapassar o devido.
	in.AmountCents = 20000
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Create(ctx, owner, in); errs <- err }()
	}
	wg.Wait()
	close(errs)
	success, conflict := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPaymentExceedsBalance) {
			conflict++
		} else {
			t.Errorf("erro inesperado: %v", err)
		}
	}
	assert.Equal(t, 1, success)
	assert.Equal(t, 1, conflict)
	summary, err = service.Summarize(ctx, owner, Period{StartDate: date, EndDate: date})
	require.NoError(t, err)
	assert.EqualValues(t, 40000, summary.Employees[0].PaidAmountCents)
	assert.Zero(t, summary.Employees[0].RemainingAmountCents)
}
