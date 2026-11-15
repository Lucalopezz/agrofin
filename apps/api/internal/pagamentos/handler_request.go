// Decodifica período de consulta e corpo de pagamento.
package pagamentos

import (
	"errors"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
	"net/http"
	"net/url"
	"strings"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
)

type createPaymentRequest struct {
	EmployeeID  string `json:"employee_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	AmountCents int64  `json:"amount_cents"`
	Notes       string `json:"notes"`
}

func decodeCreatePaymentRequest(w http.ResponseWriter, r *http.Request, userID string) (CreatePaymentInput, bool) {
	request, err := httpjson.Decode[createPaymentRequest](w, r)
	if errors.Is(err, httpjson.ErrUnsupportedMediaType) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return CreatePaymentInput{}, false
	}
	if errors.Is(err, httpjson.ErrBodyTooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreatePaymentInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON com os campos do pagamento"})
		return CreatePaymentInput{}, false
	}
	startDate, err := calendarDate.Parse(request.StartDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "start_date deve usar o formato AAAA-MM-DD"})
		return CreatePaymentInput{}, false
	}
	endDate, err := calendarDate.Parse(request.EndDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve usar o formato AAAA-MM-DD"})
		return CreatePaymentInput{}, false
	}
	input := CreatePaymentInput{EmployeeID: request.EmployeeID, PeriodStart: startDate, PeriodEnd: endDate, AmountCents: request.AmountCents, Notes: request.Notes}
	payment, err := normalizeCreatePayment(userID, input)
	if err != nil {
		writeCreatePaymentResponse(w, Payment{}, err)
		return CreatePaymentInput{}, false
	}
	return CreatePaymentInput{EmployeeID: payment.EmployeeID, PeriodStart: payment.PeriodStart, PeriodEnd: payment.PeriodEnd.AddDate(0, 0, -1), AmountCents: payment.AmountCents, Notes: payment.Notes}, true
}

func decodePeriodQuery(w http.ResponseWriter, r *http.Request) (Period, bool) {
	// Exigir exatamente os dois limites evita uma leitura acidental de todo o histórico.
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Parâmetros de consulta inválidos"})
		return Period{}, false
	}
	startValues, hasStart := query["start_date"]
	endValues, hasEnd := query["end_date"]
	if len(query) != 2 || !hasStart || !hasEnd || len(startValues) != 1 || len(endValues) != 1 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Informe start_date e end_date uma única vez"})
		return Period{}, false
	}

	startDate, err := calendarDate.Parse(startValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "start_date deve usar o formato AAAA-MM-DD"})
		return Period{}, false
	}
	endDate, err := calendarDate.Parse(endValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve usar o formato AAAA-MM-DD"})
		return Period{}, false
	}
	if endDate.Before(startDate) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve ser igual ou posterior a start_date"})
		return Period{}, false
	}
	return Period{StartDate: startDate, EndDate: endDate}, true
}

// normalizeCreatePayment valida e normaliza os dados recebidos no HTTP.
func normalizeCreatePayment(userID string, input CreatePaymentInput) (Payment, error) {
	ownerID, err := uuid.Normalize(userID)
	if err != nil {
		return Payment{}, ErrInvalidPayrollOwner
	}
	employeeID, err := uuid.Normalize(input.EmployeeID)
	if err != nil {
		return Payment{}, ErrInvalidPaymentEmployee
	}
	start, endInclusive, err := calendarDate.NormalizeInclusivePeriod(input.PeriodStart, input.PeriodEnd)
	if err != nil {
		return Payment{}, ErrInvalidPaymentPeriod
	}
	amountCents, err := normalizePaymentAmount(input.AmountCents)
	if err != nil {
		return Payment{}, err
	}

	return Payment{
		UserID: ownerID, EmployeeID: employeeID,
		PeriodStart: start, PeriodEnd: endInclusive.AddDate(0, 0, 1),
		AmountCents: amountCents, Notes: strings.TrimSpace(input.Notes),
	}, nil
}

// normalizePaymentAmount aplica a regra compartilhada e preserva o erro do domínio de pagamentos.
func normalizePaymentAmount(amountCents int64) (int64, error) {
	if !money.IsPositiveCents(amountCents) {
		return 0, ErrInvalidPaymentAmount
	}
	return amountCents, nil
}
