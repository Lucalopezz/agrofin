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
	if err := decoder.Decode(&input); err == nil && input != nil {
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				err = errors.New("mais de um valor json")
			} else {
				err = trailingErr
			}
		}
	} else if err == nil {
		err = errors.New("objeto json obrigatório")
	}
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "corpo da requisição excede 4 KiB")
		return nil, false
	}
	if err != nil || input == nil {
		writeError(w, http.StatusBadRequest, "envie um único objeto JSON com login_name e password")
		return nil, false
	}
	if input.LoginName == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "login_name e password são obrigatórios")
		return nil, false
	}
	return input, true
}
