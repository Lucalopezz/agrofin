// Testa a autenticação e a recuperação dos valores financeiros de vendas.
package vendas

import (
	"bytes"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaleCipherAuthenticatesContext(t *testing.T) {
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{7}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keys)
	require.NoError(t, err)
	attrs := []string{"exportacao", "2026-11-12"}
	protected, err := cipher.encrypt("agrofin.sale.unit-price.v1", testOwner, "sale", "item", attrs, 4000)
	require.NoError(t, err)
	amount, err := cipher.decrypt("agrofin.sale.unit-price.v1", testOwner, "sale", "item", attrs, protected)
	require.NoError(t, err)
	assert.EqualValues(t, 4000, amount)
	// A autenticação impede reutilizar o envelope em outro dono, item, tipo ou propósito.
	for _, tc := range []struct {
		name, purpose, owner, item string
		attrs                      []string
	}{
		{"outro dono", "agrofin.sale.unit-price.v1", "outro", "item", attrs},
		{"outro item", "agrofin.sale.unit-price.v1", testOwner, "outro", attrs},
		{"outra classificação", "agrofin.sale.unit-price.v1", testOwner, "item", []string{"mercado", "2026-11-12"}},
		{"outro propósito", "agrofin.sale.receipt.v1", testOwner, "item", attrs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := cipher.decrypt(tc.purpose, tc.owner, "sale", tc.item, tc.attrs, protected)
			assert.Error(t, err)
		})
	}
	protected.Ciphertext[0] ^= 1
	_, err = cipher.decrypt("agrofin.sale.unit-price.v1", testOwner, "sale", "item", attrs, protected)
	assert.Error(t, err)
}

func TestOpenReceiptWithDifferentTimeZone(t *testing.T) {
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{7}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keys)
	require.NoError(t, err)
	instant := time.Date(2026, 11, 12, 12, 30, 0, 123000, time.UTC)
	amount, err := cipher.encrypt("agrofin.sale.receipt.v1", testOwner, "sale", "receipt", []string{instant.Format(time.RFC3339Nano)}, 30000)
	require.NoError(t, err)
	// A data do banco pode representar o mesmo instante no fuso local.
	receipt, err := NewService(nil, cipher).openReceipt(ProtectedReceipt{ID: "receipt", UserID: testOwner, SaleID: "sale", ReceivedAt: instant.In(time.FixedZone("-03", -3*60*60)), Amount: amount})
	require.NoError(t, err)
	assert.EqualValues(t, 30000, receipt.AmountCents)
}
