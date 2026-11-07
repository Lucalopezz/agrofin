package pagamentos

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/colheitas"
	"github.com/lucalopezz/agrofin/apps/api/internal/diarias"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type harvestSummarizerFunc func(context.Context, string, colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error)

func (f harvestSummarizerFunc) Summarize(ctx context.Context, userID string, period colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
	return f(ctx, userID, period)
}

type dailyWorkSummarizerFunc func(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error)

func (f dailyWorkSummarizerFunc) Summarize(ctx context.Context, userID string, period diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
	return f(ctx, userID, period)
}

func TestServiceSummarizeCombinesHarvestsAndDailyWorks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 6)
	employeeA := "employee-a"
	employeeB := "employee-b"
	employeeC := "employee-c"
	harvestCalls, dailyCalls := 0, 0
	service := NewService(
		harvestSummarizerFunc(func(gotCtx context.Context, userID string, period colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
			harvestCalls++
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "user-a", userID)
			assert.Equal(t, colheitas.HarvestPeriod{StartDate: start, EndDate: end}, period)
			return []colheitas.HarvestSummary{
				{
					EmployeeID: employeeA, EmployeeName: "João",
					NormalBoxes: 10, MillingBoxes: 4,
					NormalAmountCents: 5000, MillingAmountCents: 800, HarvestAmountCents: 5800,
				},
				{EmployeeID: employeeB, EmployeeName: "Bruna", HarvestAmountCents: 1200},
			}, nil
		}),
		dailyWorkSummarizerFunc(func(gotCtx context.Context, userID string, period diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
			dailyCalls++
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "user-a", userID)
			assert.Equal(t, diarias.DailyWorkPeriod{StartDate: start, EndDate: end}, period)
			return []diarias.DailyWorkSummary{
				{EmployeeID: employeeA, EmployeeName: "João", DailyWorkAmountCents: 8000},
				{EmployeeID: employeeC, EmployeeName: "Ana", DailyWorkAmountCents: 2500},
			}, nil
		}),
	)

	got, err := service.Summarize(ctx, "user-a", Period{StartDate: start, EndDate: end})
	require.NoError(t, err)
	assert.Equal(t, Period{StartDate: start, EndDate: end}, got.Period)
	assert.Equal(t, 1, harvestCalls)
	assert.Equal(t, 1, dailyCalls)
	require.Len(t, got.Employees, 3)
	assert.Equal(t, EmployeeSummary{
		EmployeeID: employeeC, EmployeeName: "Ana", DailyWorkAmountCents: 2500, DueAmountCents: 2500,
	}, got.Employees[0])
	assert.Equal(t, EmployeeSummary{
		EmployeeID: employeeB, EmployeeName: "Bruna", HarvestAmountCents: 1200, DueAmountCents: 1200,
	}, got.Employees[1])
	assert.Equal(t, EmployeeSummary{
		EmployeeID: employeeA, EmployeeName: "João", NormalBoxes: 10, MillingBoxes: 4,
		NormalAmountCents: 5000, MillingAmountCents: 800, HarvestAmountCents: 5800,
		DailyWorkAmountCents: 8000, DueAmountCents: 13800,
	}, got.Employees[2])
}

func TestServiceSummarizeRejectsInvalidPeriodBeforeDependencies(t *testing.T) {
	date := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	calls := 0
	service := NewService(
		harvestSummarizerFunc(func(context.Context, string, colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
			calls++
			return nil, nil
		}),
		dailyWorkSummarizerFunc(func(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
			calls++
			return nil, nil
		}),
	)
	_, err := service.Summarize(context.Background(), "user-a", Period{StartDate: date.AddDate(0, 0, 1), EndDate: date})
	assert.ErrorIs(t, err, ErrInvalidPayrollPeriod)
	assert.Zero(t, calls)
}

func TestServiceSummarizeEmptyPeriodReturnsInitializedList(t *testing.T) {
	date := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	service := NewService(
		harvestSummarizerFunc(func(context.Context, string, colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
			return []colheitas.HarvestSummary{}, nil
		}),
		dailyWorkSummarizerFunc(func(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
			return []diarias.DailyWorkSummary{}, nil
		}),
	)
	got, err := service.Summarize(context.Background(), "user-a", Period{StartDate: date, EndDate: date})
	require.NoError(t, err)
	assert.NotNil(t, got.Employees)
	assert.Empty(t, got.Employees)
}

func TestServiceSummarizePropagatesDependencyErrors(t *testing.T) {
	date := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	failure := errors.New("falha interna")
	t.Run("colheitas", func(t *testing.T) {
		service := NewService(
			harvestSummarizerFunc(func(context.Context, string, colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
				return nil, failure
			}),
			dailyWorkSummarizerFunc(func(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
				t.Fatal("diárias não devem ser consultadas depois da falha")
				return nil, nil
			}),
		)
		_, err := service.Summarize(context.Background(), "user-a", Period{StartDate: date, EndDate: date})
		assert.ErrorIs(t, err, failure)
	})
	t.Run("diárias", func(t *testing.T) {
		service := NewService(
			harvestSummarizerFunc(func(context.Context, string, colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
				return []colheitas.HarvestSummary{}, nil
			}),
			dailyWorkSummarizerFunc(func(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
				return nil, failure
			}),
		)
		_, err := service.Summarize(context.Background(), "user-a", Period{StartDate: date, EndDate: date})
		assert.ErrorIs(t, err, failure)
	})
}

func TestServiceSummarizeRejectsTotalOverflow(t *testing.T) {
	date := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	service := NewService(
		harvestSummarizerFunc(func(context.Context, string, colheitas.HarvestPeriod) ([]colheitas.HarvestSummary, error) {
			return []colheitas.HarvestSummary{{EmployeeID: "employee-a", HarvestAmountCents: math.MaxInt64}}, nil
		}),
		dailyWorkSummarizerFunc(func(context.Context, string, diarias.DailyWorkPeriod) ([]diarias.DailyWorkSummary, error) {
			return []diarias.DailyWorkSummary{{EmployeeID: "employee-a", DailyWorkAmountCents: 1}}, nil
		}),
	)
	_, err := service.Summarize(context.Background(), "user-a", Period{StartDate: date, EndDate: date})
	assert.ErrorIs(t, err, ErrPayrollSummaryOverflow)
}
