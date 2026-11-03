package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
)

const (
	rateActiveKeyVersionEnv = "HARVEST_RATE_ACTIVE_KEY_VERSION"
	rateKeysHexJSONEnv      = "HARVEST_RATE_KEYS_HEX_JSON"
)

// rateCipherFromEnv carrega as chaves fora do banco e valida toda a configuração
// antes de abrir o servidor, para nunca disponibilizar escrita sem cifra.
func rateCipherFromEnv() (*colheitas.RateCipher, error) {
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

	protector, err := colheitas.NewRateCipher(activeVersion, keys)
	if err != nil {
		// A mensagem não inclui os valores recebidos do ambiente, que contêm segredos.
		return nil, fmt.Errorf("configuração de chaves de tarifa inválida")
	}
	return protector, nil
}
