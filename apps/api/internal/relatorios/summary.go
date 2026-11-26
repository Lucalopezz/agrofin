// Define a projeção de produção, vendas e caixa para um período.
package relatorios

import "time"

type Period struct {
	StartDate time.Time
	EndDate   time.Time
}

type Summary struct {
	StartDate string         `json:"start_date"`
	EndDate   string         `json:"end_date"`
	TimeZone  string         `json:"time_zone"`
	Harvest   HarvestSummary `json:"harvest"`
	Sales     SalesSummary   `json:"sales"`
	Payroll   PayrollSummary `json:"payroll"`
	Cash      CashSummary    `json:"cash"`
}

type HarvestSummary struct {
	TotalBoxes   int64             `json:"total_boxes"`
	NormalBoxes  int64             `json:"normal_boxes"`
	MillingBoxes int64             `json:"milling_boxes"`
	ByEmployee   []EmployeeHarvest `json:"by_employee"`
}

type EmployeeHarvest struct {
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	NormalBoxes  int64  `json:"normal_boxes"`
	MillingBoxes int64  `json:"milling_boxes"`
}

type SalesSummary struct {
	TotalBoxes      int64            `json:"total_boxes"`
	BoxesByType     map[string]int64 `json:"boxes_by_type"`
	GrossCents      int64            `json:"gross_amount_cents"`
	DiscountCents   int64            `json:"discount_amount_cents"`
	NetCents        int64            `json:"net_amount_cents"`
	AverageNetCents *int64           `json:"average_net_price_cents"`
}

type PayrollSummary struct {
	DueCents   int64         `json:"due_amount_cents"`
	PaidCents  int64         `json:"paid_amount_cents"`
	ByEmployee []EmployeeDue `json:"by_employee"`
}

type EmployeeDue struct {
	EmployeeID     string `json:"employee_id"`
	EmployeeName   string `json:"employee_name"`
	HarvestCents   int64  `json:"harvest_amount_cents"`
	DailyWorkCents int64  `json:"daily_work_amount_cents"`
	DueCents       int64  `json:"due_amount_cents"`
}

type CashSummary struct {
	ReceivedCents          int64            `json:"received_amount_cents"`
	PaidExpensesCents      int64            `json:"paid_expenses_amount_cents"`
	PaidExpensesByCategory map[string]int64 `json:"paid_expenses_by_category"`
	BalanceCents           int64            `json:"balance_amount_cents"`
}

// cashWindow reconstrói as meias-noites no fuso do negócio. As datas civis
// chegam à API em UTC apenas como representação do dia informado.
func (p Period) cashWindow(location *time.Location) (time.Time, time.Time) {
	midnight := func(day time.Time) time.Time {
		year, month, date := day.Date()
		start := time.Date(year, month, date, 0, 0, 0, 0, location)
		// No início do antigo horário de verão, a meia-noite de São Paulo
		// podia não existir. Se time.Date cair na véspera, use a transição,
		// que é o primeiro instante do dia pedido.
		if start.Day() != date {
			_, transition := start.ZoneBounds()
			if !transition.IsZero() && transition.Day() == date {
				return transition
			}
		}
		return start
	}
	return midnight(p.StartDate).UTC(), midnight(p.EndDate.AddDate(0, 0, 1)).UTC()
}
