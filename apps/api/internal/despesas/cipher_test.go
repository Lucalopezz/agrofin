// Testa a proteção dos valores financeiros e seu contexto autenticado.
package despesas

import (
	"bytes"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpenseCipherContext(t *testing.T) {
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{3}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keys)
	require.NoError(t, err)
	expense := Expense{ID: "despesa", UserID: "conta", Category: "Energia", PaidAt: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), AmountCents: 12345}
	protected, err := cipher.protect(expense)
	require.NoError(t, err)
	opened, err := cipher.open(protected)
	require.NoError(t, err)
	assert.EqualValues(t, 12345, opened.AmountCents)
	// Metadados vinculados ao envelope impedem mover o valor para outra despesa.
	altered := protected
	altered.Expense.ID = "outra"
	_, err = cipher.open(altered)
	assert.Error(t, err)
	altered = protected
	altered.Expense.Category = "Outro"
	_, err = cipher.open(altered)
	assert.Error(t, err)
	protected.Amount.Ciphertext[0] ^= 1
	_, err = cipher.open(protected)
	assert.Error(t, err)
}

func TestOpenPreviousExpenseEnvelope(t *testing.T) {
	keys, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{3}, 32)})
	require.NoError(t, err)
	previous, err := financialcipher.NewCipher[expensePurpose](keys)
	require.NoError(t, err)
	expense := Expense{ID: "despesa", UserID: "conta", Category: "Energia", PaidAt: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)}
	// Simula o contexto usado antes de extrair a cifra do serviço.
	amount, err := previous.Encrypt(financialcipher.Context[expensePurpose]{Purpose: "agrofin.expense.amount.v1", UserID: "conta", RecordID: "despesa", Attributes: []string{"2026-11-15", "Energia"}}, 12345)
	require.NoError(t, err)
	cipher, err := NewCipher(keys)
	require.NoError(t, err)
	opened, err := cipher.open(protectedExpense{Expense: expense, Amount: amount})
	require.NoError(t, err)
	assert.EqualValues(t, 12345, opened.AmountCents)
}
