// Testa cifra e autenticação dos valores de diária.
package diarias

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyWorkCipherRoundTripAndContextBinding(t *testing.T) {
	keyring, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{8}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keyring)
	require.NoError(t, err)
	date := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)

	for _, cents := range []int64{1, 8000, math.MaxInt64} {
		protected, err := cipher.EncryptDailyWorkAmount("user-a", "daily-a", validEmployeeID, date, cents)
		require.NoError(t, err)
		assert.Equal(t, "v1", protected.KeyVersion)
		assert.Len(t, protected.Nonce, 12)
		got, err := cipher.DecryptDailyWorkAmount("user-a", "daily-a", validEmployeeID, date, protected)
		require.NoError(t, err)
		assert.Equal(t, cents, got)
	}

	protected, err := cipher.EncryptDailyWorkAmount("user-a", "daily-a", validEmployeeID, date, 8000)
	require.NoError(t, err)
	tests := []struct {
		name       string
		userID     string
		dailyWork  string
		employeeID string
		date       time.Time
	}{
		{name: "outra conta", userID: "user-b", dailyWork: "daily-a", employeeID: validEmployeeID, date: date},
		{name: "outro lançamento", userID: "user-a", dailyWork: "daily-b", employeeID: validEmployeeID, date: date},
		{name: "outro funcionário", userID: "user-a", dailyWork: "daily-a", employeeID: "b1b2c3d4-1234-4567-89ab-0123456789ab", date: date},
		{name: "outra data", userID: "user-a", dailyWork: "daily-a", employeeID: validEmployeeID, date: date.AddDate(0, 0, 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cipher.DecryptDailyWorkAmount(tt.userID, tt.dailyWork, tt.employeeID, tt.date, protected)
			assert.ErrorIs(t, err, ErrInvalidProtectedDailyWorkAmount)
			assert.Zero(t, got)
		})
	}
}

func TestDailyWorkCipherRejectsInvalidInputs(t *testing.T) {
	keyring, err := financialcipher.NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{9}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keyring)
	require.NoError(t, err)
	date := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)

	_, err = NewCipher(nil)
	assert.ErrorIs(t, err, ErrInvalidDailyWorkCipherConfig)
	_, err = cipher.EncryptDailyWorkAmount("user", "daily", validEmployeeID, date, 0)
	assert.ErrorIs(t, err, ErrInvalidDailyWorkAmount)
	_, err = cipher.EncryptDailyWorkAmount("user", "daily", "", date, 500)
	assert.ErrorIs(t, err, ErrInvalidDailyWorkCipherContext)
	_, err = cipher.EncryptDailyWorkAmount("user", "daily", validEmployeeID, time.Time{}, 500)
	assert.ErrorIs(t, err, ErrInvalidDailyWorkCipherContext)

	protected, err := cipher.EncryptDailyWorkAmount("user", "daily", validEmployeeID, date, 500)
	require.NoError(t, err)
	protected.Ciphertext[0] ^= 1
	got, err := cipher.DecryptDailyWorkAmount("user", "daily", validEmployeeID, date, protected)
	assert.ErrorIs(t, err, ErrInvalidProtectedDailyWorkAmount)
	assert.Zero(t, got)
}
