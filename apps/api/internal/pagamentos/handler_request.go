// Decodifica período de consulta e corpo idempotente de pagamento.
package pagamentos

import (
	"errors"
	"net/http"
	"net/url"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
)

type createPaymentRequest struct {
	EmployeeID  string `json:"employee_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	AmountCents int64  `json:"amount_cents"`
	Notes       string `json:"notes"`
}

func decodeCreatePaymentRequest(w http.ResponseWriter, r *http.Request) (CreatePaymentInput, bool) {
	// A chave identifica uma operação lógica: o cliente a reutiliza ao repetir o mesmo
	// envio após timeout e gera outra para um pagamento novo. Values permite rejeitar
	// cabeçalhos duplicados, que poderiam tornar essa identidade ambígua.
	keys := r.Header.Values("Idempotency-Key")
	if len(keys) != 1 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie uma única chave no cabeçalho Idempotency-Key"})
		return CreatePaymentInput{}, false
	}
	requestKey := keys[0]

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
	return CreatePaymentInput{
		EmployeeID: request.EmployeeID, PeriodStart: startDate, PeriodEnd: endDate,
		AmountCents: request.AmountCents, Notes: request.Notes, RequestKey: requestKey,
	}, true
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
	return Period{StartDate: startDate, EndDate: endDate}, true
}
