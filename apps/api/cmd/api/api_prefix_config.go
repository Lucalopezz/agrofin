// Resolve e valida o prefixo público das rotas a partir da configuração de ambiente.
package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func apiPrefixFromEnv() (string, error) {
	apiPrefix := strings.TrimSpace(os.Getenv("API_PREFIX"))
	if apiPrefix == "" {
		apiPrefix = defaultAPIPrefix
	}
	// O prefixo é usado como caminho e também passado ao StripPrefix; rejeitar
	// caminhos ambíguos evita que a rota pública e a rota interna se desencontrem.
	if !strings.HasPrefix(apiPrefix, "/") || strings.Contains(apiPrefix, "//") || strings.ContainsAny(apiPrefix, "?# \t\r\n") {
		return "", fmt.Errorf("API_PREFIX inválido: use um caminho absoluto como %s", defaultAPIPrefix)
	}

	apiPrefix = strings.TrimSuffix(apiPrefix, "/")
	if apiPrefix == "" || path.Clean(apiPrefix) != apiPrefix {
		return "", fmt.Errorf("API_PREFIX inválido: use um caminho absoluto como %s", defaultAPIPrefix)
	}
	return apiPrefix, nil
}
