// Regressão para entradas inválidas no cadastro de insumos.
package insumos

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeCreateRequestRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		status      int
	}{
		{"nome vazio", `{"name":"  ","category":"fertilizante","default_unit":"kg"}`, "application/json", http.StatusBadRequest},
		{"categoria desconhecida", `{"name":"Adubo","category":"outro","default_unit":"kg"}`, "application/json", http.StatusBadRequest},
		{"unidade ausente", `{"name":"Adubo","category":"fertilizante"}`, "application/json", http.StatusBadRequest},
		{"observação com caractere nulo", `{"name":"Adubo","category":"fertilizante","default_unit":"kg","notes":"\u0000"}`, "application/json", http.StatusBadRequest},
		{"estado definido pelo cliente", `{"name":"Adubo","category":"fertilizante","default_unit":"kg","stock_control_state":"tracked"}`, "application/json", http.StatusBadRequest},
		{"saldo inicial no cadastro", `{"name":"Adubo","category":"fertilizante","default_unit":"kg","initial_stock":0}`, "application/json", http.StatusBadRequest},
		{"corpo nulo", "null", "application/json", http.StatusBadRequest},
		{"tipo de mídia incorreto", "{}", "text/plain", http.StatusUnsupportedMediaType},
		{"corpo acima do limite", `{"notes":"` + strings.Repeat("a", 4096) + `"}`, "application/json", http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/input-products", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()

			_, ok := decodeCreateRequest(response, request)

			assert.False(t, ok)
			assert.Equal(t, tt.status, response.Code)
		})
	}
}
