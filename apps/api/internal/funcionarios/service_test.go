// Testa regras de cadastro, renomeação, filtro e estado do funcionário.
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
	create     func(context.Context, Employee) (Employee, error)
	list       func(context.Context, string, *bool) ([]Employee, error)
	rename     func(context.Context, Employee) (Employee, error)
	deactivate func(context.Context, Employee) error
	reactivate func(context.Context, Employee) error
}

func (r employeeRepositoryStub) Create(ctx context.Context, employee Employee) (Employee, error) {
	return r.create(ctx, employee)
}

func (r employeeRepositoryStub) List(ctx context.Context, userID string, active *bool) ([]Employee, error) {
	return r.list(ctx, userID, active)
}

func (r employeeRepositoryStub) Rename(ctx context.Context, employee Employee) (Employee, error) {
	return r.rename(ctx, employee)
}

func (r employeeRepositoryStub) Deactivate(ctx context.Context, employee Employee) error {
	return r.deactivate(ctx, employee)
}

func (r employeeRepositoryStub) Reactivate(ctx context.Context, employee Employee) error {
	return r.reactivate(ctx, employee)
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

func TestServiceRename(t *testing.T) {
	t.Run("normaliza nome e preserva identificação contexto e estado retornado", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		expectedInput := Employee{ID: "employee-id", UserID: "user-id", Name: "João  D'Ávila"}
		// A renomeação também deve funcionar para um funcionário inativo.
		expected := expectedInput
		expected.Active = false
		called := false
		repo := employeeRepositoryStub{rename: func(gotCtx context.Context, employee Employee) (Employee, error) {
			called = true
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, expectedInput, employee)
			return expected, nil
		}}
		got, err := NewService(repo).Rename(ctx, "user-id", "employee-id", " \tJoão  D'Ávila\n")
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, expected, got)
	})

	for _, name := range []string{"", " \t\n"} {
		t.Run("nome inválido não chega à persistência "+name, func(t *testing.T) {
			repo := employeeRepositoryStub{rename: func(context.Context, Employee) (Employee, error) {
				// A regra precisa bloquear o UPDATE, não apenas devolver um erro depois dele.
				t.Fatal("persistência chamada com nome inválido")
				return Employee{}, nil
			}}
			got, err := NewService(repo).Rename(context.Background(), "user-id", "employee-id", name)
			assert.ErrorIs(t, err, ErrInvalidName)
			assert.Equal(t, Employee{}, got)
		})
	}

	for _, expectedErr := range []error{ErrNotFound, context.Canceled, context.DeadlineExceeded, errors.New("falha de persistência")} {
		t.Run("preserva erro "+expectedErr.Error(), func(t *testing.T) {
			repo := employeeRepositoryStub{rename: func(context.Context, Employee) (Employee, error) {
				return Employee{}, expectedErr
			}}
			_, err := NewService(repo).Rename(context.Background(), "user-id", "employee-id", "Ana")
			assert.ErrorIs(t, err, expectedErr)
		})
	}
}

func TestServiceDeactivate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := false
	repo := employeeRepositoryStub{deactivate: func(gotCtx context.Context, employee Employee) error {
		called = true
		assert.Same(t, ctx, gotCtx)
		// A entidade não pode perder o vínculo com o gerente nem transportar
		// um novo nome: a operação escolhe apenas o estado inativo.
		assert.Equal(t, Employee{ID: "employee-id", UserID: "user-id", Active: false}, employee)
		return nil
	}}
	require.NoError(t, NewService(repo).Deactivate(ctx, "user-id", "employee-id"))
	assert.True(t, called)

	for _, expectedErr := range []error{ErrNotFound, context.Canceled, context.DeadlineExceeded, errors.New("falha de persistência")} {
		t.Run("preserva erro "+expectedErr.Error(), func(t *testing.T) {
			repo := employeeRepositoryStub{deactivate: func(context.Context, Employee) error {
				return expectedErr
			}}
			err := NewService(repo).Deactivate(ctx, "user-id", "employee-id")
			assert.ErrorIs(t, err, expectedErr)
		})
	}
}

func TestServiceReactivate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := false
	repo := employeeRepositoryStub{reactivate: func(gotCtx context.Context, employee Employee) error {
		called = true
		assert.Same(t, ctx, gotCtx)
		// A reativação mantém o vínculo e escolhe true explicitamente, sem
		// transportar um nome novo ou depender do valor zero do estado.
		assert.Equal(t, Employee{ID: "employee-id", UserID: "user-id", Active: true}, employee)
		return nil
	}}
	require.NoError(t, NewService(repo).Reactivate(ctx, "user-id", "employee-id"))
	assert.True(t, called)

	for _, expectedErr := range []error{ErrNotFound, context.Canceled, context.DeadlineExceeded, errors.New("falha de persistência")} {
		t.Run("preserva erro "+expectedErr.Error(), func(t *testing.T) {
			repo := employeeRepositoryStub{reactivate: func(context.Context, Employee) error {
				return expectedErr
			}}
			err := NewService(repo).Reactivate(ctx, "user-id", "employee-id")
			assert.ErrorIs(t, err, expectedErr)
		})
	}
}
