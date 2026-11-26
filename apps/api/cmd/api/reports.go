// Compõe as leituras do relatório na mesma visão do PostgreSQL.
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
	// O fuso do relatório também precisa funcionar em imagens sem a base IANA.
	_ "time/tzdata"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
	"github.com/lucalopezz/agrofin/apps/api/internal/despesas"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	"github.com/lucalopezz/agrofin/apps/api/internal/pagamentos"
	"github.com/lucalopezz/agrofin/apps/api/internal/relatorios"
	"github.com/lucalopezz/agrofin/apps/api/internal/vendas"
)

type reportCiphers struct {
	harvest   *colheitas.RateCipher
	dailyWork *diarias.Cipher
	payments  *pagamentos.Cipher
	sales     *vendas.Cipher
	expenses  *despesas.Cipher
}

func registerReports(
	mux *http.ServeMux, pool *pgxpool.Pool, ciphers reportCiphers,
	payroll *pagamentos.PayrollService, sales *vendas.Service, dailyWork *diarias.Service,
	requireSession func(http.Handler) http.Handler,
) error {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return fmt.Errorf("carregar fuso dos relatórios: %w", err)
	}
	summarize := func(ctx context.Context, userID string, period relatorios.Period) (relatorios.Summary, error) {
		// Uma visão única impede que os cartões da mesma resposta misturem
		// lançamentos anteriores e posteriores a uma alteração concorrente.
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
		if err != nil {
			return relatorios.Summary{}, err
		}
		defer func() {
			// A conexão precisa ser liberada mesmo se o navegador cancelar a leitura.
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}()

		dailyWork := diarias.NewService(diarias.NewRepository(tx), ciphers.dailyWork)
		harvest := colheitas.NewHarvestService(colheitas.NewRepository(tx), ciphers.harvest)
		payroll := pagamentos.NewPayrollService(harvest, dailyWork, pagamentos.NewRepository(tx), ciphers.payments)
		sales := vendas.NewService(vendas.NewRepository(tx), ciphers.sales)
		expenses := despesas.NewService(despesas.NewRepository(tx), ciphers.expenses)

		summary, err := relatorios.Summarize(ctx, userID, period, location, payroll, sales, expenses)
		if err != nil {
			return relatorios.Summary{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return relatorios.Summary{}, err
		}
		return summary, nil
	}
	relatorios.RegisterRoutes(mux, relatorios.NewHandler(summarize, payroll, sales, dailyWork, location), requireSession)
	return nil
}
