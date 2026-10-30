package funcionarios

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A função substitui apenas a persistência para observar se a validação
// bloqueia a gravação e quais dados chegam ao repositório.
type employeeCreatorFunc func(context.Context, Employee) (Employee, error)

func (f employeeCreatorFunc) Create(ctx context.Context, employee Employee) (Employee, error) {
	return f(ctx, employee)
}

func TestServiceCreate(t *testing.T) {
	t.Run("monta entidade validada e preserva contexto", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		expectedInput := Employee{UserID: "user-id", Name: "João D'Ávila", Active: true}
		expected := expectedInput
		expected.ID = "employee-id"
		called := false
		repo := employeeCreatorFunc(func(gotCtx context.Context, employee Employee) (Employee, error) {
			called = true
			// Same verifica a identidade do contexto, não apenas seus campos:
			// criar outro contexto no serviço quebraria a propagação esperada.
			assert.Same(t, ctx, gotCtx)
			// O serviço entrega a entidade completa para criação, ainda sem ID.
			// O resultado da persistência inclui o identificador gerado pelo banco.
			assert.Equal(t, expectedInput, employee)
			return expected, nil
		})

		got, err := NewService(repo).Create(ctx, "user-id", "  João D'Ávila \n")
		require.NoError(t, err)
		assert.Equal(t, expected, got)
		assert.True(t, called, "a persistência deve ser chamada para um nome válido")
	})

	t.Run("nome vazio não chega à persistência", func(t *testing.T) {
		repo := employeeCreatorFunc(func(context.Context, Employee) (Employee, error) {
			t.Fatal("persistência chamada com nome inválido")
			return Employee{}, nil
		})
		got, err := NewService(repo).Create(context.Background(), "user-id", " \t\n")
		assert.ErrorIs(t, err, ErrInvalidName)
		assert.Equal(t, Employee{}, got)
	})

	t.Run("preserva erro da persistência", func(t *testing.T) {
		expectedErr := errors.New("falha de persistência")
		repo := employeeCreatorFunc(func(context.Context, Employee) (Employee, error) {
			return Employee{}, expectedErr
		})
		_, err := NewService(repo).Create(context.Background(), "user-id", "Ana")
		assert.ErrorIs(t, err, expectedErr)
	})
}
