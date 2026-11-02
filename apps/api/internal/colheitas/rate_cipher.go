package colheitas

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	rateKeySize       = 32
	rateNonceSize     = 12
	ratePlaintextSize = 8
)

// ProtectedRateAmount corresponde ao envelope financeiro previsto no esquema.
// Somente estes bytes e a versão, nunca os centavos em claro, poderão ir ao banco.
type ProtectedRateAmount struct {
	Ciphertext []byte
	Nonce      []byte
	KeyVersion string
}

// RateCipher protege valores de tarifa com AES-256-GCM. As chaves chegam em
// memória pela composição; o componente não as gera, lê do banco ou persiste.
type RateCipher struct {
	activeVersion string
	keys          map[string]cipher.AEAD
}

// NewRateCipher usa a versão ativa para novas cifras e mantém versões anteriores
// disponíveis para recuperação. Trocar a versão ativa exige reconstruir a instância.
func NewRateCipher(activeVersion string, keys map[string][]byte) (*RateCipher, error) {
	if activeVersion == "" || strings.TrimSpace(activeVersion) != activeVersion {
		return nil, ErrInvalidRateCipherConfig
	}
	if _, ok := keys[activeVersion]; !ok {
		return nil, ErrInvalidRateCipherConfig
	}
	prepared := make(map[string]cipher.AEAD, len(keys))
	seen := make(map[[rateKeySize]byte]bool, len(keys))
	for version, key := range keys {
		if version == "" || strings.TrimSpace(version) != version || !utf8.ValidString(version) || len(key) != rateKeySize {
			return nil, ErrInvalidRateCipherConfig
		}
		// Uma versão nova precisa de material novo: apenas renomear a mesma chave
		// não constitui rotação nem reinicia seus limites de utilização.
		var material [rateKeySize]byte
		copy(material[:], key)
		if seen[material] {
			return nil, ErrInvalidRateCipherConfig
		}
		seen[material] = true
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, ErrInvalidRateCipherConfig
		}
		// A biblioteca gera o nonce aleatório e o inclui no resultado de Seal.
		// O limite de 2^32 cifras por chave deve ser respeitado entre todas as
		// instâncias que a compartilhem, inclusive após reinícios, antes da produção.
		aead, err := cipher.NewGCMWithRandomNonce(block)
		if err != nil {
			return nil, ErrInvalidRateCipherConfig
		}
		prepared[version] = aead
	}
	// Guardar cifras preparadas em um mapa próprio evita depender de alterações
	// posteriores no mapa ou nos slices de chave recebidos pelo construtor.
	return &RateCipher{activeVersion: activeVersion, keys: prepared}, nil
}

// Encrypt vincula os centavos à conta e ao registro. O ID da tarifa precisa
// estar definido antes da cifra, mesmo que a persistência ainda não tenha ocorrido.
func (c *RateCipher) Encrypt(userID, rateID string, amountCents int64) (ProtectedRateAmount, error) {
	if c == nil || len(c.keys) == 0 {
		return ProtectedRateAmount{}, ErrInvalidRateCipherConfig
	}
	aead, ok := c.keys[c.activeVersion]
	if !ok {
		return ProtectedRateAmount{}, ErrInvalidRateCipherConfig
	}
	if amountCents <= 0 {
		return ProtectedRateAmount{}, ErrInvalidRateAmount
	}
	aad, err := rateAdditionalData(c.activeVersion, userID, rateID)
	if err != nil {
		return ProtectedRateAmount{}, err
	}
	var plaintext [ratePlaintextSize]byte
	defer clear(plaintext[:])
	// O domínio já usa centavos inteiros positivos; oito bytes preservam int64
	// sem conversões decimais, ponto flutuante ou cópias de valor em texto.
	binary.BigEndian.PutUint64(plaintext[:], uint64(amountCents))
	sealed := aead.Seal(nil, nil, plaintext[:], aad)
	// O esquema tem campos separados. Extraímos o prefixo de 96 bits gerado
	// pela biblioteca e copiamos os slices para não compartilhar seus buffers.
	return ProtectedRateAmount{
		Nonce:      append([]byte(nil), sealed[:rateNonceSize]...),
		Ciphertext: append([]byte(nil), sealed[rateNonceSize:]...),
		KeyVersion: c.activeVersion,
	}, nil
}

// Decrypt só devolve centavos após autenticar envelope, conta e registro.
// Versão ausente, material incorreto e adulteração não expõem detalhes privados.
func (c *RateCipher) Decrypt(userID, rateID string, amount ProtectedRateAmount) (int64, error) {
	if c == nil || len(c.keys) == 0 {
		return 0, ErrInvalidRateCipherConfig
	}
	aead, ok := c.keys[amount.KeyVersion]
	if !ok || len(amount.Nonce) != rateNonceSize || len(amount.Ciphertext) != ratePlaintextSize+aead.Overhead()-rateNonceSize {
		return 0, ErrInvalidProtectedRateAmount
	}
	aad, err := rateAdditionalData(amount.KeyVersion, userID, rateID)
	if err != nil {
		return 0, err
	}
	sealed := make([]byte, 0, len(amount.Nonce)+len(amount.Ciphertext))
	sealed = append(sealed, amount.Nonce...)
	sealed = append(sealed, amount.Ciphertext...)
	plaintext, err := aead.Open(nil, nil, sealed, aad)
	if err != nil {
		return 0, ErrInvalidProtectedRateAmount
	}
	defer clear(plaintext)
	value := binary.BigEndian.Uint64(plaintext)
	if value == 0 || value > math.MaxInt64 {
		return 0, ErrInvalidProtectedRateAmount
	}
	return int64(value), nil
}

func rateAdditionalData(version, userID, rateID string) ([]byte, error) {
	if userID == "" || rateID == "" || strings.TrimSpace(userID) != userID || strings.TrimSpace(rateID) != rateID || !utf8.ValidString(userID) || !utf8.ValidString(rateID) {
		return nil, ErrInvalidRateCipherContext
	}
	// JSON torna a composição inequívoca. Conta, registro, campo/formato e versão
	// são autenticados sem serem cifrados, impedindo transplantar um valor entre eles.
	data, err := json.Marshal([4]string{"agrofin.harvest-rate.amount.v1", version, userID, rateID})
	if err != nil {
		return nil, ErrInvalidRateCipherContext
	}
	return data, nil
}
