// Define períodos e estruturas de resposta do resumo de folha por funcionário.
package pagamentos

import (
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
)

// Period usa limites inclusivos explícitos enquanto o início oficial da semana
// de pagamento ainda não foi definido.
type Period struct {
	StartDate time.Time
	EndDate   time.Time
}

// EmployeeSummary é uma projeção de leitura; não cria outro agregado de escrita.
type EmployeeSummary struct {
	EmployeeID           string
	EmployeeName         string
	NormalBoxes          int64
	MillingBoxes         int64
	NormalAmountCents    int64
	MillingAmountCents   int64
	HarvestAmountCents   int64
	DailyWorkAmountCents int64
	DueAmountCents       int64
}

// PeriodSummary mostra o intervalo exato usado para produzir o total por pessoa.
type PeriodSummary struct {
	Period    Period
	Employees []EmployeeSummary
}

func (period Period) harvestPeriod() colheitas.HarvestPeriod {
	return colheitas.HarvestPeriod{StartDate: period.StartDate, EndDate: period.EndDate}
}
