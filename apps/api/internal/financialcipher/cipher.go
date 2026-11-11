// Package financialcipher implementa envelopes autenticados para valores financeiros.
package financialcipher

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
)

const (
	// AES-256 exige uma chave de exatamente 32 bytes.
	keySize = 32
	// NewGCMWithRandomNonce gera nonces de 96 bits e os prefixa ao resultado de Seal.
	nonceSize = 12
	// Os domínios usam centavos positivos em int64, representados em oito bytes.
	plaintextSize = 8
)

var (
	ErrInvalidConfig    = errors.New("financialcipher: invalid key configuration")
	ErrInvalidContext   = errors.New("financialcipher: invalid amount context")
	ErrInvalidAmount    = errors.New("financialcipher: amount must be positive")
	ErrInvalidProtected = errors.New("financialcipher: protected amount is invalid")
)

// Amount[P] é o envelope que pode atravessar a fronteira do Repository.
// Ciphertext contém o valor cifrado e a tag de autenticação do GCM; Nonce precisa
// ser guardado para a leitura futura, mas não é secreto. KeyVersion seleciona a
// chave correta quando houver versões antigas que ainda precisem ser lidas.
// Os centavos em claro não fazem parte deste tipo. P é um marcador só de tipo:
// não acrescenta dados ao envelope persistido, mas impede misturar envelopes de
// domínios diferentes no código Go.
type Amount[P any] struct {
	Ciphertext []byte
	Nonce      []byte
	KeyVersion string
}

// Context[P] descreve os dados que serão autenticados junto do valor, sem cifrá-los.
// Purpose separa campos financeiros e domínios; UserID e RecordID impedem mover
// o envelope entre contas ou registros; Attributes acrescenta campos que definem
// o significado do valor, como funcionário, data, tipo e quantidade.
// A mesma sequência e a mesma ordem precisam ser reconstruídas para descriptografar.
// P deve ser o mesmo marcador usado no Cipher[P] e Amount[P] correspondentes.
type Context[P any] struct {
	Purpose    string
	UserID     string
	RecordID   string
	Attributes []string
}

// Keyring mantém as cifras prontas para todas as versões configuradas e seleciona
// a versão ativa para novas gravações. A composição da aplicação carrega as chaves;
// este tipo não as lê do ambiente nem as persiste.
type Keyring struct {
	activeVersion string
	keys          map[string]cipher.AEAD
}

// Cipher[P] oferece operações criptográficas marcadas com P. O marcador não muda
// os bytes cifrados; ele faz o compilador rejeitar o uso de Context ou Amount de
// outro domínio. Os propósitos em Context continuam necessários para autenticação
// em tempo de execução e para reconstruir envelopes persistidos.
type Cipher[P any] struct {
	keyring *Keyring
}

// NewKeyring valida o chaveiro e prepara uma AEAD para cada versão disponível. A versão
// ativa precisa existir no mapa; as demais permanecem carregadas para ler envelopes
// antigos durante uma rotação. Duas versões não podem apontar para os mesmos bytes:
// apenas renomear uma chave não é rotação e não reinicia seus limites de uso.
//
// O nonce aleatório do GCM evita que o chamador tenha de gerá-lo, mas impõe um
// limite operacional por chave que precisa ser acompanhado entre instâncias e
// reinícios. Este construtor não consegue fazer esse acompanhamento global; o
// chamador continua responsável por provisionar e preservar as versões de chave.
func NewKeyring(activeVersion string, keys map[string][]byte) (*Keyring, error) {
	if activeVersion == "" || strings.TrimSpace(activeVersion) != activeVersion {
		return nil, ErrInvalidConfig
	}
	if _, ok := keys[activeVersion]; !ok {
		return nil, ErrInvalidConfig
	}

	// Cada versão recebe uma cifra preparada uma única vez; requisições só leem
	// este mapa, enquanto a seleção da chave de escrita permanece explícita.
	prepared := make(map[string]cipher.AEAD, len(keys))
	seen := make(map[[keySize]byte]bool, len(keys))
	for version, key := range keys {
		if version == "" || strings.TrimSpace(version) != version || !utf8.ValidString(version) || len(key) != keySize {
			return nil, ErrInvalidConfig
		}
		var material [keySize]byte
		copy(material[:], key)
		if seen[material] {
			return nil, ErrInvalidConfig
		}
		seen[material] = true

		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		// Esta variante gera o nonce dentro de Seal e o inclui no resultado. Por isso,
		// Encrypt e Decrypt passam nonce nil ao AEAD e separam o prefixo para a coluna
		// nonce já existente no esquema do banco.
		aead, err := cipher.NewGCMWithRandomNonce(block)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		prepared[version] = aead
	}
	return &Keyring{activeVersion: activeVersion, keys: prepared}, nil
}

// NewCipher cria uma visão do chaveiro para um domínio marcado por P. O mesmo
// Keyring pode ser compartilhado por, por exemplo, Cipher[diaria] e Cipher[colheita],
// mantendo uma única configuração de chaves sem permitir misturar seus envelopes.
func NewCipher[P any](keyring *Keyring) (*Cipher[P], error) {
	if keyring == nil || len(keyring.keys) == 0 {
		return nil, ErrInvalidConfig
	}
	if _, ok := keyring.keys[keyring.activeVersion]; !ok {
		return nil, ErrInvalidConfig
	}
	return &Cipher[P]{keyring: keyring}, nil
}

// Encrypt protege centavos positivos com a versão ativa e autentica Context junto
// do valor. O contexto não fica secreto, mas qualquer alteração nele faz a leitura
// futura falhar; isso impede transplantar uma cifra para outro dono ou lançamento.
// O envelope devolvido separa o nonce aleatório do ciphertext para caber no modelo
// persistido atualmente. A cópia temporária em claro é apagada antes do retorno.
//
// Por exemplo, R$ 5,00 chega como 500 centavos. Encrypt devolve Amount com bytes
// cifrados, nonce e versão da chave; o chamador persiste esse envelope. Para
// recuperar 500, Decrypt precisa receber o mesmo Context usado na criação.
func (c *Cipher[P]) Encrypt(context Context[P], amountCents int64) (Amount[P], error) {
	if c == nil || c.keyring == nil || len(c.keyring.keys) == 0 {
		return Amount[P]{}, ErrInvalidConfig
	}
	aead, ok := c.keyring.keys[c.keyring.activeVersion]
	if !ok {
		return Amount[P]{}, ErrInvalidConfig
	}
	// AAD (dados adicionais autenticados) vincula o valor aos metadados sem cifrá-los.
	// O marcador P não é serializado; Purpose é a separação que também precisa existir
	// em execução para que a aplicação reconstrua os mesmos bytes numa leitura futura.
	aad, err := additionalData(c.keyring.activeVersion, context)
	if err != nil {
		return Amount[P]{}, err
	}
	if !money.IsPositiveCents(amountCents) {
		return Amount[P]{}, ErrInvalidAmount
	}

	// Centavos continuam inteiros: a representação fixa em big-endian evita
	// conversão para decimal ou ponto flutuante antes da cifra.
	var plaintext [plaintextSize]byte
	defer clear(plaintext[:])
	binary.BigEndian.PutUint64(plaintext[:], uint64(amountCents))
	// O nonce nil é obrigatório nesta AEAD específica; o GCM gera um novo nonce
	// aleatório e o prefixa ao valor cifrado e à tag de autenticação.
	sealed := aead.Seal(nil, nil, plaintext[:], aad)
	return Amount[P]{
		// O esquema armazena nonce e ciphertext em colunas separadas. Copiar os bytes
		// evita que o envelope retenha fatias do buffer temporário retornado por Seal.
		Nonce:      append([]byte(nil), sealed[:nonceSize]...),
		Ciphertext: append([]byte(nil), sealed[nonceSize:]...),
		KeyVersion: c.keyring.activeVersion,
	}, nil
}

// Decrypt seleciona a chave pela versão do envelope, reconstrói o AAD e só devolve
// centavos depois que GCM valida a tag. Uma troca de contexto ou qualquer alteração
// nos bytes protegidos resulta em ErrInvalidProtected, sem expor o motivo interno.
// Os bytes em claro são apagados depois da conversão para o inteiro do domínio.
func (c *Cipher[P]) Decrypt(context Context[P], amount Amount[P]) (int64, error) {
	aead, err := c.aeadFor(amount)
	if err != nil {
		return 0, err
	}
	aad, err := additionalData(amount.KeyVersion, context)
	if err != nil {
		return 0, err
	}
	// Seal recebeu os campos concatenados; Open exige reconstruir essa mesma forma.
	// O banco mantém nonce separado por compatibilidade com as colunas existentes.
	sealed := make([]byte, 0, len(amount.Nonce)+len(amount.Ciphertext))
	sealed = append(sealed, amount.Nonce...)
	sealed = append(sealed, amount.Ciphertext...)
	plaintext, err := aead.Open(nil, nil, sealed, aad)
	if err != nil {
		return 0, ErrInvalidProtected
	}
	defer clear(plaintext)
	// AES-GCM autentica bytes, não regras de negócio. Esta checagem adicional impede
	// que um envelope válido, mas fora do domínio de centavos positivos em int64,
	// seja aceito como valor financeiro.
	value := binary.BigEndian.Uint64(plaintext)
	if value == 0 || value > math.MaxInt64 {
		return 0, ErrInvalidProtected
	}
	return int64(value), nil
}

func (c *Cipher[P]) aeadFor(amount Amount[P]) (cipher.AEAD, error) {
	if c == nil || c.keyring == nil || len(c.keyring.keys) == 0 {
		return nil, ErrInvalidConfig
	}
	// A versão vem do envelope persistido. Uma versão desconhecida ou um formato
	// incompatível deve ser tratado como dado protegido inválido, não como valor zero.
	aead, ok := c.keyring.keys[amount.KeyVersion]
	// O nonce é separado e tem 12 bytes. Overhead inclui nonce e tag nesta AEAD;
	// removendo o nonce, restam oito bytes cifrados mais a tag de 16 bytes.
	if !ok || len(amount.Nonce) != nonceSize || len(amount.Ciphertext) != plaintextSize+aead.Overhead()-nonceSize {
		return nil, ErrInvalidProtected
	}
	return aead, nil
}

func additionalData[P any](version string, context Context[P]) ([]byte, error) {
	if !validContextValue(version) || !validContextValue(context.Purpose) ||
		!validContextValue(context.UserID) || !validContextValue(context.RecordID) {
		return nil, ErrInvalidContext
	}
	// A ordem é um contrato persistido: propósito, versão, conta, registro e depois
	// atributos. Alterá-la impediria a leitura de envelopes existentes, mesmo com a
	// chave correta; o JSON em array também evita ambiguidades de concatenação.
	values := make([]string, 0, 4+len(context.Attributes))
	values = append(values, context.Purpose, version, context.UserID, context.RecordID)
	for _, attribute := range context.Attributes {
		if !validContextValue(attribute) {
			return nil, ErrInvalidContext
		}
		values = append(values, attribute)
	}
	// A codificação preserva exatamente o formato de AAD das tarifas e fotografias
	// de colheita já gravadas antes da extração deste pacote compartilhado.
	data, err := json.Marshal(values)
	if err != nil {
		return nil, ErrInvalidContext
	}
	return data, nil
}

func validContextValue(value string) bool {
	// Não normalizamos o texto silenciosamente: espaços ou UTF-8 inválido poderiam
	// gerar duas representações diferentes do mesmo identificador entre gravação e leitura.
	return value != "" && strings.TrimSpace(value) == value && utf8.ValidString(value)
}
