// Carrega e valida o chaveiro financeiro compartilhado a partir das variáveis de ambiente.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
)

const (
	rateActiveKeyVersionEnv = "HARVEST_RATE_ACTIVE_KEY_VERSION"
	rateKeysHexJSONEnv      = "HARVEST_RATE_KEYS_HEX_JSON"
)

// financialKeyringFromEnv carrega o chaveiro fora do banco e valida a
// configuração antes de abrir o servidor. Os nomes antigos das variáveis são
// mantidos para não invalidar ambientes já configurados.
func financialKeyringFromEnv() (*financialcipher.Keyring, error) {
	activeVersion := os.Getenv(rateActiveKeyVersionEnv)
	encodedKeys := os.Getenv(rateKeysHexJSONEnv)
	if activeVersion == "" || encodedKeys == "" {
		return nil, fmt.Errorf("%s e %s são obrigatórias", rateActiveKeyVersionEnv, rateKeysHexJSONEnv)
	}

	var keysHex map[string]string
	if err := json.Unmarshal([]byte(encodedKeys), &keysHex); err != nil || len(keysHex) == 0 {
		return nil, fmt.Errorf("%s deve ser um objeto JSON com chaves hexadecimais", rateKeysHexJSONEnv)
	}

	keys := make(map[string][]byte, len(keysHex))
	defer func() {
		for _, key := range keys {
			clear(key)
		}
	}()
	for version, encodedKey := range keysHex {
		key, err := hex.DecodeString(encodedKey)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("%s deve conter chaves de 32 bytes em hexadecimal", rateKeysHexJSONEnv)
		}
		keys[version] = key
	}

	keyring, err := financialcipher.NewKeyring(activeVersion, keys)
	if err != nil {
		// A mensagem não inclui os valores recebidos do ambiente, que contêm segredos.
		return nil, fmt.Errorf("configuração das chaves financeiras inválida")
	}
	return keyring, nil
}
