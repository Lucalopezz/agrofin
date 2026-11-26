// Verifica no PostgreSQL o resumo autenticado e as datas de cada lançamento.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
	"github.com/lucalopezz/agrofin/apps/api/internal/despesas"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/lucalopezz/agrofin/apps/api/internal/pagamentos"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
	"github.com/lucalopezz/agrofin/apps/api/internal/relatorios"
	"github.com/lucalopezz/agrofin/apps/api/internal/vendas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodReportIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("configure TEST_DATABASE_URL com banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	defer pool.Close()

	var owners []string
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, table := range []string{
			"employee_payments", "harvest_items", "harvests", "harvest_rates", "daily_work",
			"sale_receipts", "sale_discounts", "sale_items", "sales", "expenses", "employees", "users",
		} {
			column := "user_id"
			if table == "users" {
				column = "id"
			}
			_, err := pool.Exec(cleanup, "DELETE FROM "+table+" WHERE "+column+"::text = ANY($1)", owners)
			assert.NoError(t, err)
		}
	}()
	for range 2 {
		var owner string
		err := pool.QueryRow(ctx, `
			INSERT INTO users(name,login_name,password_hash)
			VALUES ('Relatório','relatorio-'||gen_random_uuid()::text,'hash')
			RETURNING id::text
		`).Scan(&owner)
		require.NoError(t, err)
		owners = append(owners, owner)
	}
	owner, other := owners[0], owners[1]
	var employee string
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO employees(user_id,name) VALUES ($1,'Maria') RETURNING id::text
	`, owner).Scan(&employee))

	ciphers := newReportTestCiphers(t)
	harvestRepo := colheitas.NewRepository(pool)
	harvest := colheitas.NewHarvestService(harvestRepo, ciphers.harvest)
	rates := colheitas.NewService(harvestRepo, ciphers.harvest)
	works := diarias.NewService(diarias.NewRepository(pool), ciphers.dailyWork)
	payroll := pagamentos.NewPayrollService(harvest, works, pagamentos.NewRepository(pool), ciphers.payments)
	sales := vendas.NewService(vendas.NewRepository(pool), ciphers.sales)
	expenses := despesas.NewService(despesas.NewRepository(pool), ciphers.expenses)

	saoPaulo, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	now := time.Now().In(saoPaulo)
	worked := date.Normalize(now).AddDate(0, 0, -7)
	_, err = rates.CreateRate(ctx, owner, colheitas.CreateRateInput{
		BoxType: colheitas.BoxTypeNormal, ValidFrom: worked, AmountCents: 1000,
	})
	require.NoError(t, err)
	_, err = harvest.Create(ctx, owner, colheitas.CreateHarvestInput{
		EmployeeID: employee, HarvestDate: worked,
		Items: []colheitas.HarvestItemInput{{BoxType: colheitas.BoxTypeNormal, Quantity: 10}},
	})
	require.NoError(t, err)
	_, err = works.Create(ctx, owner, diarias.CreateDailyWorkInput{
		EmployeeID: employee, WorkDate: worked, AmountCents: 4000,
	})
	require.NoError(t, err)
	saleInput := vendas.CreateSaleInput{
		SoldAt: worked,
		Items: []vendas.SaleItemInput{
			{SaleType: vendas.SaleTypeExportacao, Quantity: 8, UnitPriceCents: 4500},
			{SaleType: vendas.SaleTypeMercado, Quantity: 2, UnitPriceCents: 3000},
		},
		Discount: &vendas.SaleDiscountInput{AmountCents: 2000, Reason: "Ajuste de qualidade"},
	}
	sale, err := sales.Create(ctx, owner, saleInput)
	require.NoError(t, err)
	receipt, err := sales.Receive(ctx, owner, vendas.CreateReceiptInput{SaleID: sale.ID, AmountCents: 25000})
	require.NoError(t, err)

	inLocalTime := receipt.ReceivedAt.In(saoPaulo)
	cashDay := date.Normalize(inLocalTime)
	year, month, day := inLocalTime.Date()
	cashStart := time.Date(year, month, day, 0, 0, 0, 0, saoPaulo)
	firstPayment := insertReportPayment(t, ctx, pool, ciphers.payments, pagamentos.Payment{
		UserID: owner, EmployeeID: employee, PeriodStart: worked,
		PeriodEnd: worked.AddDate(0, 0, 1), PaidAt: cashStart.UTC(), AmountCents: 6000,
	})
	// A meia-noite seguinte pertence ao período seguinte, ainda que a origem seja a mesma.
	insertReportPayment(t, ctx, pool, ciphers.payments, pagamentos.Payment{
		UserID: owner, EmployeeID: employee, PeriodStart: worked,
		PeriodEnd: worked.AddDate(0, 0, 1), PaidAt: cashStart.AddDate(0, 0, 1).UTC(), AmountCents: 1000,
	})
	for userID, amount := range map[string]int64{owner: 9000, other: 32000} {
		_, err := expenses.Create(ctx, userID, despesas.Input{
			PaidAt: cashDay, Category: "Energia", AmountCents: amount,
		})
		require.NoError(t, err)
	}
	_, err = sales.Create(ctx, other, saleInput)
	require.NoError(t, err)
	// O relatório inclui fatos do funcionário mesmo depois que ele é desativado.
	_, err = pool.Exec(ctx, "UPDATE employees SET active=false, deactivated_at=now() WHERE id=$1", employee)
	require.NoError(t, err)

	mux := http.NewServeMux()
	require.NoError(t, registerReports(mux, pool, ciphers, payroll, sales, works,
		func(handler http.Handler) http.Handler { return handler }))
	get := func(path, userID string) *httptest.ResponseRecorder {
		return getReport(t, mux, path, userID)
	}
	inPeriod := func(day time.Time) string {
		value := day.Format(date.Layout)
		return "?start_date=" + value + "&end_date=" + value
	}
	read := func(userID string, day time.Time) relatorios.Summary {
		res := get("/reports/period-summary"+inPeriod(day), userID)
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
		var summary relatorios.Summary
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &summary))
		return summary
	}

	workSummary := read(owner, worked)
	require.Len(t, workSummary.Harvest.ByEmployee, 1)
	assert.Equal(t, employee, workSummary.Harvest.ByEmployee[0].EmployeeID)
	assert.EqualValues(t, 10, workSummary.Harvest.TotalBoxes)
	assert.EqualValues(t, 14000, workSummary.Payroll.DueCents)
	assert.EqualValues(t, 4000, workSummary.Payroll.ByEmployee[0].DailyWorkCents)
	assert.EqualValues(t, 42000, workSummary.Sales.GrossCents)
	assert.EqualValues(t, 2000, workSummary.Sales.DiscountCents)
	assert.EqualValues(t, 40000, workSummary.Sales.NetCents)
	require.NotNil(t, workSummary.Sales.AverageNetCents)
	assert.EqualValues(t, 4000, *workSummary.Sales.AverageNetCents)
	assert.Zero(t, workSummary.Cash.BalanceCents)

	cashSummary := read(owner, cashDay)
	assert.Zero(t, cashSummary.Payroll.DueCents)
	assert.Zero(t, cashSummary.Sales.NetCents)
	assert.Nil(t, cashSummary.Sales.AverageNetCents)
	assert.EqualValues(t, 25000, cashSummary.Cash.ReceivedCents)
	assert.EqualValues(t, 6000, cashSummary.Payroll.PaidCents)
	assert.EqualValues(t, 9000, cashSummary.Cash.PaidExpensesByCategory["Energia"])
	assert.EqualValues(t, 10000, cashSummary.Cash.BalanceCents)
	assert.EqualValues(t, -1000, read(owner, cashDay.AddDate(0, 0, 1)).Cash.BalanceCents)

	otherSummary := read(other, cashDay)
	assert.Zero(t, otherSummary.Cash.ReceivedCents)
	assert.Zero(t, otherSummary.Payroll.PaidCents)
	assert.EqualValues(t, -32000, otherSummary.Cash.BalanceCents)
	empty := read(other, worked.AddDate(0, 0, -1))
	assert.Empty(t, empty.Harvest.ByEmployee)
	assert.NotNil(t, empty.Harvest.ByEmployee)
	assert.Nil(t, empty.Sales.AverageNetCents)

	for _, check := range []struct {
		path, id string
		day      time.Time
	}{
		{"/sale-receipts", receipt.ID, cashDay},
		{"/employee-payments", firstPayment, cashDay},
		{"/reports/sold-sales", sale.ID, worked},
	} {
		res := get(check.path+inPeriod(check.day), owner)
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		assert.Contains(t, res.Body.String(), check.id)
		assert.NotContains(t, get(check.path+inPeriod(check.day), other).Body.String(), check.id)
	}
	assert.Contains(t, get("/daily-work"+inPeriod(worked), owner).Body.String(), employee)
	original, err := payroll.Summarize(ctx, owner, pagamentos.Period{StartDate: worked, EndDate: worked})
	require.NoError(t, err)
	require.Len(t, original.Employees, 1)
	assert.EqualValues(t, 7000, original.Employees[0].RemainingAmountCents)
}

func TestPeriodReportRejectsUnauthorizedAndInvalidPeriods(t *testing.T) {
	mux := http.NewServeMux()
	require.NoError(t, registerReports(mux, nil, reportCiphers{}, nil, nil, nil,
		func(handler http.Handler) http.Handler { return handler }))
	const route = "/reports/period-summary"
	assert.Equal(t, http.StatusUnauthorized, getReport(t, mux, route, "").Code)
	for _, query := range []string{
		"", "?start_date=2026-11-05", "?start_date=2026-11-05&end_date=2026-11-04",
		"?start_date=2026-02-30&end_date=2026-11-05",
		"?start_date=2026-11-05&end_date=2026-11-05&end_date=2026-11-06",
		"?start_date=2026-11-05&end_date=2026-11-05&user_id=outra-conta",
	} {
		assert.Equal(t, http.StatusBadRequest, getReport(t, mux, route+query, "sessao").Code, query)
	}
}

func getReport(t *testing.T, mux *http.ServeMux, path, userID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if userID != "" {
		req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: userID}))
	}
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	return res
}

func newReportTestCiphers(t *testing.T) reportCiphers {
	t.Helper()
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{7}, 32)})
	require.NoError(t, err)
	harvest, err := colheitas.NewRateCipherWithKeyring(keys)
	require.NoError(t, err)
	dailyWork, err := diarias.NewCipher(keys)
	require.NoError(t, err)
	payments, err := pagamentos.NewCipher(keys)
	require.NoError(t, err)
	sales, err := vendas.NewCipher(keys)
	require.NoError(t, err)
	expenses, err := despesas.NewCipher(keys)
	require.NoError(t, err)
	return reportCiphers{harvest: harvest, dailyWork: dailyWork, payments: payments, sales: sales, expenses: expenses}
}

func insertReportPayment(
	t *testing.T, ctx context.Context, pool *pgxpool.Pool, cipher *pagamentos.Cipher, payment pagamentos.Payment,
) string {
	t.Helper()
	// A cifra autentica o instante registrado. Gravar o envelope com um instante
	// conhecido permite testar a meia-noite sem alterar o relógio de produção.
	var err error
	payment.ID, err = uuid.New()
	require.NoError(t, err)
	p, err := cipher.EncryptPaymentAmount(payment)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO employee_payments
		  (id,user_id,employee_id,period_start,period_end,paid_at,amount_ciphertext,amount_nonce,amount_key_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, p.ID, p.UserID, p.EmployeeID, p.PeriodStart, p.PeriodEnd, p.PaidAt,
		p.Amount.Ciphertext, p.Amount.Nonce, p.Amount.KeyVersion)
	require.NoError(t, err)
	return p.ID
}
