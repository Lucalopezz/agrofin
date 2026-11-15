// Decodifica corpos e períodos das requisições de despesas.
package despesas

import (
	"errors"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
	"net/http"
	"net/url"
	"time"
)

type expenseRequest struct {
	PaidAt      string `json:"paid_at"`
	Category    string `json:"category"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
}

func decodeInput(w http.ResponseWriter, r *http.Request) (Input, bool) {
	v, err := httpjson.Decode[expenseRequest](w, r)
	switch {
	case errors.Is(err, httpjson.ErrUnsupportedMediaType):
		writeError(w, 415, "Envie o corpo como application/json")
		return Input{}, false
	case errors.Is(err, httpjson.ErrBodyTooLarge):
		writeError(w, 413, "Corpo da requisição excede 4 KiB")
		return Input{}, false
	case err != nil || v == nil:
		writeError(w, 400, "Envie um único objeto JSON com os dados da despesa")
		return Input{}, false
	}
	date, err := calendarDate.Parse(v.PaidAt)
	if err != nil {
		writeError(w, 400, "paid_at deve usar AAAA-MM-DD")
		return Input{}, false
	}
	return Input{PaidAt: date, Category: v.Category, Description: v.Description, AmountCents: v.AmountCents}, true
}
func decodePeriod(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 2 || len(q["start_date"]) != 1 || len(q["end_date"]) != 1 {
		writeError(w, 400, "Informe start_date e end_date uma única vez")
		return time.Time{}, time.Time{}, false
	}
	start, err := calendarDate.Parse(q.Get("start_date"))
	if err != nil {
		writeError(w, 400, "start_date deve usar AAAA-MM-DD")
		return time.Time{}, time.Time{}, false
	}
	end, err := calendarDate.Parse(q.Get("end_date"))
	if err != nil {
		writeError(w, 400, "end_date deve usar AAAA-MM-DD")
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}
