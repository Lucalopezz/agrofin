package financialcipher

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey(seed byte) []byte {
	return bytes.Repeat([]byte{seed}, keySize)
}

type testPurpose struct{}

func newTestCipher(t *testing.T, activeVersion string, keys map[string][]byte) *Cipher[testPurpose] {
	t.Helper()
	keyring, err := NewKeyring(activeVersion, keys)
	require.NoError(t, err)
	c, err := NewCipher[testPurpose](keyring)
	require.NoError(t, err)
	return c
}

func TestNewRejectsInvalidKeyConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		version string
		keys    map[string][]byte
	}{
		{name: "sem versão", keys: map[string][]byte{"v1": testKey(1)}},
		{name: "sem chaves", version: "v1"},
		{name: "versão ausente", version: "v2", keys: map[string][]byte{"v1": testKey(1)}},
		{name: "chave curta", version: "v1", keys: map[string][]byte{"v1": make([]byte, keySize-1)}},
		{name: "versões repetem material", version: "v1", keys: map[string][]byte{"v1": testKey(1), "v2": testKey(1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewKeyring(tt.version, tt.keys)
			assert.ErrorIs(t, err, ErrInvalidConfig)
			assert.Nil(t, got)
		})
	}
}

func TestCipherRoundTripAndKeyRotation(t *testing.T) {
	old := newTestCipher(t, "v1", map[string][]byte{"v1": testKey(1)})
	context := Context[testPurpose]{Purpose: "agrofin.daily-work.amount.v1", UserID: "user-a", RecordID: "daily-a", Attributes: []string{"employee-a", "2026-11-04"}}
	for _, cents := range []int64{1, 500, math.MaxInt64} {
		amount, err := old.Encrypt(context, cents)
		require.NoError(t, err)
		assert.Equal(t, "v1", amount.KeyVersion)
		assert.Len(t, amount.Nonce, nonceSize)
		assert.Len(t, amount.Ciphertext, 24)
		var plaintext [plaintextSize]byte
		binary.BigEndian.PutUint64(plaintext[:], uint64(cents))
		assert.False(t, bytes.Contains(amount.Ciphertext, plaintext[:]))

		current := newTestCipher(t, "v2", map[string][]byte{"v1": testKey(1), "v2": testKey(2)})
		got, err := current.Decrypt(context, amount)
		require.NoError(t, err)
		assert.Equal(t, cents, got)
		fresh, err := current.Encrypt(context, cents)
		require.NoError(t, err)
		assert.Equal(t, "v2", fresh.KeyVersion)
	}
}

func TestCipherSeparatesContextsAndUsesFreshNonces(t *testing.T) {
	c := newTestCipher(t, "v1", map[string][]byte{"v1": testKey(3)})
	context := Context[testPurpose]{Purpose: "agrofin.daily-work.amount.v1", UserID: "user-a", RecordID: "daily-a", Attributes: []string{"employee-a", "2026-11-04"}}
	first, err := c.Encrypt(context, 500)
	require.NoError(t, err)
	second, err := c.Encrypt(context, 500)
	require.NoError(t, err)
	assert.NotEqual(t, first.Nonce, second.Nonce)

	contexts := []Context[testPurpose]{
		{Purpose: "agrofin.harvest-rate.amount.v1", UserID: "user-a", RecordID: "daily-a"},
		{Purpose: context.Purpose, UserID: "user-b", RecordID: "daily-a", Attributes: context.Attributes},
		{Purpose: context.Purpose, UserID: "user-a", RecordID: "daily-b", Attributes: context.Attributes},
		{Purpose: context.Purpose, UserID: "user-a", RecordID: "daily-a", Attributes: []string{"employee-b", "2026-11-04"}},
	}
	for _, wrongContext := range contexts {
		_, err := c.Decrypt(wrongContext, first)
		assert.ErrorIs(t, err, ErrInvalidProtected)
	}
}

func TestCipherRejectsInvalidInputAndTamperedEnvelope(t *testing.T) {
	c := newTestCipher(t, "v1", map[string][]byte{"v1": testKey(4)})
	context := Context[testPurpose]{Purpose: "agrofin.daily-work.amount.v1", UserID: "user-a", RecordID: "daily-a"}
	for _, cents := range []int64{0, -1, math.MinInt64} {
		_, err := c.Encrypt(context, cents)
		assert.ErrorIs(t, err, ErrInvalidAmount)
	}
	for _, invalidContext := range []Context[testPurpose]{
		{}, {Purpose: "purpose", UserID: "", RecordID: "record"},
		{Purpose: " purpose", UserID: "user", RecordID: "record"},
		{Purpose: "purpose", UserID: "user", RecordID: "record", Attributes: []string{""}},
	} {
		_, err := c.Encrypt(invalidContext, 500)
		assert.ErrorIs(t, err, ErrInvalidContext)
	}

	amount, err := c.Encrypt(context, 500)
	require.NoError(t, err)
	amount.Ciphertext[0] ^= 1
	got, err := c.Decrypt(context, amount)
	assert.ErrorIs(t, err, ErrInvalidProtected)
	assert.Zero(t, got)
	_, err = c.Decrypt(context, Amount[testPurpose]{KeyVersion: "unknown", Nonce: make([]byte, nonceSize), Ciphertext: make([]byte, 24)})
	assert.ErrorIs(t, err, ErrInvalidProtected)
}

func TestCipherPreservesExistingRateAADFormat(t *testing.T) {
	got, err := additionalData("v1", Context[testPurpose]{
		Purpose: "agrofin.harvest-rate.amount.v1", UserID: "user-a", RecordID: "rate-a",
	})
	require.NoError(t, err)
	assert.Equal(t, `["agrofin.harvest-rate.amount.v1","v1","user-a","rate-a"]`, string(got))
}

func TestCipherPreservesExistingAppliedRateAADFormat(t *testing.T) {
	got, err := additionalData("v1", Context[testPurpose]{
		Purpose: "agrofin.harvest-entry.applied-rate.v1",
		UserID:  "user-a", RecordID: "harvest-a",
		Attributes: []string{"employee-a", "2026-11-04", "normal", "10"},
	})
	require.NoError(t, err)
	assert.Equal(t, `["agrofin.harvest-entry.applied-rate.v1","v1","user-a","harvest-a","employee-a","2026-11-04","normal","10"]`, string(got))
}

func TestCipherRejectsAuthenticatedInvalidAmount(t *testing.T) {
	c := newTestCipher(t, "v1", map[string][]byte{"v1": testKey(5)})
	context := Context[testPurpose]{Purpose: "agrofin.daily-work.amount.v1", UserID: "user-a", RecordID: "daily-a"}
	aad, err := additionalData("v1", context)
	require.NoError(t, err)
	for _, invalid := range []uint64{0, 1 << 63, math.MaxUint64} {
		var plaintext [plaintextSize]byte
		binary.BigEndian.PutUint64(plaintext[:], invalid)
		sealed := c.keyring.keys["v1"].Seal(nil, nil, plaintext[:], aad)
		got, err := c.Decrypt(context, Amount[testPurpose]{
			KeyVersion: "v1", Nonce: sealed[:nonceSize], Ciphertext: sealed[nonceSize:],
		})
		assert.ErrorIs(t, err, ErrInvalidProtected)
		assert.Zero(t, got)
	}
}
