package diarias

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
)

const dailyWorkDateLayout = "2006-01-02"

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

	employeeID, err := normalizeDailyWorkEmployeeID(request.EmployeeID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "employee_id deve ser um UUID válido"})
		return CreateDailyWorkInput{}, false
	}
	workDate, err := parseDailyWorkDate(request.WorkDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "work_date deve usar o formato AAAA-MM-DD"})
		return CreateDailyWorkInput{}, false
	}
	return CreateDailyWorkInput{
		EmployeeID: employeeID, WorkDate: workDate,
		AmountCents: request.AmountCents, Notes: request.Notes,
	}, true
}

func parseDailyWorkDate(value string) (time.Time, error) {
	if len(value) != len(dailyWorkDateLayout) || strings.TrimSpace(value) != value {
		return time.Time{}, errors.New("data da diária inválida")
	}
	date, err := time.Parse(dailyWorkDateLayout, value)
	if err != nil || date.Format(dailyWorkDateLayout) != value {
		return time.Time{}, errors.New("data da diária inválida")
	}
	return date, nil
}
