package auth

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

type credentialsRequest struct {
	LoginName string `json:"login_name"`
	Password  string `json:"password"`
}

// decodeCredentialsRequest valida somente o formato HTTP/JSON. Regras como
// tamanho mínimo da senha e normalização do login pertencem ao Service.
func decodeCredentialsRequest(w http.ResponseWriter, r *http.Request) (*credentialsRequest, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "envie o corpo como application/json")
		return nil, false
	}

	// Login e cadastro compartilham este limite, que restringe memória e impede
	// que campos extras (por exemplo, um user_id escolhido pelo cliente) passem ocultos.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	// Rejeitar campos desconhecidos faz o servidor falhar fechado: enviar user_id
	// por engano não parece funcionar silenciosamente nem muda a propriedade.
	decoder.DisallowUnknownFields()
	// O ponteiro distingue um objeto JSON de "null"; sem ele, null poderia ser
	// confundido com um objeto vazio com campos zero-value.
	var input *credentialsRequest
	decodeErr := decoder.Decode(&input)
	if decodeErr == nil && input != nil {
		// Um segundo Decode precisa encontrar EOF. Assim, "{} {}" e lixo após o
		// primeiro objeto não são aceitos como uma única credencial válida.
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
