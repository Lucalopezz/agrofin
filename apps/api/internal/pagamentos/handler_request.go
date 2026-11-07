package pagamentos

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const periodDateLayout = "2006-01-02"

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

	startDate, err := parsePeriodDate(startValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "start_date deve usar o formato AAAA-MM-DD"})
		return Period{}, false
	}
	endDate, err := parsePeriodDate(endValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve usar o formato AAAA-MM-DD"})
		return Period{}, false
	}
	return Period{StartDate: startDate, EndDate: endDate}, true
}

func parsePeriodDate(value string) (time.Time, error) {
	if len(value) != len(periodDateLayout) || strings.TrimSpace(value) != value {
		return time.Time{}, ErrInvalidPayrollPeriod
	}
	date, err := time.Parse(periodDateLayout, value)
	if err != nil || date.Year() < 1 || date.Format(periodDateLayout) != value {
		return time.Time{}, ErrInvalidPayrollPeriod
	}
	return date, nil
}
