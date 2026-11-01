package funcionarios

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cada teste fornece somente a operação esperada; chamar outra operação
// deixa a função nil falhar, revelando um desvio no fluxo do serviço.
type employeeRepositoryStub struct {
	create func(context.Context, Employee) (Employee, error)
	list   func(context.Context, string) ([]Employee, error)
}

func (r employeeRepositoryStub) Create(ctx context.Context, employee Employee) (Employee, error) {
	return r.create(ctx, employee)
}

func (r employeeRepositoryStub) List(ctx context.Context, userID string) ([]Employee, error) {
	return r.list(ctx, userID)
}

func TestServiceCreate(t *testing.T) {
	t.Run("monta entidade validada e preserva contexto", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		expectedInput := Employee{UserID: "user-id", Name: "João D'Ávila", Active: true}
		expected := expectedInput
		expected.ID = "employee-id"
		called := false
		repo := employeeRepositoryStub{create: func(gotCtx context.Context, employee Employee) (Employee, error) {
			called = true
			// Same verifica a identidade do contexto, não apenas seus campos:
			// criar outro contexto no serviço quebraria a propagação esperada.
			assert.Same(t, ctx, gotCtx)
			// O serviço entrega a entidade completa para criação, ainda sem ID.
			// O resultado da persistência inclui o identificador gerado pelo banco.
			assert.Equal(t, expectedInput, employee)
			return expected, nil
		}}

		got, err := NewService(repo).Create(ctx, "user-id", "  João D'Ávila \n")
		require.NoError(t, err)
		assert.Equal(t, expected, got)
		assert.True(t, called, "a persistência deve ser chamada para um nome válido")
	})

	t.Run("nome vazio não chega à persistência", func(t *testing.T) {
		repo := employeeRepositoryStub{create: func(context.Context, Employee) (Employee, error) {
			t.Fatal("persistência chamada com nome inválido")
			return Employee{}, nil
		}}
		got, err := NewService(repo).Create(context.Background(), "user-id", " \t\n")
		assert.ErrorIs(t, err, ErrInvalidName)
		assert.Equal(t, Employee{}, got)
	})

	t.Run("preserva erro da persistência", func(t *testing.T) {
		expectedErr := errors.New("falha de persistência")
		repo := employeeRepositoryStub{create: func(context.Context, Employee) (Employee, error) {
			return Employee{}, expectedErr
		}}
		_, err := NewService(repo).Create(context.Background(), "user-id", "Ana")
		assert.ErrorIs(t, err, expectedErr)
	})
}
