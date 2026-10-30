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

// handlerService é o contrato HTTP mínimo; os handlers não conhecem SQL nem
// implementam as regras de senha e sessão.
type handlerService interface {
	RegisterManager(ctx context.Context, loginName, password string) error
	Login(ctx context.Context, loginName, password string) (createdSession, error)
	Logout(ctx context.Context, token string) error
}

// HandlerConfig contém valores de configuração já lidos e validados pelo main.
// O pacote auth não precisa conhecer nomes de variáveis de ambiente.
type HandlerConfig struct {
	SetupToken   string
	CookieSecure bool
}

// Handler traduz requests/responses HTTP e delega as regras de autenticação ao Service.
type Handler struct {
	service      handlerService
	setupToken   string
	cookieSecure bool
}

// NewHandler recebe o serviço e a configuração do cookie sem ler o ambiente diretamente.
func NewHandler(service handlerService, config HandlerConfig) *Handler {
	return &Handler{service: service, setupToken: config.SetupToken, cookieSecure: config.CookieSecure}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	// O cadastro é apenas de instalação. Depois de existir o gerente único, a
	// restrição do banco também impede outro cadastro mesmo que o token vaze.
	if h.setupToken == "" {
		writeError(w, http.StatusServiceUnavailable, "cadastro inicial indisponível")
		return
	}
	if !matchesSetupToken(h.setupToken, r.Header.Get(setupTokenHeader)) {
		writeError(w, http.StatusForbidden, "token de cadastro inválido")
		return
	}

	input, ok := decodeCredentialsRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.RegisterManager(r.Context(), input.LoginName, input.Password); err != nil {
		switch {
		case errors.Is(err, ErrManagerAlreadyRegistered):
			writeError(w, http.StatusConflict, "cadastro do gerente já concluído")
		case errors.Is(err, ErrAuthenticationBusy):
			writeError(w, http.StatusServiceUnavailable, "autenticação temporariamente indisponível")
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

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentialsRequest(w, r)
	if !ok {
		return
	}

	// O handler cuida do protocolo HTTP; comparação da senha e criação da sessão
	// ficam no Service. A limitação de tráfego por IP será responsabilidade da borda.
	session, err := h.service.Login(r.Context(), input.LoginName, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		case errors.Is(err, ErrAuthenticationBusy):
			writeError(w, http.StatusServiceUnavailable, "autenticação temporariamente indisponível")
		default:
			log.Print("falha ao autenticar o gerente")
			writeError(w, http.StatusInternalServerError, "não foi possível iniciar a sessão")
		}
		return
	}

	// O token não vai no JSON. O navegador o recebe apenas em cookie HttpOnly,
	// que JavaScript da página não consegue ler diretamente.
	setSessionCookie(w, session.Token, session.ExpiresAt, h.cookieSecure)
	writeJSON(w, http.StatusOK, sessionResponse{Authenticated: true})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName(h.cookieSecure))
	token := ""
	if err == nil {
		token = cookie.Value
	}
	// O endpoint funciona sem cookie para o cliente limpar estado local após
	// expiração; revogação e limpeza são idempotentes.
	if err := h.service.Logout(r.Context(), token); err != nil {
		log.Print("falha ao encerrar sessão")
		writeError(w, http.StatusInternalServerError, "não foi possível encerrar a sessão")
		return
	}

	clearSessionCookie(w, h.cookieSecure)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	// A rota é protegida por middleware, mas esta guarda também protege o handler
	// se for montado isoladamente em outro mux ou em um teste.
	if _, ok := PrincipalFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "autenticação necessária")
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{Authenticated: true})
}

func matchesSetupToken(expected, provided string) bool {
	if len(provided) == 0 || len(provided) > 256 {
		return false
	}

	// Hashes do mesmo tamanho permitem comparação em tempo constante mesmo se os
	// tokens tiverem comprimentos diferentes. Isso protege o segredo de bootstrap.
	expectedHash := sha256.Sum256([]byte(expected))
	providedHash := sha256.Sum256([]byte(provided))
	return subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) == 1
}
