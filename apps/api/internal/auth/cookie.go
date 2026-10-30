package auth

import (
	"net/http"
	"time"
)

const (
	// HTTP local não permite Secure; o nome separado evita confundir esse cookie
	// de desenvolvimento com o cookie emitido em produção HTTPS.
	localSessionCookieName = "agrofin_session"
	// O prefixo __Host- é uma defesa do navegador: exige Secure, Path=/ e sem Domain.
	secureSessionCookieName = "__Host-agrofin_session"
)

func sessionCookieName(secure bool) string {
	if secure {
		return secureSessionCookieName
	}
	return localSessionCookieName
}

// setSessionCookie é o único ponto que serializa o token para o navegador.
// HttpOnly bloqueia leitura por JavaScript; SameSite=Lax reduz envios cross-site.
func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		// A expiração efetiva ainda é verificada no servidor; o mínimo evita que
		// arredondamento para zero faça o navegador descartar o cookie recém-emitido.
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName(secure),
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	// Para apagar, o navegador precisa receber o mesmo nome e Path do cookie
	// original, com expiração passada e MaxAge negativo.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName(secure),
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
