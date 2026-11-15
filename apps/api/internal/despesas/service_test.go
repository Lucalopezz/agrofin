// Testa a validação das despesas pagas e suas datas civis.
package despesas

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	owner := "11111111-1111-4111-8111-111111111111"
	valid := Input{PaidAt: time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC), Category: " Energia ", Description: " Conta ", AmountCents: 1}
	e, err := normalize(owner, valid)
	require.NoError(t, err)
	assert.Equal(t, "Energia", e.Category)
	assert.Equal(t, "Conta", e.Description)
	assert.Equal(t, 0, e.PaidAt.Hour())
	for _, tc := range []struct {
		name string
		in   Input
	}{{"data vazia", Input{Category: "Energia", AmountCents: 1}}, {"categoria vazia", Input{PaidAt: valid.PaidAt, Category: "  ", AmountCents: 1}}, {"valor zero", Input{PaidAt: valid.PaidAt, Category: "Energia"}}, {"valor negativo", Input{PaidAt: valid.PaidAt, Category: "Energia", AmountCents: -1}}} {
		t.Run(tc.name, func(t *testing.T) { _, err := normalize(owner, tc.in); assert.ErrorIs(t, err, ErrInvalid) })
	}
	_, err = normalize("inválido", valid)
	assert.ErrorIs(t, err, ErrInvalid)
}
