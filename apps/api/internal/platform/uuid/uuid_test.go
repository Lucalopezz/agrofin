package uuid

import (
	"testing"

	googleuuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReturnsUniqueValidUUID(t *testing.T) {
	first, err := New()
	require.NoError(t, err)
	second, err := New()
	require.NoError(t, err)

	_, err = googleuuid.Parse(first)
	require.NoError(t, err)
	_, err = googleuuid.Parse(second)
	require.NoError(t, err)
	assert.NotEqual(t, first, second)
}
