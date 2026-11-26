// Liga o resumo e as consultas de conferência à sessão e ao período HTTP.
package relatorios

import (
	"context"
	"net/http"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/auth"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	"github.com/lucalopezz/agrofin/apps/api/internal/vendas"
)

type dailyWorkReader interface {
	List(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.WorkRecord, error)
}

type Handler struct {
	summary   func(context.Context, string, Period) (Summary, error)
	payroll   payrollReader
	sales     salesReader
	dailyWork dailyWorkReader
	location  *time.Location
}

func NewHandler(
	summary func(context.Context, string, Period) (Summary, error),
	payroll payrollReader, sales salesReader, dailyWork dailyWorkReader,
	location *time.Location,
) *Handler {
	return &Handler{summary: summary, payroll: payroll, sales: sales, dailyWork: dailyWork, location: location}
}

func periodHandler[T any](read func(context.Context, string, Period) (T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Autenticação necessária")
			return
		}
		period, ok := decodePeriod(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "Informe início e fim válidos no formato AAAA-MM-DD")
			return
		}
		value, err := read(r.Context(), principal.UserID, period)
		writeResult(w, value, err)
	}
}

func (h *Handler) payments(ctx context.Context, userID string, period Period) ([]paymentResponse, error) {
	start, end := period.cashWindow(h.location)
	rows, err := h.payroll.ListPaid(ctx, userID, start, end)
	return mapRows(rows, newPaymentResponse), err
}

func (h *Handler) receipts(ctx context.Context, userID string, period Period) ([]receiptResponse, error) {
	start, end := period.cashWindow(h.location)
	rows, err := h.sales.ListReceived(ctx, userID, start, end)
	return mapRows(rows, newReceiptResponse), err
}

func (h *Handler) dailyWorks(ctx context.Context, userID string, period Period) ([]dailyWorkResponse, error) {
	rows, err := h.dailyWork.List(ctx, userID, diarias.DailyWorkPeriod(period))
	return mapRows(rows, newDailyWorkResponse), err
}

func (h *Handler) soldSales(ctx context.Context, userID string, period Period) ([]soldSaleResponse, error) {
	rows, err := h.sales.ListSold(ctx, userID, vendas.SalePeriod(period))
	return mapRows(rows, newSoldSaleResponse), err
}

func mapRows[T, R any](rows []T, convert func(T) R) []R {
	result := make([]R, 0, len(rows))
	for _, row := range rows {
		result = append(result, convert(row))
	}
	return result
}
