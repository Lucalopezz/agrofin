package colheitas

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const rateDateLayout = "2006-01-02"

type createRateRequest struct {
	BoxType     BoxType `json:"box_type"`
	AmountCents int64   `json:"amount_cents"`
	ValidFrom   string  `json:"valid_from"`
	ValidUntil  *string `json:"valid_until"`
}

type createHarvestEntryRequest struct {
	EmployeeID  string                     `json:"employee_id"`
	HarvestDate string                     `json:"harvest_date"`
	Items       []createHarvestItemRequest `json:"items"`
}

type createHarvestItemRequest struct {
	BoxType  BoxType `json:"box_type"`
	Quantity int32   `json:"quantity"`
}

func decodeCreateRateRequest(w http.ResponseWriter, r *http.Request) (CreateRateInput, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return CreateRateInput{}, false
	}

	// O payload é pequeno e fechado; o limite e os campos estritos evitam aceitar
	// propriedades como user_id ou employee_id que não pertencem à entrada.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request *createRateRequest
	err = decoder.Decode(&request)
	if err == nil && request != nil {
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				err = errors.New("mais de um valor JSON")
			} else {
				err = trailingErr
			}
		}
	}
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreateRateInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON com os campos da tarifa"})
		return CreateRateInput{}, false
	}

	validFrom, err := parseRateDate(request.ValidFrom)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo valid_from deve usar o formato AAAA-MM-DD"})
		return CreateRateInput{}, false
	}
	var validUntil *time.Time
	if request.ValidUntil != nil {
		parsed, err := parseRateDate(*request.ValidUntil)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo valid_until deve usar AAAA-MM-DD ou ser null"})
			return CreateRateInput{}, false
		}
		validUntil = &parsed
	}

	return CreateRateInput{
		BoxType: request.BoxType, AmountCents: request.AmountCents,
		ValidFrom: validFrom, ValidUntil: validUntil,
	}, true
}

func decodeCreateHarvestRequest(w http.ResponseWriter, r *http.Request) (CreateHarvestInput, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return CreateHarvestInput{}, false
	}

	// Até dois tipos de caixa formam um lançamento pequeno; o limite evita que
	// um corpo excessivo ocupe memória antes da validação dos itens.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request *createHarvestEntryRequest
	err = decoder.Decode(&request)
	if err == nil && request != nil {
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				err = errors.New("mais de um valor JSON")
			} else {
				err = trailingErr
			}
		}
	}
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreateHarvestInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON com os campos do lançamento"})
		return CreateHarvestInput{}, false
	}
	employeeID, err := normalizeHarvestEmployeeID(request.EmployeeID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "employee_id deve ser um UUID válido"})
		return CreateHarvestInput{}, false
	}
	harvestDate, err := parseRateDate(request.HarvestDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "harvest_date deve usar o formato AAAA-MM-DD"})
		return CreateHarvestInput{}, false
	}
	items := make([]HarvestItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, HarvestItemInput{BoxType: item.BoxType, Quantity: item.Quantity})
	}
	return CreateHarvestInput{
		EmployeeID: employeeID, HarvestDate: harvestDate, Items: items,
	}, true
}

func decodeListHarvestEntriesRequest(w http.ResponseWriter, r *http.Request) (HarvestPeriod, bool) {
	// Exigir limites explícitos impede que uma consulta de tela leia todo o histórico
	// por acidente; os dois dias pertencem ao intervalo retornado.
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Parâmetros de consulta inválidos"})
		return HarvestPeriod{}, false
	}
	startValues, hasStart := query["start_date"]
	endValues, hasEnd := query["end_date"]
	if len(query) != 2 || !hasStart || !hasEnd || len(startValues) != 1 || len(endValues) != 1 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Informe start_date e end_date uma única vez"})
		return HarvestPeriod{}, false
	}

	startDate, err := parseRateDate(startValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "start_date deve usar o formato AAAA-MM-DD"})
		return HarvestPeriod{}, false
	}
	endDate, err := parseRateDate(endValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve usar o formato AAAA-MM-DD"})
		return HarvestPeriod{}, false
	}
	return HarvestPeriod{StartDate: startDate, EndDate: endDate}, true
}

func parseRateDate(value string) (time.Time, error) {
	if len(value) != len(rateDateLayout) || strings.TrimSpace(value) != value {
		return time.Time{}, errors.New("data de tarifa inválida")
	}
	date, err := time.Parse(rateDateLayout, value)
	if err != nil || date.Year() < 1 || date.Format(rateDateLayout) != value {
		return time.Time{}, errors.New("data de tarifa inválida")
	}
	return date, nil
}
