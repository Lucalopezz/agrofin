package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

const (
	argonVersion         = 19
	passwordMemory       = 19 * 1024
	passwordTime         = 2
	passwordThreads      = 1
	passwordSaltBytes    = 16
	passwordKeyBytes     = 32
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
	password, err := validateNewPassword(password, loginName)
	if err != nil {
		return "", err
	}
	return hashPasswordMaterial(password)
}

func hashPasswordMaterial(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", errors.New("não foi possível gerar o salt da senha")
	}
	return encodePasswordHash(password, salt, passwordMemory, passwordTime, passwordThreads), nil
}

func encodePasswordHash(password string, salt []byte, memory uint32, iterations uint32, threads uint8) string {
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

func validateNewPassword(password, loginName string) (string, error) {
	password, err := normalizePasswordInput(password)
	if err != nil {
		return "", err
	}
	runeCount := utf8.RuneCountInString(password)
	if runeCount < minimumPasswordRunes || runeCount > maximumPasswordRunes {
		return "", invalidInput("senha deve ter entre 15 e 128 caracteres")
	}

	folded := strings.ToLower(password)
	if _, blocked := commonPasswords[folded]; blocked || (loginName != "" && folded == loginName) {
		return "", invalidInput("escolha uma senha menos comum")
	}
	return password, nil
}

func normalizePasswordInput(password string) (string, error) {
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
