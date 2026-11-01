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

type registrationRequest struct {
	Name      string `json:"name"`
	LoginName string `json:"login_name"`
	Password  string `json:"password"`
}

type managerNamePatchRequest struct {
	Name string `json:"name"`
}

type managerStatusPatchRequest struct {
	// Ponteiro diferencia false enviado explicitamente de campo ausente ou null.
	Active *bool `json:"active"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// decodeCredentialsRequest valida o formato de login; as regras do login ficam no Service.
func decodeCredentialsRequest(w http.ResponseWriter, r *http.Request) (*credentialsRequest, bool) {
	var input *credentialsRequest
	if !decodeJSONRequest(w, r, &input) {
		return nil, false
	}
	if input == nil || input.LoginName == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "login_name e password são obrigatórios")
		return nil, false
	}
	return input, true
}

// decodeRegistrationRequest exige nome de exibição e credenciais no cadastro de conta.
func decodeRegistrationRequest(w http.ResponseWriter, r *http.Request) (*registrationRequest, bool) {
	var input *registrationRequest
	if !decodeJSONRequest(w, r, &input) {
		return nil, false
	}
	if input == nil || input.Name == "" || input.LoginName == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "name, login_name e password são obrigatórios")
		return nil, false
	}
	return input, true
}

// decodeManagerNamePatchRequest aceita somente name; login_name continua imutável pela API.
func decodeManagerNamePatchRequest(w http.ResponseWriter, r *http.Request) (*managerNamePatchRequest, bool) {
	var input *managerNamePatchRequest
	if !decodeJSONRequest(w, r, &input) {
		return nil, false
	}
	if input == nil || input.Name == "" {
		writeError(w, http.StatusBadRequest, "name é obrigatório")
		return nil, false
	}
	return input, true
}

// decodeManagerStatusPatchRequest exige active explícito para não interpretar
// um PATCH vazio como uma ordem de desativação.
func decodeManagerStatusPatchRequest(w http.ResponseWriter, r *http.Request) (*managerStatusPatchRequest, bool) {
	var input *managerStatusPatchRequest
	if !decodeJSONRequest(w, r, &input) {
		return nil, false
	}
	if input == nil || input.Active == nil {
		writeError(w, http.StatusBadRequest, "active é obrigatório e deve ser true ou false")
		return nil, false
	}
	return input, true
}

// decodePasswordChangeRequest aceita as duas senhas necessárias; confirmação
// de senha é validação de formulário e não precisa ser enviada à API.
func decodePasswordChangeRequest(w http.ResponseWriter, r *http.Request) (*passwordChangeRequest, bool) {
	var input *passwordChangeRequest
	if !decodeJSONRequest(w, r, &input) {
		return nil, false
	}
	if input == nil || input.CurrentPassword == "" || input.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "current_password e new_password são obrigatórios")
		return nil, false
	}
	return input, true
}

// decodeJSONRequest centraliza limites e regras do protocolo para os corpos da auth.
// O parâmetro recebe um ponteiro para ponteiro; assim o chamador distingue JSON null.
func decodeJSONRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "envie o corpo como application/json")
		return false
	}

	// Corpos pequenos limitam memória e evitam aceitar campos que o endpoint não conhece.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	decodeErr := decoder.Decode(target)
	if decodeErr == nil {
		// Um segundo Decode precisa encontrar EOF; isso rejeita outro objeto ou lixo
		// depois do primeiro JSON válido.
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				decodeErr = errors.New("mais de um valor json")
			} else {
				decodeErr = trailingErr
			}
		}
	}
	var sizeErr *http.MaxBytesError
	if errors.As(decodeErr, &sizeErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "corpo da requisição excede 4 KiB")
		return false
	}
	if decodeErr != nil {
		writeError(w, http.StatusBadRequest, "envie um único objeto JSON válido")
		return false
	}
	return true
}
