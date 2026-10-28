package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevelopmentAccountID(t *testing.T) {
	t.Run("sem configuração mantém cadastro desabilitado", func(t *testing.T) {
		id, err := developmentAccountID(context.Background(), nil, "", "")
		require.NoError(t, err)
		assert.Empty(t, id)
	})
	t.Run("conta de desenvolvimento não habilita escrita em produção", func(t *testing.T) {
		// O pool nil comprova que a configuração é rejeitada antes de consultar o banco.
		id, err := developmentAccountID(context.Background(), nil, "production", "account-id")
		require.Error(t, err)
		assert.Empty(t, id)
	})
}
