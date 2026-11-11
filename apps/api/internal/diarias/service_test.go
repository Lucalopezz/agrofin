// Testa validação, criação protegida e soma de diárias por período.
package diarias

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dailyWorkRepositoryStub struct {
	check  func(context.Context, string, string) error
	create func(context.Context, ProtectedDailyWork) error
	list   func(context.Context, string, DailyWorkPeriod) ([]ProtectedDailyWorkRecord, error)
}

func (r dailyWorkRepositoryStub) CheckActiveEmployee(ctx context.Context, userID, employeeID string) error {
	return r.check(ctx, userID, employeeID)
}

func (r dailyWorkRepositoryStub) CreateDailyWork(ctx context.Context, work ProtectedDailyWork) error {
	return r.create(ctx, work)
}

func (r dailyWorkRepositoryStub) ListDailyWorks(ctx context.Context, userID string, period DailyWorkPeriod) ([]ProtectedDailyWorkRecord, error) {
	if r.list == nil {
		return nil, errors.New("listagem de diárias não configurada no stub")
	}
	return r.list(ctx, userID, period)
}

type dailyWorkProtectorFunc func(string, string, string, time.Time, int64) (ProtectedAmount, error)

func (f dailyWorkProtectorFunc) EncryptDailyWorkAmount(userID, dailyWorkID, employeeID string, workDate time.Time, cents int64) (ProtectedAmount, error) {
	return f(userID, dailyWorkID, employeeID, workDate, cents)
}

func (dailyWorkProtectorFunc) DecryptDailyWorkAmount(string, string, string, time.Time, ProtectedAmount) (int64, error) {
	return 0, ErrInvalidProtectedDailyWorkAmount
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
	var generatedID string
	repo := dailyWorkRepositoryStub{
		check: func(gotCtx context.Context, userID, employeeID string) error {
			calls = append(calls, "funcionário")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "user-a", userID)
			assert.Equal(t, validEmployeeID, employeeID)
			return nil
		},
		create: func(gotCtx context.Context, work ProtectedDailyWork) error {
			calls = append(calls, "gravação")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, generatedID, work.ID)
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
		generatedID = dailyWorkID
		assert.Equal(t, expectedDate, date)
		return cipher.EncryptDailyWorkAmount(userID, dailyWorkID, employeeID, date, cents)
	})
	got, err := NewService(repo, protector).Create(ctx, "user-a", CreateDailyWorkInput{
		EmployeeID: strings.ToUpper(validEmployeeID),
		WorkDate:   inputDate, AmountCents: 8000, Notes: " Trabalho no talhão 2 ",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"funcionário", "cifra", "gravação"}, calls)
	assert.NotEmpty(t, generatedID)
	assert.Equal(t, generatedID, got.ID)
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
		name             string
		checkErr         error
		cancelAfterCheck bool
		protectErr       error
		createErr        error
		expectedErr      error
		expected         []string
	}{
		{name: "falha ao validar funcionário", checkErr: failure, expectedErr: failure, expected: []string{"funcionário"}},
		{name: "funcionário inativo", checkErr: ErrEmployeeInactive, expectedErr: ErrEmployeeInactive, expected: []string{"funcionário"}},
		{name: "cancelamento após validar funcionário", cancelAfterCheck: true, expectedErr: context.Canceled, expected: []string{"funcionário"}},
		{name: "falha na cifra", protectErr: failure, expectedErr: failure, expected: []string{"funcionário", "cifra"}},
		{name: "falha na gravação", createErr: failure, expectedErr: failure, expected: []string{"funcionário", "cifra", "gravação"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := []string{}
			caseCtx, caseCancel := context.WithCancel(context.Background())
			defer caseCancel()
			repo := dailyWorkRepositoryStub{
				check: func(context.Context, string, string) error {
					calls = append(calls, "funcionário")
					if tt.cancelAfterCheck {
						caseCancel()
					}
					return tt.checkErr
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
			got, err := NewService(repo, protector).Create(caseCtx, "user-a", CreateDailyWorkInput{
				EmployeeID: validEmployeeID, WorkDate: date, AmountCents: 500,
			})
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, DailyWork{}, got)
			assert.Equal(t, tt.expected, calls)
		})
	}
}

func TestServiceSummarizeDailyWorks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	keyring, err := financialcipher.NewKeyring("test-v1", map[string][]byte{"test-v1": bytes.Repeat([]byte{5}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keyring)
	require.NoError(t, err)
	start := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 6)
	employeeA := validEmployeeID
	employeeB := "00000000-0000-4000-8000-000000000002"
	nameByID := map[string]string{employeeA: "João", employeeB: "Ana"}
	values := []struct {
		id         string
		employeeID string
		date       time.Time
		amount     int64
	}{
		{id: "daily-a1", employeeID: employeeA, date: start, amount: 8000},
		{id: "daily-a2", employeeID: employeeA, date: end, amount: 2000},
		{id: "daily-b1", employeeID: employeeB, date: start, amount: 500},
	}
	records := make([]ProtectedDailyWorkRecord, 0, len(values))
	for _, value := range values {
		amount, err := cipher.EncryptDailyWorkAmount("user-a", value.id, value.employeeID, value.date, value.amount)
		require.NoError(t, err)
		records = append(records, ProtectedDailyWorkRecord{
			Work: ProtectedDailyWork{
				ID: value.id, UserID: "user-a", EmployeeID: value.employeeID,
				WorkDate: value.date, Amount: amount,
			},
			EmployeeName: nameByID[value.employeeID],
		})
	}
	repo := dailyWorkRepositoryStub{
		list: func(gotCtx context.Context, userID string, period DailyWorkPeriod) ([]ProtectedDailyWorkRecord, error) {
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "user-a", userID)
			assert.Equal(t, DailyWorkPeriod{StartDate: start, EndDate: end}, period)
			return records, nil
		},
	}

	got, err := NewService(repo, cipher).Summarize(ctx, "user-a", DailyWorkPeriod{
		StartDate: start, EndDate: end,
	})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, DailyWorkSummary{EmployeeID: employeeB, EmployeeName: "Ana", DailyWorkAmountCents: 500}, got[0])
	assert.Equal(t, DailyWorkSummary{EmployeeID: employeeA, EmployeeName: "João", DailyWorkAmountCents: 10000}, got[1])

	// Um envelope adulterado invalida a consulta inteira, sem retornar a soma
	// parcial dos registros que foram processados antes dele.
	tampered := append([]byte(nil), records[0].Work.Amount.Ciphertext...)
	tampered[0] ^= 1
	records[0].Work.Amount.Ciphertext = tampered
	partial, err := NewService(repo, cipher).Summarize(ctx, "user-a", DailyWorkPeriod{StartDate: start, EndDate: end})
	assert.ErrorIs(t, err, ErrInvalidProtectedDailyWorkAmount)
	assert.Nil(t, partial)
}

func TestServiceSummarizeDailyWorksRejectsInvalidPeriodBeforeReading(t *testing.T) {
	end := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, 1)
	service := NewService(dailyWorkRepositoryStub{}, nil)
	_, err := service.Summarize(context.Background(), "user-a", DailyWorkPeriod{StartDate: start, EndDate: end})
	assert.ErrorIs(t, err, ErrInvalidDailyWorkPeriod)
}

func TestServiceSummarizeDailyWorksRejectsOverflow(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	keyring, err := financialcipher.NewKeyring("test-v1", map[string][]byte{"test-v1": bytes.Repeat([]byte{8}, 32)})
	require.NoError(t, err)
	cipher, err := NewCipher(keyring)
	require.NoError(t, err)
	values := []struct {
		id     string
		amount int64
	}{{id: "daily-overflow-a", amount: math.MaxInt64}, {id: "daily-overflow-b", amount: 1}}
	records := make([]ProtectedDailyWorkRecord, 0, len(values))
	for _, value := range values {
		protected, err := cipher.EncryptDailyWorkAmount("user-a", value.id, validEmployeeID, date, value.amount)
		require.NoError(t, err)
		records = append(records, ProtectedDailyWorkRecord{
			Work: ProtectedDailyWork{
				ID: value.id, EmployeeID: validEmployeeID, WorkDate: date, Amount: protected,
			},
			EmployeeName: "João",
		})
	}
	repo := dailyWorkRepositoryStub{
		list: func(context.Context, string, DailyWorkPeriod) ([]ProtectedDailyWorkRecord, error) {
			return records, nil
		},
	}

	_, err = NewService(repo, cipher).Summarize(ctx, "user-a", DailyWorkPeriod{StartDate: date, EndDate: date})
	assert.ErrorIs(t, err, ErrDailyWorkSummaryOverflow)
}
