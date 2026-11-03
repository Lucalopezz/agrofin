package colheitas

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
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
