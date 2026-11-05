package colheitas

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Material determinístico é exclusivo das fixtures, nunca de uma chave real.
func testRateKey(seed byte) []byte {
	return bytes.Repeat([]byte{seed}, 32)
}

func TestNewRateCipher(t *testing.T) {
	tests := []struct {
		name    string
		version string
		keys    map[string][]byte
	}{
		{name: "sem versão ativa", keys: map[string][]byte{"v1": testRateKey(1)}},
		{name: "sem chaves", version: "v1"},
		{name: "versão ativa ausente", version: "v2", keys: map[string][]byte{"v1": testRateKey(1)}},
		{name: "versão ativa com espaços", version: " v1", keys: map[string][]byte{"v1": testRateKey(1)}},
		{name: "versão histórica vazia", version: "v1", keys: map[string][]byte{"v1": testRateKey(1), "": testRateKey(2)}},
		{name: "versão histórica com espaços", version: "v1", keys: map[string][]byte{"v1": testRateKey(1), "v2 ": testRateKey(2)}},
		{name: "versão com utf8 inválido", version: "v1", keys: map[string][]byte{"v1": testRateKey(1), string([]byte{0xff}): testRateKey(2)}},
		{name: "aes 128 não pertence ao formato", version: "v1", keys: map[string][]byte{"v1": make([]byte, 16)}},
		{name: "aes 192 não pertence ao formato", version: "v1", keys: map[string][]byte{"v1": make([]byte, 24)}},
		{name: "chave truncada", version: "v1", keys: map[string][]byte{"v1": make([]byte, 31)}},
		{name: "chave excessiva", version: "v1", keys: map[string][]byte{"v1": make([]byte, 33)}},
		{name: "chave histórica inválida", version: "v1", keys: map[string][]byte{"v1": testRateKey(1), "v2": nil}},
		{name: "renomear material não é rotação", version: "v2", keys: map[string][]byte{"v1": testRateKey(1), "v2": testRateKey(1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRateCipher(tt.version, tt.keys)
			assert.ErrorIs(t, err, ErrInvalidRateCipherConfig)
			assert.Nil(t, got)
		})
	}
}

func TestRateCipherRoundTripAndRecovery(t *testing.T) {
	c, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(1)})
	require.NoError(t, err)
	for _, cents := range []int64{1, 500, math.MaxInt64} {
		amount, err := c.Encrypt("user-a", "rate-a", cents)
		require.NoError(t, err)
		assert.Equal(t, "v1", amount.KeyVersion)
		assert.Len(t, amount.Nonce, 12)
		assert.Len(t, amount.Ciphertext, 24)
		var plaintext [8]byte
		binary.BigEndian.PutUint64(plaintext[:], uint64(cents))
		assert.False(t, bytes.Contains(amount.Ciphertext, plaintext[:]), "o envelope não contém a representação em claro")

		// Simula recuperar somente o envelope salvo e a chave mantida separadamente.
		// Nenhum campo de centavos acompanha o envelope serializado.
		serialized, err := json.Marshal(amount)
		require.NoError(t, err)
		var restored ProtectedRateAmount
		require.NoError(t, json.Unmarshal(serialized, &restored))
		recovered, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(1)})
		require.NoError(t, err)
		got, err := recovered.Decrypt("user-a", "rate-a", restored)
		require.NoError(t, err)
		assert.Equal(t, cents, got)
	}
	first, err := c.Encrypt("user-a", "rate-a", 500)
	require.NoError(t, err)
	second, err := c.Encrypt("user-a", "rate-a", 500)
	require.NoError(t, err)
	assert.NotEqual(t, first.Nonce, second.Nonce, "repetir um valor exige um nonce novo")
	assert.NotEqual(t, first.Ciphertext, second.Ciphertext)
}

func TestRateCipherRejectsInvalidInput(t *testing.T) {
	c, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(1)})
	require.NoError(t, err)
	for _, cents := range []int64{0, -1, math.MinInt64} {
		got, err := c.Encrypt("user-a", "rate-a", cents)
		assert.ErrorIs(t, err, ErrInvalidRateAmount)
		assert.Equal(t, ProtectedRateAmount{}, got)
	}
	valid, err := c.Encrypt("user-a", "rate-a", 500)
	require.NoError(t, err)
	for _, ids := range [][2]string{{"", "rate-a"}, {"user-a", ""}, {" user-a", "rate-a"}, {"user-a", "rate-a "}, {string([]byte{0xff}), "rate-a"}} {
		got, err := c.Encrypt(ids[0], ids[1], 500)
		assert.ErrorIs(t, err, ErrInvalidRateCipherContext)
		assert.Equal(t, ProtectedRateAmount{}, got)
		cents, err := c.Decrypt(ids[0], ids[1], valid)
		assert.ErrorIs(t, err, ErrInvalidRateCipherContext)
		assert.Zero(t, cents)
	}
	for _, invalid := range []*RateCipher{nil, {}} {
		_, err := invalid.Encrypt("user-a", "rate-a", 500)
		assert.ErrorIs(t, err, ErrInvalidRateCipherConfig)
		_, err = invalid.Decrypt("user-a", "rate-a", valid)
		assert.ErrorIs(t, err, ErrInvalidRateCipherConfig)
	}
}

func TestRateCipherRejectsTampering(t *testing.T) {
	c, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(1), "v2": testRateKey(2)})
	require.NoError(t, err)
	tests := []struct {
		name   string
		change func(*ProtectedRateAmount)
	}{
		{name: "cifra alterada", change: func(a *ProtectedRateAmount) { a.Ciphertext[0] ^= 1 }},
		{name: "tag alterada", change: func(a *ProtectedRateAmount) { a.Ciphertext[len(a.Ciphertext)-1] ^= 1 }},
		{name: "nonce alterado", change: func(a *ProtectedRateAmount) { a.Nonce[0] ^= 1 }},
		{name: "cifra truncada", change: func(a *ProtectedRateAmount) { a.Ciphertext = a.Ciphertext[:23] }},
		{name: "cifra excessiva", change: func(a *ProtectedRateAmount) { a.Ciphertext = append(a.Ciphertext, 0) }},
		{name: "nonce truncado", change: func(a *ProtectedRateAmount) { a.Nonce = a.Nonce[:11] }},
		{name: "nonce excessivo", change: func(a *ProtectedRateAmount) { a.Nonce = append(a.Nonce, 0) }},
		{name: "versão desconhecida", change: func(a *ProtectedRateAmount) { a.KeyVersion = "v3" }},
		{name: "versão trocada", change: func(a *ProtectedRateAmount) { a.KeyVersion = "v2" }},
		{name: "envelope vazio", change: func(a *ProtectedRateAmount) { *a = ProtectedRateAmount{} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := c.Encrypt("user-a", "rate-a", 500)
			require.NoError(t, err)
			tt.change(&amount)
			got, err := c.Decrypt("user-a", "rate-a", amount)
			assert.ErrorIs(t, err, ErrInvalidProtectedRateAmount)
			assert.Zero(t, got, "uma falha nunca devolve valor parcialmente decifrado")
			assert.NotContains(t, err.Error(), "500")
		})
	}
	amount, err := c.Encrypt("user-a", "rate-a", 500)
	require.NoError(t, err)
	for _, ids := range [][2]string{{"user-b", "rate-a"}, {"user-a", "rate-b"}} {
		got, err := c.Decrypt(ids[0], ids[1], amount)
		assert.ErrorIs(t, err, ErrInvalidProtectedRateAmount)
		assert.Zero(t, got)
	}
}

func TestRateCipherKeyRotation(t *testing.T) {
	old, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(1)})
	require.NoError(t, err)
	previous, err := old.Encrypt("user-a", "rate-a", 500)
	require.NoError(t, err)
	current, err := NewRateCipher("v2", map[string][]byte{"v1": testRateKey(1), "v2": testRateKey(2)})
	require.NoError(t, err)
	got, err := current.Decrypt("user-a", "rate-a", previous)
	require.NoError(t, err)
	assert.EqualValues(t, 500, got)
	newAmount, err := current.Encrypt("user-a", "rate-a", 600)
	require.NoError(t, err)
	assert.Equal(t, "v2", newAmount.KeyVersion)
	got, err = current.Decrypt("user-a", "rate-a", newAmount)
	require.NoError(t, err)
	assert.EqualValues(t, 600, got)

	// Não há tentativa silenciosa com outra chave quando falta a versão antiga.
	withoutOld, err := NewRateCipher("v2", map[string][]byte{"v2": testRateKey(2)})
	require.NoError(t, err)
	_, err = withoutOld.Decrypt("user-a", "rate-a", previous)
	assert.ErrorIs(t, err, ErrInvalidProtectedRateAmount)
	wrongMaterial, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(2)})
	require.NoError(t, err)
	_, err = wrongMaterial.Decrypt("user-a", "rate-a", previous)
	assert.ErrorIs(t, err, ErrInvalidProtectedRateAmount)
}

func TestRateCipherOwnsPreparedKeys(t *testing.T) {
	keys := map[string][]byte{"v1": testRateKey(1)}
	c, err := NewRateCipher("v1", keys)
	require.NoError(t, err)
	amount, err := c.Encrypt("user-a", "rate-a", 500)
	require.NoError(t, err)
	clear(keys["v1"])
	delete(keys, "v1")
	got, err := c.Decrypt("user-a", "rate-a", amount)
	require.NoError(t, err)
	assert.EqualValues(t, 500, got)
}
