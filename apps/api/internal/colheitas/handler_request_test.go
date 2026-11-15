// Testa validação de entrada de tarifas e colheitas antes do serviço.
package colheitas

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateAndHarvestRequestValidation(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		rate       bool
	}{
		{"tarifa sem valor", `{"box_type":"normal","amount_cents":0,"valid_from":"2026-11-01"}`, true},
		{"tipo inválido", `{"box_type":"mercado","amount_cents":100,"valid_from":"2026-11-01"}`, true},
		{"fim inválido", `{"box_type":"normal","amount_cents":100,"valid_from":"2026-11-01","valid_until":"2026-11-01"}`, true},
		{"funcionário inválido", `{"employee_id":"errado","harvest_date":"2026-11-01","items":[{"box_type":"normal","quantity":1}]}`, false},
		{"quantidade inválida", `{"employee_id":"11111111-1111-4111-8111-111111111111","harvest_date":"2026-11-01","items":[{"box_type":"normal","quantity":0}]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			var ok bool
			if tc.rate {
				_, ok = decodeCreateRateRequest(w, r)
			} else {
				_, ok = decodeCreateHarvestRequest(w, r)
			}
			assert.False(t, ok)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
