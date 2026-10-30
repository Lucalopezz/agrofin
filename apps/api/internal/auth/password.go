package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

const (
	// v=19 identifica a versão de Argon2 usada pelo formato PHC persistido.
	argonVersion = 19
	// Parâmetros iniciais do Argon2id (em KiB, iterações, lanes). Aumentar custo
	// melhora resistência offline, mas também aumenta CPU/memória por login.
	passwordMemory    = 19 * 1024
	passwordTime      = 2
	passwordThreads   = 1
	passwordSaltBytes = 16
	passwordKeyBytes  = 32
	// Contamos caracteres Unicode, não bytes, para que senhas acentuadas não
	// tenham tamanho percebido diferente por causa da codificação UTF-8.
	minimumPasswordRunes = 15
	maximumPasswordRunes = 128
)

// A lista pequena bloqueia escolhas notoriamente fracas até o produto adotar uma lista de senhas comprometidas atualizável.
var commonPasswords = map[string]struct{}{
	"123456789012345":  {},
	"password1234567":  {},
	"passwordpassword": {},
	"qwertyuiop12345":  {},
	"senha1234567890":  {},
	"adminadminadmin":  {},
	"letmeinletmein1":  {},
	"agrofinagrofin1":  {},
}

func createPasswordHash(password, loginName string) (string, error) {
	// Cadastro e login normalizam a senha da mesma maneira; isso evita que duas
	// formas Unicode visualmente iguais virem credenciais diferentes.
	password, err := validateNewPassword(password, loginName)
	if err != nil {
		return "", err
	}
	return hashPasswordMaterial(password)
}

func hashPasswordMaterial(password string) (string, error) {
	// Salt aleatório e exclusivo impede que senhas iguais produzam hashes iguais
	// e dificulta o uso de tabelas pré-computadas.
	salt := make([]byte, passwordSaltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", errors.New("não foi possível gerar o salt da senha")
	}
	return encodePasswordHash(password, salt, passwordMemory, passwordTime, passwordThreads), nil
}

func encodePasswordHash(password string, salt []byte, memory uint32, iterations uint32, threads uint8) string {
	// O formato PHC guarda algoritmo, versão, parâmetros, salt e resultado juntos,
	// permitindo verificar a senha sem armazenar uma chave de decifragem.
	digest := argon2.IDKey([]byte(password), salt, iterations, memory, threads, passwordKeyBytes)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argonVersion,
		memory,
		iterations,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest),
	)
}

func verifyPasswordHash(encodedHash, password string) bool {
	// O conteúdo vem do banco; limitar tamanho/parâmetros evita que dado corrompido
	// ou adulterado transforme um login em alocação arbitrária de memória.
	if len(encodedHash) > 512 {
		performDummyPasswordHash(password)
		return false
	}
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		performDummyPasswordHash(password)
		return false
	}

	parameters, ok := parseArgon2Parameters(parts[3])
	if !ok {
		performDummyPasswordHash(password)
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < passwordSaltBytes || len(salt) > 64 {
		performDummyPasswordHash(password)
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		performDummyPasswordHash(password)
		return false
	}

	actual := argon2.IDKey([]byte(password), salt, parameters.iterations, parameters.memory, parameters.threads, uint32(len(expected)))
	// Comparação em tempo constante evita que o tempo revele quantos bytes do
	// resultado calculado já coincidiam com o valor persistido.
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

type argon2Parameters struct {
	memory     uint32
	iterations uint32
	threads    uint8
}

func parseArgon2Parameters(value string) (argon2Parameters, bool) {
	// O parser aceita exatamente m, t e p uma vez cada; parâmetros extras ou
	// repetidos não devem alterar silenciosamente o significado do hash.
	var parameters argon2Parameters
	seen := make(map[string]bool, 3)
	for _, part := range strings.Split(value, ",") {
		key, raw, ok := strings.Cut(part, "=")
		if !ok || seen[key] {
			return argon2Parameters{}, false
		}
		seen[key] = true
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return argon2Parameters{}, false
		}
		switch key {
		case "m":
			parameters.memory = uint32(parsed)
		case "t":
			parameters.iterations = uint32(parsed)
		case "p":
			if parsed > 255 {
				return argon2Parameters{}, false
			}
			parameters.threads = uint8(parsed)
		default:
			return argon2Parameters{}, false
		}
	}

	// Limites rígidos evitam que um hash adulterado no banco solicite memória ou CPU arbitrárias.
	if len(seen) != 3 || parameters.memory < 8*uint32(parameters.threads) || parameters.memory > 64*1024 || parameters.iterations == 0 || parameters.iterations > 5 || parameters.threads == 0 || parameters.threads > 4 {
		return argon2Parameters{}, false
	}
	return parameters, true
}

func performDummyPasswordHash(password string) {
	// Usuários inexistentes e hashes inválidos ainda pagam o custo do Argon2id para reduzir diferenças de tempo no login.
	_ = argon2.IDKey([]byte(password), []byte("agrofin-login-dummy-salt"), passwordTime, passwordMemory, passwordThreads, passwordKeyBytes)
}

func validateNewPassword(password, loginName string) (string, error) {
	password, err := normalizePasswordInput(password)
	if err != nil {
		return "", err
	}
	runeCount := utf8.RuneCountInString(password)
	if runeCount < minimumPasswordRunes || runeCount > maximumPasswordRunes {
		return "", invalidInput("senha deve ter entre 15 e 128 caracteres")
	}

	// Bloqueio pequeno e local para senhas conhecidamente fracas; não substitui
	// uma lista atualizável de senhas vazadas, ainda não adotada pelo projeto.
	folded := strings.ToLower(password)
	if _, blocked := commonPasswords[folded]; blocked || (loginName != "" && folded == loginName) {
		return "", invalidInput("escolha uma senha menos comum")
	}
	return password, nil
}

func normalizePasswordInput(password string) (string, error) {
	// Limite em bytes evita trabalho desnecessário antes da normalização; o limite
	// em runes depois protege o contrato de senha independentemente da escrita UTF-8.
	if !utf8.ValidString(password) || len(password) > 512 {
		return "", invalidInput("senha inválida")
	}
	password = norm.NFC.String(password)
	if utf8.RuneCountInString(password) > maximumPasswordRunes {
		return "", invalidInput("senha excede 128 caracteres")
	}
	return password, nil
}

func normalizeLoginName(loginName string) (string, error) {
	// Login é case-insensitive e usa um alfabeto pequeno para evitar ambiguidades
	// de comparação; ponto, hífen e sublinhado só são aceitos depois do primeiro char.
	loginName = strings.ToLower(strings.TrimSpace(loginName))
	if len(loginName) < 3 || len(loginName) > 64 {
		return "", invalidInput("nome de acesso deve ter entre 3 e 64 caracteres")
	}
	for index, char := range loginName {
		valid := char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || (index > 0 && (char == '.' || char == '_' || char == '-'))
		if !valid {
			return "", invalidInput("nome de acesso deve usar letras, números, ponto, hífen ou sublinhado")
		}
	}
	return loginName, nil
}
