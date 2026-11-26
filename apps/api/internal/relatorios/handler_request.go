// Valida os limites civis usados pelo resumo e pelos seus detalhes.
package relatorios

import (
	"net/http"
	"net/url"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

func decodePeriod(r *http.Request) (Period, bool) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(query) != 2 || len(query["start_date"]) != 1 || len(query["end_date"]) != 1 {
		return Period{}, false
	}
	start, startErr := date.Parse(query.Get("start_date"))
	end, endErr := date.Parse(query.Get("end_date"))
	if startErr != nil || endErr != nil || end.Before(start) {
		return Period{}, false
	}
	return Period{StartDate: start, EndDate: end}, true
}
