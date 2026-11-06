// Package httpjson reúne regras comuns para decodificar corpos JSON de requisições HTTP.
package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

const maxBodyBytes = 4 * 1024

var (
	ErrUnsupportedMediaType = errors.New("tipo de conteúdo não suportado")
	ErrBodyTooLarge         = errors.New("corpo da requisição excede 4 KiB")
	ErrInvalidJSON          = errors.New("corpo não contém um único objeto JSON válido")
)

// Decode exige application/json, limita o corpo, rejeita campos desconhecidos
// e aceita somente um valor JSON. O retorno é ponteiro para que os handlers
// possam distinguir JSON null de um objeto vazio e aplicar a mensagem do domínio.
func Decode[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, ErrUnsupportedMediaType
	}

	// O mesmo limite evita que endpoints pequenos carreguem corpos sem tamanho
	// definido; MaxBytesReader também permite classificar excesso como HTTP 413.
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var value *T
	if err := decoder.Decode(&value); err != nil {
		return nil, classifyDecodeError(err)
	}

	// Decodificar uma segunda vez exige EOF e rejeita tanto outro objeto quanto
	// lixo após o JSON válido, sem precisar ler o corpo inteiro para memória.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, classifyDecodeError(err)
	}
	return value, nil
}

func classifyDecodeError(err error) error {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		return ErrBodyTooLarge
	}
	return ErrInvalidJSON
}
