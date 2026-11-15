// Decodifica filtros e corpos HTTP de tarifas e lançamentos de colheita.
package colheitas

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

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

type updateHarvestEntryRequest struct {
	Items []createHarvestItemRequest `json:"items"`
}

func decodeCreateRateRequest(w http.ResponseWriter, r *http.Request) (CreateRateInput, bool) {
	request, err := httpjson.Decode[createRateRequest](w, r)
	if errors.Is(err, httpjson.ErrUnsupportedMediaType) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return CreateRateInput{}, false
	}
	if errors.Is(err, httpjson.ErrBodyTooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreateRateInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON com os campos da tarifa"})
		return CreateRateInput{}, false
	}

	validFrom, err := calendarDate.Parse(request.ValidFrom)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo valid_from deve usar o formato AAAA-MM-DD"})
		return CreateRateInput{}, false
	}
	var validUntil *time.Time
	if request.ValidUntil != nil {
		parsed, err := calendarDate.Parse(*request.ValidUntil)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo valid_until deve usar AAAA-MM-DD ou ser null"})
			return CreateRateInput{}, false
		}
		validUntil = &parsed
	}

	_, err = normalizeRate(Rate{BoxType: request.BoxType, AmountCents: request.AmountCents, ValidFrom: validFrom, ValidUntil: validUntil})
	if err != nil {
		writeCreateRateResponse(w, Rate{}, err)
		return CreateRateInput{}, false
	}
	return CreateRateInput{
		BoxType: request.BoxType, AmountCents: request.AmountCents,
		ValidFrom: validFrom, ValidUntil: validUntil,
	}, true
}

func decodeCreateHarvestRequest(w http.ResponseWriter, r *http.Request) (CreateHarvestInput, bool) {
	request, err := httpjson.Decode[createHarvestEntryRequest](w, r)
	if errors.Is(err, httpjson.ErrUnsupportedMediaType) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return CreateHarvestInput{}, false
	}
	if errors.Is(err, httpjson.ErrBodyTooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return CreateHarvestInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON com os campos do lançamento"})
		return CreateHarvestInput{}, false
	}
	harvestDate, err := calendarDate.Parse(request.HarvestDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "harvest_date deve usar o formato AAAA-MM-DD"})
		return CreateHarvestInput{}, false
	}
	items := make([]HarvestItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, HarvestItemInput{BoxType: item.BoxType, Quantity: item.Quantity})
	}
	harvestItems := make([]HarvestItem, 0, len(items))
	for _, item := range items {
		harvestItems = append(harvestItems, HarvestItem{BoxType: item.BoxType, Quantity: item.Quantity})
	}
	harvest, err := normalizeHarvest(Harvest{EmployeeID: request.EmployeeID, HarvestDate: harvestDate, Items: harvestItems})
	if err != nil {
		writeCreateHarvestResponse(w, Harvest{}, err)
		return CreateHarvestInput{}, false
	}
	return CreateHarvestInput{EmployeeID: harvest.EmployeeID, HarvestDate: harvest.HarvestDate, Items: items}, true
}

func decodeHarvestPeriodQuery(w http.ResponseWriter, r *http.Request) (HarvestPeriod, bool) {
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

	startDate, err := calendarDate.Parse(startValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "start_date deve usar o formato AAAA-MM-DD"})
		return HarvestPeriod{}, false
	}
	endDate, err := calendarDate.Parse(endValues[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve usar o formato AAAA-MM-DD"})
		return HarvestPeriod{}, false
	}
	if endDate.Before(startDate) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve ser igual ou posterior a start_date"})
		return HarvestPeriod{}, false
	}
	return HarvestPeriod{StartDate: startDate, EndDate: endDate}, true
}

func decodeUpdateHarvestRequest(w http.ResponseWriter, r *http.Request) (UpdateHarvestInput, bool) {
	request, err := httpjson.Decode[updateHarvestEntryRequest](w, r)
	if errors.Is(err, httpjson.ErrUnsupportedMediaType) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return UpdateHarvestInput{}, false
	}
	if errors.Is(err, httpjson.ErrBodyTooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return UpdateHarvestInput{}, false
	}
	if err != nil || request == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON com os itens do lançamento"})
		return UpdateHarvestInput{}, false
	}
	items := make([]HarvestItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, HarvestItemInput{BoxType: item.BoxType, Quantity: item.Quantity})
	}
	if len(items) == 0 {
		writeUpdateHarvestResponse(w, Harvest{}, ErrInvalidHarvestItems)
		return UpdateHarvestInput{}, false
	}
	seen := make(map[BoxType]bool, len(items))
	for _, item := range items {
		if item.BoxType != BoxTypeNormal && item.BoxType != BoxTypeMoagem {
			writeUpdateHarvestResponse(w, Harvest{}, ErrInvalidBoxType)
			return UpdateHarvestInput{}, false
		}
		if item.Quantity <= 0 {
			writeUpdateHarvestResponse(w, Harvest{}, ErrInvalidHarvestQuantity)
			return UpdateHarvestInput{}, false
		}
		if seen[item.BoxType] {
			writeUpdateHarvestResponse(w, Harvest{}, ErrDuplicateHarvestItem)
			return UpdateHarvestInput{}, false
		}
		seen[item.BoxType] = true
	}
	return UpdateHarvestInput{Items: items}, true
}

// decodeHarvestID valida o identificador HTTP antes de chamar o serviço.
func decodeHarvestID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := uuid.Normalize(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O ID da colheita deve ser um UUID válido"})
		return "", false
	}
	return id, true
}

// normalizeRate valida os dados enviados antes de chamar o serviço de tarifas.
func normalizeRate(rate Rate) (Rate, error) {
	switch rate.BoxType {
	case BoxTypeNormal, BoxTypeMoagem:
	default:
		return Rate{}, ErrInvalidBoxType
	}

	if !money.IsPositiveCents(rate.AmountCents) {
		return Rate{}, ErrInvalidRateAmount
	}
	rate.ValidFrom = calendarDate.Normalize(rate.ValidFrom)
	if rate.ValidFrom.IsZero() {
		return Rate{}, ErrInvalidRatePeriod
	}

	if rate.ValidUntil != nil {
		until := calendarDate.Normalize(*rate.ValidUntil)
		if until.IsZero() || !until.After(rate.ValidFrom) {
			return Rate{}, ErrInvalidRatePeriod
		}
		// Copiar a data evita compartilhar o ponteiro recebido: uma alteração no
		// argumento do chamador não pode mudar depois a vigência que foi validada.
		rate.ValidUntil = &until
	}
	return rate, nil
}
