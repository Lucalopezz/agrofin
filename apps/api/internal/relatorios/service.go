// Consolida projeções dos domínios sem duplicar seus cálculos e cifras.
package relatorios

import (
	"context"
	"errors"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/despesas"
	"github.com/lucalopezz/agrofin/apps/api/internal/pagamentos"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/vendas"
)

type payrollReader interface {
	SummarizeDue(context.Context, string, pagamentos.Period) (pagamentos.PeriodSummary, error)
	ListPaid(context.Context, string, time.Time, time.Time) ([]pagamentos.PaidPayment, error)
}

type salesReader interface {
	ListSold(context.Context, string, vendas.SalePeriod) ([]vendas.Sale, error)
	ListReceived(context.Context, string, time.Time, time.Time) ([]vendas.ReceiptRecord, error)
}

type expenseReader interface {
	List(context.Context, string, time.Time, time.Time) ([]despesas.Expense, error)
}

func Summarize(
	ctx context.Context, userID string, period Period, location *time.Location,
	payroll payrollReader, sales salesReader, expenses expenseReader,
) (Summary, error) {
	harvest, team, err := summarizeHarvestAndDue(ctx, userID, period, payroll)
	if err != nil {
		return Summary{}, err
	}
	sold, err := summarizeSales(ctx, userID, period, sales)
	if err != nil {
		return Summary{}, err
	}
	cash, paidCents, err := summarizeCash(ctx, userID, period, location, payroll, sales, expenses)
	if err != nil {
		return Summary{}, err
	}
	team.PaidCents = paidCents

	return Summary{
		StartDate: period.StartDate.Format(date.Layout),
		EndDate:   period.EndDate.Format(date.Layout),
		TimeZone:  location.String(),
		Harvest:   harvest,
		Payroll:   team,
		Sales:     sold,
		Cash:      cash,
	}, ctx.Err()
}

func summarizeHarvestAndDue(
	ctx context.Context, userID string, period Period, payroll payrollReader,
) (HarvestSummary, PayrollSummary, error) {
	due, err := payroll.SummarizeDue(ctx, userID, pagamentos.Period(period))
	if err != nil {
		return HarvestSummary{}, PayrollSummary{}, err
	}

	// A folha agrupa colheitas e diárias por funcionário, então uma leitura
	// alimenta tanto as quantidades colhidas quanto o valor devido.
	harvest := HarvestSummary{ByEmployee: []EmployeeHarvest{}}
	payrollSummary := PayrollSummary{ByEmployee: []EmployeeDue{}}
	var sum totals
	for _, employee := range due.Employees {
		sum.add(&harvest.NormalBoxes, employee.NormalBoxes)
		sum.add(&harvest.MillingBoxes, employee.MillingBoxes)
		sum.add(&payrollSummary.DueCents, employee.DueAmountCents)
		if employee.NormalBoxes > 0 || employee.MillingBoxes > 0 {
			harvest.ByEmployee = append(harvest.ByEmployee, EmployeeHarvest{
				EmployeeID: employee.EmployeeID, EmployeeName: employee.EmployeeName,
				NormalBoxes: employee.NormalBoxes, MillingBoxes: employee.MillingBoxes,
			})
		}
		payrollSummary.ByEmployee = append(payrollSummary.ByEmployee, EmployeeDue{
			EmployeeID: employee.EmployeeID, EmployeeName: employee.EmployeeName,
			HarvestCents: employee.HarvestAmountCents, DailyWorkCents: employee.DailyWorkAmountCents,
			DueCents: employee.DueAmountCents,
		})
	}
	sum.add(&harvest.TotalBoxes, harvest.NormalBoxes)
	sum.add(&harvest.TotalBoxes, harvest.MillingBoxes)
	return harvest, payrollSummary, sum.err
}

func summarizeSales(ctx context.Context, userID string, period Period, sales salesReader) (SalesSummary, error) {
	sold, err := sales.ListSold(ctx, userID, vendas.SalePeriod(period))
	if err != nil {
		return SalesSummary{}, err
	}

	summary := SalesSummary{BoxesByType: map[string]int64{
		string(vendas.SaleTypeExportacao): 0,
		string(vendas.SaleTypeMercado):    0,
		string(vendas.SaleTypeMoagem):     0,
	}}
	var sum totals
	for _, sale := range sold {
		sum.add(&summary.GrossCents, sale.GrossCents)
		sum.add(&summary.DiscountCents, sale.DiscountCents)
		sum.add(&summary.NetCents, sale.NetCents)
		for _, item := range sale.Items {
			key := string(item.SaleType)
			count := summary.BoxesByType[key]
			sum.add(&count, int64(item.Quantity))
			summary.BoxesByType[key] = count
			sum.add(&summary.TotalBoxes, int64(item.Quantity))
		}
	}
	if sum.err != nil {
		return SalesSummary{}, sum.err
	}
	summary.AverageNetCents = averagePrice(summary.NetCents, summary.TotalBoxes)
	return summary, nil
}

func summarizeCash(
	ctx context.Context, userID string, period Period, location *time.Location,
	payroll payrollReader, sales salesReader, expenses expenseReader,
) (CashSummary, int64, error) {
	// Pagamentos e recebimentos têm horário efetivo. Despesas seguem a data
	// civil informada no lançamento e recebem o período original.
	start, end := period.cashWindow(location)
	paid, err := payroll.ListPaid(ctx, userID, start, end)
	if err != nil {
		return CashSummary{}, 0, err
	}
	received, err := sales.ListReceived(ctx, userID, start, end)
	if err != nil {
		return CashSummary{}, 0, err
	}
	spent, err := expenses.List(ctx, userID, period.StartDate, period.EndDate)
	if err != nil {
		return CashSummary{}, 0, err
	}

	summary := CashSummary{PaidExpensesByCategory: make(map[string]int64)}
	var sum totals
	var paidCents int64
	for _, payment := range paid {
		sum.add(&paidCents, payment.AmountCents)
	}
	for _, receipt := range received {
		sum.add(&summary.ReceivedCents, receipt.AmountCents)
	}
	for _, expense := range spent {
		amount := summary.PaidExpensesByCategory[expense.Category]
		sum.add(&amount, expense.AmountCents)
		summary.PaidExpensesByCategory[expense.Category] = amount
		sum.add(&summary.PaidExpensesCents, expense.AmountCents)
	}

	if sum.err != nil {
		return CashSummary{}, 0, sum.err
	}
	// Cada parcela já cabe no limite exato do JSON, então a subtração
	// inteira intermediária cabe em int64 mesmo quando as duas saídas são altas.
	summary.BalanceCents = summary.ReceivedCents - paidCents - summary.PaidExpensesCents
	if summary.BalanceCents < -maxExactJSONInteger {
		return CashSummary{}, 0, errInvalidTotal
	}
	// A resposta também exibe o total pago à equipe. Reusar a mesma soma
	// mantém tanto esse indicador quanto o caixa ligados às mesmas saídas.
	return summary, paidCents, nil
}

func averagePrice(net, boxes int64) *int64 {
	if boxes == 0 {
		return nil
	}
	price := net / boxes
	// Comparar com o teto de metade do divisor evita duplicar o resto.
	if net%boxes >= boxes/2+boxes%2 {
		price++
	}
	return &price
}

// O limite mantém os números JSON exatos também no cliente JavaScript.
const maxExactJSONInteger int64 = 1<<53 - 1

var errInvalidTotal = errors.New("relatorios: valor acima do limite do resumo")

// totals sinaliza quando uma soma não pode ser usada no resultado.
type totals struct{ err error }

// add recebe parcelas não negativas dos domínios e verifica o limite do total.
func (t *totals) add(total *int64, amount int64) {
	if amount > maxExactJSONInteger-*total {
		t.err = errInvalidTotal
		return
	}
	*total += amount
}
