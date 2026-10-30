package auth

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Este cabeçalho precisa ser definido pelo frontend; formulários HTML comuns não
// conseguem enviá-lo e sites externos exigiriam CORS para incluí-lo em fetch.
const requestMarkerHeader = "X-Agrofin-Request"

// ValidateSessionConfig exige uma origem pública exata e coerente com o uso de cookies Secure.
// Falhar na inicialização é mais seguro do que subir HTTPS sem cookie Secure ou
// HTTP local com um cookie que o navegador não enviaria.
func ValidateSessionConfig(publicOrigin string, cookieSecure bool) error {
	parsed, ok := normalizeOrigin(publicOrigin)
	if !ok {
		return errors.New("PUBLIC_ORIGIN deve conter somente uma origem http ou https válida")
	}
	isHTTPS := strings.HasPrefix(parsed, "https://")
	if cookieSecure != isHTTPS {
		return errors.New("SESSION_COOKIE_SECURE deve acompanhar o esquema de PUBLIC_ORIGIN")
	}
	return nil
}

// RequireSameOrigin protege operações com efeitos contra formulários e chamadas cross-site.
// Como navegadores enviam cookies automaticamente, Origin e um cabeçalho customizado
// obrigam o frontend legítimo a marcar cada escrita. A API não libera CORS com credenciais.
func RequireSameOrigin(publicOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Métodos de leitura não devem alterar estado. Se um GET passar a gravar,
		// ele precisa ser redesenhado como escrita para receber esta proteção.
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		origins := r.Header.Values("Origin")
		fetchSites := r.Header.Values("Sec-Fetch-Site")
		// Cabeçalhos repetidos são ambíguos, então não escolhemos arbitrariamente um.
		// Sec-Fetch-Site pode faltar em clientes que não são navegadores; Origin e o
		// marcador continuam obrigatórios para todas as operações de escrita.
		validFetchSite := len(fetchSites) == 0 || (len(fetchSites) == 1 && fetchSites[0] == "same-origin")
		if len(origins) != 1 || !sameOrigin(publicOrigin, origins[0]) || !validFetchSite || r.Header.Get(requestMarkerHeader) != "1" {
			w.Header().Set("Vary", "Origin, Sec-Fetch-Site")
			writeError(w, http.StatusForbidden, "origem da requisição não permitida")
			return
		}
		w.Header().Set("Vary", "Origin, Sec-Fetch-Site")
		next.ServeHTTP(w, r)
	})
}

func sameOrigin(expected, provided string) bool {
	normalizedExpected, expectedOK := normalizeOrigin(expected)
	normalizedProvided, providedOK := normalizeOrigin(provided)
	return expectedOK && providedOK && normalizedExpected == normalizedProvided
}

func normalizeOrigin(value string) (string, bool) {
	// A comparação ignora somente diferenças equivalentes (caixa do host e porta
	// padrão). Caminho, credenciais, query/fragmento com conteúdo e esquemas não
	// HTTP(S) não pertencem a uma origem e são rejeitados.
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", false
	}
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	return scheme + "://" + host, true
}
