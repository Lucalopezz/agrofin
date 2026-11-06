package auth

import (
	"errors"
	"net/http"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
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
	input, ok := decodeJSONRequest[credentialsRequest](w, r)
	if !ok {
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
	input, ok := decodeJSONRequest[registrationRequest](w, r)
	if !ok {
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
	input, ok := decodeJSONRequest[managerNamePatchRequest](w, r)
	if !ok {
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
	input, ok := decodeJSONRequest[managerStatusPatchRequest](w, r)
	if !ok {
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
	input, ok := decodeJSONRequest[passwordChangeRequest](w, r)
	if !ok {
		return nil, false
	}
	if input == nil || input.CurrentPassword == "" || input.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "current_password e new_password são obrigatórios")
		return nil, false
	}
	return input, true
}

// decodeJSONRequest adapta os erros comuns do protocolo às mensagens usadas
// pelos endpoints de auth, mantendo a validação dos campos em cada decoder.
func decodeJSONRequest[T any](w http.ResponseWriter, r *http.Request) (*T, bool) {
	input, err := httpjson.Decode[T](w, r)
	if err == nil {
		return input, true
	}
	switch {
	case errors.Is(err, httpjson.ErrUnsupportedMediaType):
		writeError(w, http.StatusUnsupportedMediaType, "envie o corpo como application/json")
	case errors.Is(err, httpjson.ErrBodyTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "corpo da requisição excede 4 KiB")
	default:
		writeError(w, http.StatusBadRequest, "envie um único objeto JSON válido")
	}
	return nil, false
}
