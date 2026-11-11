// Decodifica JSON e datas no formato aceito pelo endpoint de diárias.
package diarias

import (
	"errors"
	"net/http"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
)

type createDailyWorkRequest struct {
	EmployeeID  string `json:"employee_id"`
	WorkDate    string `json:"work_date"`
	AmountCents int64  `json:"amount_cents"`
	Notes       string `json:"notes"`
}

func decodeCreateDailyWorkRequest(w http.ResponseWriter, r *http.Request) (CreateDailyWorkInput, bool) {
	request, err := httpjson.Decode[createDailyWorkRequest](w, r)
	if errors.Is(err, httpjson.ErrUnsupportedMediaType) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return CreateDailyWorkInput{}, false
	}
	if errors.Is(err, httpjson.ErrBodyTooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreateDailyWorkInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON com os campos da diária"})
		return CreateDailyWorkInput{}, false
	}

	workDate, err := calendarDate.Parse(request.WorkDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "work_date deve usar o formato AAAA-MM-DD"})
		return CreateDailyWorkInput{}, false
	}
	return CreateDailyWorkInput{
		EmployeeID: request.EmployeeID, WorkDate: workDate,
		AmountCents: request.AmountCents, Notes: request.Notes,
	}, true
}
