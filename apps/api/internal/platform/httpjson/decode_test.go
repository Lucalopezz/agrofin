package httpjson

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sampleRequest struct {
	Name string `json:"name"`
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantName    string
		wantNil     bool
		wantErr     error
	}{
		{name: "objeto válido com parâmetro de charset", contentType: "application/json; charset=utf-8", body: `{"name":"João"}`, wantName: "João"},
		{name: "null é preservado para o domínio validar", contentType: "application/json", body: `null`, wantNil: true},
		{name: "tipo de conteúdo ausente", body: `{}`, wantErr: ErrUnsupportedMediaType},
		{name: "JSON malformado", contentType: "application/json", body: `{"name":`, wantErr: ErrInvalidJSON},
		{name: "campo desconhecido", contentType: "application/json", body: `{"name":"João","extra":true}`, wantErr: ErrInvalidJSON},
		{name: "segundo objeto", contentType: "application/json", body: `{"name":"João"}{}`, wantErr: ErrInvalidJSON},
		{name: "conteúdo após objeto", contentType: "application/json", body: `{"name":"João"} lixo`, wantErr: ErrInvalidJSON},
		{name: "corpo maior que limite", contentType: "application/json", body: `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`, wantErr: ErrBodyTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			got, err := Decode[sampleRequest](httptest.NewRecorder(), req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.Name)
		})
	}
}
