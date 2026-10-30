package auth

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

type registerRequest struct {
	LoginName string `json:"login_name"`
	Password  string `json:"password"`
}

func decodeRegisterRequest(w http.ResponseWriter, r *http.Request) (*registerRequest, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "envie o corpo como application/json")
		return nil, false
	}

	// O cadastro aceita um corpo pequeno para limitar memória e nunca ignora campos extras.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input *registerRequest
	decodeErr := decoder.Decode(&input)
	if decodeErr == nil && input != nil {
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				decodeErr = errors.New("mais de um valor json")
			} else {
				decodeErr = trailingErr
			}
		}
	} else if decodeErr == nil {
		decodeErr = errors.New("objeto json obrigatório")
	}
	var sizeErr *http.MaxBytesError
	if errors.As(decodeErr, &sizeErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "corpo da requisição excede 4 KiB")
		return nil, false
	}
	if decodeErr != nil || input == nil {
		writeError(w, http.StatusBadRequest, "envie um único objeto JSON com login_name e password")
		return nil, false
	}
	if input.LoginName == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "login_name e password são obrigatórios")
		return nil, false
	}
	return input, true
}
