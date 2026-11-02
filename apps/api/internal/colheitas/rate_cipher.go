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

// ProtectedRateAmount corresponde ao resultado da criptografia armazenado
// Somente estes bytes e a versão, nunca os centavos em claro, poderão ir ao banco.
type ProtectedRateAmount struct {
	// contém o valor criptografado e sua tag de autenticação.
	// Ciphertext: []byte{0x91, 0xAF, 0x23, ...}
	Ciphertext []byte
	// Contém o nonce utilizado para criptografar aquele valor.
	// Na descriptografia, precisamos utilizar o mesmo nonce.
	Nonce      []byte
	KeyVersion string // v1
}

// RateCipher protege valores de tarifa com AES-256-GCM. As chaves chegam em
// memória pela composição; o componente não as gera, lê do banco ou persiste.
type RateCipher struct {
	// Qual chave será usada para novas cifras, v1, v2
	activeVersion string
	// map com as especificações de cada versão, para descriptografar valores antigos
	keys map[string]cipher.AEAD
}

// NewRateCipher usa a versão ativa para novas cifras e mantém versões anteriores
// disponíveis para recuperação. Trocar a versão ativa exige reconstruir a instância.
func NewRateCipher(activeVersion string, keys map[string][]byte) (*RateCipher, error) {
	if activeVersion == "" || strings.TrimSpace(activeVersion) != activeVersion {
		return nil, ErrInvalidRateCipherConfig
	}
	// A versão ativa precisa estar presente no mapa de chaves, mesmo que seja a única.
	if _, ok := keys[activeVersion]; !ok {
		return nil, ErrInvalidRateCipherConfig
	}
	// Map que sera retornado com a cifra preparada para cada versão
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

		// Impede o uso da mesma chave com versoes diferentes
		if seen[material] {
			return nil, ErrInvalidRateCipherConfig
		}
		seen[material] = true
		// Cria a cifra AES-256-GCM para cada versão. A biblioteca gera o nonce aleatório
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
		// Guardar cifras
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
	// AAD autentica a versão, a conta e o registro, sem cifrar. O envelope não pode
	// ser transplantado para outra conta, registro ou versão sem falhar na autenticação.
	aad, err := rateAdditionalData(c.activeVersion, userID, rateID)
	if err != nil {
		return ProtectedRateAmount{}, err
	}
	// O domínio já usa centavos inteiros positivos; oito bytes preservam int64
	// sem conversões decimais, ponto flutuante ou cópias de valor em texto.
	var plaintext [ratePlaintextSize]byte
	defer clear(plaintext[:])

	// Transforma o int64 em uint64 para evitar problemas de sinal e usa big-endian
	// 500 cents -> 00 00 00 00 00 00 01 F4
	binary.BigEndian.PutUint64(plaintext[:], uint64(amountCents))
	// Criptografa e autentica o valor, gerando nonce aleatório de 96 bits e tag de 128 bits.
	sealed := aead.Seal(nil, nil, plaintext[:], aad)
	// O esquema tem campos separados. Extraímos o prefixo de 96 bits gerado
	// pela biblioteca e copiamos os slices para não compartilhar seus buffers.
	return ProtectedRateAmount{
		// separa o nonce do restante do envelope, que inclui a tag de autenticação
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
	// Recupera a cifra correspondente à versão do envelope. A versão ativa não é obrigatória
	aead, ok := c.keys[amount.KeyVersion]
	// Validações de integridade do envelope: versão, nonce e tamanho do ciphertext 8 + 28 - 12 = 24 bytes
	if !ok || len(amount.Nonce) != rateNonceSize || len(amount.Ciphertext) != ratePlaintextSize+aead.Overhead()-rateNonceSize {
		return 0, ErrInvalidProtectedRateAmount
	}

	aad, err := rateAdditionalData(amount.KeyVersion, userID, rateID)
	if err != nil {
		return 0, err
	}
	// Na encriptaçao, foi separado o nonce do ciphertext, temos que juntar
	sealed := make([]byte, 0, len(amount.Nonce)+len(amount.Ciphertext))
	sealed = append(sealed, amount.Nonce...)
	sealed = append(sealed, amount.Ciphertext...)
	// Descriptografa e autentica o envelope, devolvendo os centavos em claro.
	plaintext, err := aead.Open(nil, nil, sealed, aad)
	if err != nil {
		return 0, ErrInvalidProtectedRateAmount
	}
	defer clear(plaintext)
	// Converte de volta para int64, validando o domínio positivo. O envelope não pode
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

// EXEMPLO DE FUNCIONAMENTO DO AES-256-GCM
//
// Imagine que uma caixa de limão colhida custa R$ 5,00.
// No sistema, esse valor é representado como 500 centavos.
//
// 1. CRIPTOGRAFANDO O VALOR
//
//    protected, err := rateCipher.Encrypt("user-123", "rate-456", 500)
//
//    O resultado possui três informações:
//
//    ProtectedRateAmount{
//        Ciphertext: []byte{...}, // Valor criptografado + tag de autenticação (24 bytes)
//        Nonce:      []byte{...}, // Número aleatório gerado para essa operação (12 bytes)
//        KeyVersion: "v1",       // Versão da chave utilizada
//    }
//
//    Esses três campos são armazenados no PostgreSQL.
//    O valor original (500) não é armazenado em texto puro.
//
// 2. DESCRIPTOGRAFANDO O VALOR
//
//    amount, err := rateCipher.Decrypt("user-123", "rate-456", protected)
//
//    Resultado:
//        amount = 500
//        err = nil
//
//    O sistema recupera o valor original de R$ 5,00.
//
// 3. CRIPTOGRAFANDO O MESMO VALOR NOVAMENTE
//
//    first, _ := rateCipher.Encrypt("user-123", "rate-456", 500)
//    second, _ := rateCipher.Encrypt("user-123", "rate-456", 500)
//
//    first.Ciphertext != second.Ciphertext  // Normalmente true
//    first.Nonce != second.Nonce            // Normalmente true
//
//    Isso acontece porque cada operação utiliza um nonce aleatório.
//    Mesmo assim, ambos podem ser descriptografados para 500 centavos.
//
// 4. TENTANDO DESCRIPTOGRAFAR COM OUTRO USUÁRIO
//
//    amount, err := rateCipher.Decrypt("user-999", "rate-456", protected)
//
//    Resultado:
//        amount = 0
//        err = ErrInvalidProtectedRateAmount
//
//    O erro acontece porque o userID faz parte do AAD (dados autenticados).
//    O valor foi criptografado para "user-123", não para "user-999".
//
// 5. TENTANDO UTILIZAR O VALOR EM OUTRA TARIFA
//
//    amount, err := rateCipher.Decrypt("user-123", "rate-999", protected)
//
//    Resultado:
//        amount = 0
//        err = ErrInvalidProtectedRateAmount
//
//    Isso impede copiar o valor criptografado de uma tarifa para outra,
//    mesmo quando ambas pertencem ao mesmo usuário.
//
// 6. ALTERANDO OS BYTES CRIPTOGRAFADOS
//
//    adulterado := ProtectedRateAmount{
//        Ciphertext: append([]byte(nil), protected.Ciphertext...),
//        Nonce:      append([]byte(nil), protected.Nonce...),
//        KeyVersion: protected.KeyVersion,
//    }
//
//    adulterado.Ciphertext[0] ^= 0x01
//
//    amount, err := rateCipher.Decrypt("user-123", "rate-456", adulterado)
//
//    Resultado:
//        amount = 0
//        err = ErrInvalidProtectedRateAmount
//
//    O AES-GCM detecta que o conteúdo foi modificado e rejeita a operação.
//
// RESUMO:
//
//    Valor original:      500 centavos (R$ 5,00)
//    Criptografia:        AES-256-GCM
//    Chave:               32 bytes (secreta)
//    Nonce:               12 bytes (não secreto)
//    Ciphertext:          24 bytes (8 bytes cifrados + 16 bytes de autenticação)
//    Versão da chave:     "v1"
//    Dados autenticados:  versão + userID + rateID + identificador do campo
//
// A criptografia protege a confidencialidade do valor.
// A autenticação detecta alterações e impede transplantar o valor
// para outra conta ou tarifa sem que a descriptografia falhe.
