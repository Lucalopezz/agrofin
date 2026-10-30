package funcionarios

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

type createEmployeeRequest struct {
	Name string `json:"name"`
}

func decodeCreateEmployeeRequest(w http.ResponseWriter, r *http.Request) (*createEmployeeRequest, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "envie o corpo como application/json"})
		return nil, false
	}

	// O cadastro contém apenas um nome. Limitamos o corpo, incluindo espaços
	// extras, para não consumir memória sem limite ao decodificar a requisição.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	// O decodificador não ignora campos extras, que podem indicar erro do cliente.
	decoder.DisallowUnknownFields()
	var input *createEmployeeRequest

	// Decodifica o JSON em input. Se houver erro, input permanece nil e err descreve o problema.
	err = decoder.Decode(&input)
	if err == nil && input != nil {
		// Um segundo JSON ou conteúdo extra não pode ser ignorado depois do nome.
		// {"name": "João"}{"name": "Maria"}
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				err = errors.New("mais de um valor json")
			} else {
				err = trailingErr
			}
		}
	}
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "corpo da requisição excede 4 KiB"})
		return nil, false
	}
	if err != nil || input == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "envie um único objeto JSON contendo somente name"})
		return nil, false
	}

	return input, true
}
