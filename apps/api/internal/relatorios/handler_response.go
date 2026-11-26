// Serializa detalhes públicos e oculta dados financeiros em respostas de erro.
package relatorios

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	"github.com/lucalopezz/agrofin/apps/api/internal/pagamentos"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/vendas"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeResult(w http.ResponseWriter, value any, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "Tempo para consultar o período esgotado")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "Requisição cancelada")
	case err != nil:
		log.Print("erro ao consultar dados do relatório")
		writeError(w, http.StatusInternalServerError, "Não foi possível consultar o período")
	default:
		writeJSON(w, http.StatusOK, value)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("erro ao escrever resposta do relatório")
	}
}

type paymentResponse struct {
	ID           string `json:"id"`
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	PaidAt       string `json:"paid_at"`
	AmountCents  int64  `json:"amount_cents"`
	Notes        string `json:"notes,omitempty"`
}

func newPaymentResponse(row pagamentos.PaidPayment) paymentResponse {
	return paymentResponse{
		ID: row.ID, EmployeeID: row.EmployeeID, EmployeeName: row.EmployeeName,
		StartDate: row.PeriodStart.Format(date.Layout),
		// O banco usa limite exclusivo, mas a tela exibe o último dia trabalhado.
		EndDate:     row.PeriodEnd.AddDate(0, 0, -1).Format(date.Layout),
		PaidAt:      row.PaidAt.UTC().Format(time.RFC3339Nano),
		AmountCents: row.AmountCents, Notes: row.Notes,
	}
}

type receiptResponse struct {
	ID          string `json:"id"`
	SaleID      string `json:"sale_id"`
	SoldAt      string `json:"sold_at"`
	ReceivedAt  string `json:"received_at"`
	AmountCents int64  `json:"amount_cents"`
}

func newReceiptResponse(row vendas.ReceiptRecord) receiptResponse {
	return receiptResponse{
		ID: row.ID, SaleID: row.SaleID, SoldAt: row.SoldAt.Format(date.Layout),
		ReceivedAt: row.ReceivedAt.UTC().Format(time.RFC3339Nano), AmountCents: row.AmountCents,
	}
}

type dailyWorkResponse struct {
	ID           string `json:"id"`
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	WorkDate     string `json:"work_date"`
	AmountCents  int64  `json:"amount_cents"`
	Notes        string `json:"notes,omitempty"`
}

func newDailyWorkResponse(row diarias.WorkRecord) dailyWorkResponse {
	return dailyWorkResponse{
		ID: row.ID, EmployeeID: row.EmployeeID, EmployeeName: row.EmployeeName,
		WorkDate: row.WorkDate.Format(date.Layout), AmountCents: row.AmountCents, Notes: row.Notes,
	}
}

type soldSaleResponse struct {
	ID         string             `json:"id"`
	SoldAt     string             `json:"sold_at"`
	BuyerName  string             `json:"buyer_name,omitempty"`
	Items      []soldItemResponse `json:"items"`
	Discount   *discountResponse  `json:"discount,omitempty"`
	GrossCents int64              `json:"gross_amount_cents"`
	NetCents   int64              `json:"net_amount_cents"`
}

type soldItemResponse struct {
	Type      vendas.SaleType `json:"sale_type"`
	Quantity  int32           `json:"quantity"`
	UnitCents int64           `json:"unit_price_cents"`
	LineCents int64           `json:"line_amount_cents"`
}

type discountResponse struct {
	Reason      string `json:"reason"`
	AmountCents int64  `json:"amount_cents"`
}

func newSoldSaleResponse(row vendas.Sale) soldSaleResponse {
	result := soldSaleResponse{
		ID: row.ID, SoldAt: row.SoldAt.Format(date.Layout), BuyerName: row.BuyerName,
		GrossCents: row.GrossCents, NetCents: row.NetCents,
		Items: mapRows(row.Items, func(item vendas.SaleItem) soldItemResponse {
			return soldItemResponse{
				Type: item.SaleType, Quantity: item.Quantity,
				UnitCents: item.UnitPriceCents, LineCents: item.LineAmountCents,
			}
		}),
	}
	if row.Discount != nil {
		result.Discount = &discountResponse{Reason: row.Discount.Reason, AmountCents: row.Discount.AmountCents}
	}
	return result
}
