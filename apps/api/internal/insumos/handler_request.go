// Decodifica e valida os campos aceitos no cadastro HTTP de insumos.
package insumos

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
)

type createRequest struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	DefaultUnit string `json:"default_unit"`
	Notes       string `json:"notes"`
}

func decodeCreateRequest(w http.ResponseWriter, r *http.Request) (CreateInput, bool) {
	request, err := httpjson.Decode[createRequest](w, r)
	switch {
	case errors.Is(err, httpjson.ErrUnsupportedMediaType):
		writeError(w, http.StatusUnsupportedMediaType, "Envie o corpo como application/json")
		return CreateInput{}, false
	case errors.Is(err, httpjson.ErrBodyTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "Corpo da requisição excede 4 KiB")
		return CreateInput{}, false
	case err != nil || request == nil:
		writeError(w, http.StatusBadRequest, "Envie um único objeto JSON com os campos do insumo")
		return CreateInput{}, false
	}

	input := CreateInput{
		Name:        strings.TrimSpace(request.Name),
		Category:    Category(strings.TrimSpace(request.Category)),
		DefaultUnit: strings.TrimSpace(request.DefaultUnit),
		Notes:       strings.TrimSpace(request.Notes),
	}
	// PostgreSQL não representa o caractere nulo em TEXT. Rejeitá-lo aqui
	// trata o valor como erro da entrada, inclusive quando veio escapado no JSON.
	switch {
	case input.Name == "" || strings.ContainsRune(input.Name, 0):
		writeError(w, http.StatusBadRequest, "Informe um nome válido para o insumo")
	case input.Category != CategoryFertilizer && input.Category != CategoryPesticide:
		writeError(w, http.StatusBadRequest, "A categoria deve ser fertilizante ou defensivo")
	case input.DefaultUnit == "" || strings.ContainsRune(input.DefaultUnit, 0):
		writeError(w, http.StatusBadRequest, "Informe uma unidade padrão válida para o insumo")
	case strings.ContainsRune(input.Notes, 0):
		writeError(w, http.StatusBadRequest, "Informe uma observação válida para o insumo")
	default:
		return input, true
	}
	return CreateInput{}, false
}
