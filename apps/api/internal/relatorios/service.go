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
	result := Summary{
		StartDate: period.StartDate.Format(date.Layout),
		EndDate:   period.EndDate.Format(date.Layout),
		TimeZone:  location.String(),
		Harvest:   HarvestSummary{ByEmployee: []EmployeeHarvest{}},
		Payroll:   PayrollSummary{ByEmployee: []EmployeeDue{}},
		Sales: SalesSummary{BoxesByType: map[string]int64{
			string(vendas.SaleTypeExportacao): 0,
			string(vendas.SaleTypeMercado):    0,
			string(vendas.SaleTypeMoagem):     0,
		}},
		Cash: CashSummary{PaidExpensesByCategory: make(map[string]int64)},
	}

	// Cada fonte usa a própria data de ocorrência. O intervalo trabalhado e
	// a data de venda não limitam os eventos de caixa.
	start, end := period.cashWindow(location)
	due, err := payroll.SummarizeDue(ctx, userID, pagamentos.Period(period))
	if err != nil {
		return Summary{}, err
	}
	sold, err := sales.ListSold(ctx, userID, vendas.SalePeriod(period))
	if err != nil {
		return Summary{}, err
	}
	paid, err := payroll.ListPaid(ctx, userID, start, end)
	if err != nil {
		return Summary{}, err
	}
	received, err := sales.ListReceived(ctx, userID, start, end)
	if err != nil {
		return Summary{}, err
	}
	spent, err := expenses.List(ctx, userID, period.StartDate, period.EndDate)
	if err != nil {
		return Summary{}, err
	}

	var sum totals
	for _, employee := range due.Employees {
		sum.add(&result.Harvest.NormalBoxes, employee.NormalBoxes)
		sum.add(&result.Harvest.MillingBoxes, employee.MillingBoxes)
		sum.add(&result.Payroll.DueCents, employee.DueAmountCents)
		if employee.NormalBoxes > 0 || employee.MillingBoxes > 0 {
			result.Harvest.ByEmployee = append(result.Harvest.ByEmployee, EmployeeHarvest{
				EmployeeID: employee.EmployeeID, EmployeeName: employee.EmployeeName,
				NormalBoxes: employee.NormalBoxes, MillingBoxes: employee.MillingBoxes,
			})
		}
		result.Payroll.ByEmployee = append(result.Payroll.ByEmployee, EmployeeDue{
			EmployeeID: employee.EmployeeID, EmployeeName: employee.EmployeeName,
			HarvestCents: employee.HarvestAmountCents, DailyWorkCents: employee.DailyWorkAmountCents,
			DueCents: employee.DueAmountCents,
		})
	}
	sum.add(&result.Harvest.TotalBoxes, result.Harvest.NormalBoxes)
	sum.add(&result.Harvest.TotalBoxes, result.Harvest.MillingBoxes)

	for _, sale := range sold {
		sum.add(&result.Sales.GrossCents, sale.GrossCents)
		sum.add(&result.Sales.DiscountCents, sale.DiscountCents)
		sum.add(&result.Sales.NetCents, sale.NetCents)
		for _, item := range sale.Items {
			key := string(item.SaleType)
			count, known := result.Sales.BoxesByType[key]
			if !known {
				return Summary{}, errors.New("relatorios: classificação de venda inválida")
			}
			sum.add(&count, int64(item.Quantity))
			result.Sales.BoxesByType[key] = count
			sum.add(&result.Sales.TotalBoxes, int64(item.Quantity))
		}
	}
	for _, payment := range paid {
		sum.add(&result.Payroll.PaidCents, payment.AmountCents)
	}
	for _, receipt := range received {
		sum.add(&result.Cash.ReceivedCents, receipt.AmountCents)
	}
	for _, expense := range spent {
		amount := result.Cash.PaidExpensesByCategory[expense.Category]
		sum.add(&amount, expense.AmountCents)
		result.Cash.PaidExpensesByCategory[expense.Category] = amount
		sum.add(&result.Cash.PaidExpensesCents, expense.AmountCents)
	}

	if sum.err != nil {
		return Summary{}, sum.err
	}
	// Cada parcela já cabe no limite exato do JSON, então a subtração
	// inteira intermediária cabe em int64 mesmo quando as duas saídas são altas.
	result.Cash.BalanceCents = result.Cash.ReceivedCents - result.Payroll.PaidCents - result.Cash.PaidExpensesCents
	if result.Cash.BalanceCents < -maxExactJSONInteger {
		return Summary{}, errInvalidTotal
	}
	result.Sales.AverageNetCents = averagePrice(result.Sales.NetCents, result.Sales.TotalBoxes)
	return result, ctx.Err()
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

var errInvalidTotal = errors.New("relatorios: valor inválido ou acima do limite do resumo")

// totals sinaliza quando uma soma não pode ser usada no resultado.
type totals struct{ err error }

func (t *totals) add(total *int64, amount int64) {
	if amount < 0 || amount > maxExactJSONInteger-*total {
		t.err = errInvalidTotal
		return
	}
	*total += amount
}
