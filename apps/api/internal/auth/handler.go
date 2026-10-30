package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
)

const setupTokenHeader = "X-Agrofin-Setup-Token"

type managerRegistrar interface {
	RegisterManager(ctx context.Context, loginName, password string) error
}

// Handler coordena o cadastro inicial do gerente.
type Handler struct {
	registrar  managerRegistrar
	setupToken string
}

// NewHandler recebe o token de cadastro inicial por configuração do servidor.
func NewHandler(registrar managerRegistrar, setupToken string) *Handler {
	return &Handler{registrar: registrar, setupToken: setupToken}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if h.setupToken == "" {
		writeError(w, http.StatusServiceUnavailable, "cadastro inicial indisponível")
		return
	}
	if !matchesSetupToken(h.setupToken, r.Header.Get(setupTokenHeader)) {
		writeError(w, http.StatusForbidden, "token de cadastro inválido")
		return
	}

	input, ok := decodeRegisterRequest(w, r)
	if !ok {
		return
	}

	if err := h.registrar.RegisterManager(r.Context(), input.LoginName, input.Password); err != nil {
		switch {
		case errors.Is(err, ErrManagerAlreadyRegistered):
			writeError(w, http.StatusConflict, "cadastro do gerente já concluído")
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Print("falha ao cadastrar o gerente")
			writeError(w, http.StatusInternalServerError, "não foi possível cadastrar o gerente")
		}
		return
	}

	writeJSON(w, http.StatusCreated, registerResponse{Registered: true})
}

func matchesSetupToken(expected, provided string) bool {
	if len(provided) == 0 || len(provided) > 256 {
		return false
	}

	// Usa o hash SHA-256 para comparar os tokens de forma segura, evitando ataques de timing.
	expectedHash := sha256.Sum256([]byte(expected))
	providedHash := sha256.Sum256([]byte(provided))
	return subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) == 1
}
