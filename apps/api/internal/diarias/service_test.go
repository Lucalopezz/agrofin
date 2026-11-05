package diarias

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dailyWorkRepositoryStub struct {
	check  func(context.Context, string, string) error
	nextID func(context.Context) (string, error)
	create func(context.Context, ProtectedDailyWork) error
}

func (r dailyWorkRepositoryStub) CheckActiveEmployee(ctx context.Context, userID, employeeID string) error {
	return r.check(ctx, userID, employeeID)
}

func (r dailyWorkRepositoryStub) NextDailyWorkID(ctx context.Context) (string, error) {
	return r.nextID(ctx)
}

func (r dailyWorkRepositoryStub) CreateDailyWork(ctx context.Context, work ProtectedDailyWork) error {
	return r.create(ctx, work)
}

type dailyWorkProtectorFunc func(string, string, string, time.Time, int64) (ProtectedAmount, error)

func (f dailyWorkProtectorFunc) EncryptDailyWorkAmount(userID, dailyWorkID, employeeID string, workDate time.Time, cents int64) (ProtectedAmount, error) {
	return f(userID, dailyWorkID, employeeID, workDate, cents)
}

func TestServiceCreateDailyWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inputDate := time.Date(2026, time.November, 4, 19, 30, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	expectedDate := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	keyring, err := financialcipher.NewKeyring("test-v1", map[string][]byte{"test-v1": bytes.Repeat([]byte{2}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keyring)
	require.NoError(t, err)
	calls := []string{}
	repo := dailyWorkRepositoryStub{
		check: func(gotCtx context.Context, userID, employeeID string) error {
			calls = append(calls, "funcionário")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "user-a", userID)
			assert.Equal(t, validEmployeeID, employeeID)
			return nil
		},
		nextID: func(gotCtx context.Context) (string, error) {
			calls = append(calls, "id")
			assert.Same(t, ctx, gotCtx)
			return "daily-a", nil
		},
		create: func(gotCtx context.Context, work ProtectedDailyWork) error {
			calls = append(calls, "gravação")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "daily-a", work.ID)
			assert.Equal(t, "user-a", work.UserID)
			assert.Equal(t, validEmployeeID, work.EmployeeID)
			assert.Equal(t, expectedDate, work.WorkDate)
			assert.Equal(t, "Trabalho no talhão 2", work.Notes)
			gotCents, err := cipher.DecryptDailyWorkAmount(work.UserID, work.ID, work.EmployeeID, work.WorkDate, work.Amount)
			require.NoError(t, err)
			assert.EqualValues(t, 8000, gotCents)
			return nil
		},
	}
	protector := dailyWorkProtectorFunc(func(userID, dailyWorkID, employeeID string, date time.Time, cents int64) (ProtectedAmount, error) {
		calls = append(calls, "cifra")
		assert.Equal(t, "daily-a", dailyWorkID)
		assert.Equal(t, expectedDate, date)
		return cipher.EncryptDailyWorkAmount(userID, dailyWorkID, employeeID, date, cents)
	})
	got, err := NewService(repo, protector).Create(ctx, "user-a", CreateDailyWorkInput{
		EmployeeID: strings.ToUpper(validEmployeeID),
		WorkDate:   inputDate, AmountCents: 8000, Notes: " Trabalho no talhão 2 ",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"funcionário", "id", "cifra", "gravação"}, calls)
	assert.Equal(t, "daily-a", got.ID)
	assert.Equal(t, "user-a", got.UserID)
	assert.Equal(t, validEmployeeID, got.EmployeeID)
	assert.Equal(t, expectedDate, got.WorkDate)
	assert.EqualValues(t, 8000, got.AmountCents)
}

func TestServiceCreateDailyWorkRejectsInvalidInputBeforeDependencies(t *testing.T) {
	date := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	valid := CreateDailyWorkInput{EmployeeID: validEmployeeID, WorkDate: date, AmountCents: 500}
	tests := []struct {
		name        string
		userID      string
		input       CreateDailyWorkInput
		expectedErr error
	}{
		{name: "sem gerente", input: valid, expectedErr: ErrInvalidDailyWorkOwner},
		{name: "uuid inválido", userID: "user-a", input: CreateDailyWorkInput{EmployeeID: "inválido", WorkDate: date, AmountCents: 500}, expectedErr: ErrInvalidEmployeeID},
		{name: "valor zero", userID: "user-a", input: CreateDailyWorkInput{EmployeeID: validEmployeeID, WorkDate: date}, expectedErr: ErrInvalidDailyWorkAmount},
		{name: "data vazia", userID: "user-a", input: CreateDailyWorkInput{EmployeeID: validEmployeeID, AmountCents: 500}, expectedErr: ErrInvalidDailyWorkDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(nil, nil)
			got, err := service.Create(context.Background(), tt.userID, tt.input)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, DailyWork{}, got)
		})
	}
}

func TestServiceCreateDailyWorkStopsOnDependencyFailure(t *testing.T) {
	failure := errors.New("falha da dependência")
	date := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name        string
		checkErr    error
		idErr       error
		protectErr  error
		createErr   error
		expectedErr error
		expected    []string
	}{
		{name: "falha ao validar funcionário", checkErr: failure, expectedErr: failure, expected: []string{"funcionário"}},
		{name: "funcionário inativo", checkErr: ErrEmployeeInactive, expectedErr: ErrEmployeeInactive, expected: []string{"funcionário"}},
		{name: "falha ao gerar id", idErr: failure, expectedErr: failure, expected: []string{"funcionário", "id"}},
		{name: "falha na cifra", protectErr: failure, expectedErr: failure, expected: []string{"funcionário", "id", "cifra"}},
		{name: "falha na gravação", createErr: failure, expectedErr: failure, expected: []string{"funcionário", "id", "cifra", "gravação"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := []string{}
			repo := dailyWorkRepositoryStub{
				check: func(context.Context, string, string) error {
					calls = append(calls, "funcionário")
					return tt.checkErr
				},
				nextID: func(context.Context) (string, error) {
					calls = append(calls, "id")
					return "daily-a", tt.idErr
				},
				create: func(context.Context, ProtectedDailyWork) error {
					calls = append(calls, "gravação")
					return tt.createErr
				},
			}
			protector := dailyWorkProtectorFunc(func(string, string, string, time.Time, int64) (ProtectedAmount, error) {
				calls = append(calls, "cifra")
				return ProtectedAmount{}, tt.protectErr
			})
			got, err := NewService(repo, protector).Create(context.Background(), "user-a", CreateDailyWorkInput{
				EmployeeID: validEmployeeID, WorkDate: date, AmountCents: 500,
			})
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, DailyWork{}, got)
			assert.Equal(t, tt.expected, calls)
		})
	}
}
