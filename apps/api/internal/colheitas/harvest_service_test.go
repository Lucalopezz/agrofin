// Testa criação, consulta e cálculo dos lançamentos de colheita.
package colheitas

import (
	"context"
	"testing"
	"time"

	googleuuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type harvestRepositoryStub struct {
	check  func(context.Context, string, string) error
	rate   func(context.Context, string, BoxType, time.Time) (ProtectedRate, error)
	create func(context.Context, ProtectedHarvest) error
	list   func(context.Context, string, HarvestPeriod) ([]ProtectedHarvestRecord, error)
}

func (r harvestRepositoryStub) CheckActiveEmployee(ctx context.Context, userID, employeeID string) error {
	return r.check(ctx, userID, employeeID)
}

func (r harvestRepositoryStub) RateForDate(ctx context.Context, userID string, boxType BoxType, date time.Time) (ProtectedRate, error) {
	return r.rate(ctx, userID, boxType, date)
}

func (r harvestRepositoryStub) CreateHarvest(ctx context.Context, harvest ProtectedHarvest) error {
	return r.create(ctx, harvest)
}

func (r harvestRepositoryStub) ListHarvests(ctx context.Context, userID string, period HarvestPeriod) ([]ProtectedHarvestRecord, error) {
	if r.list == nil {
		return nil, nil
	}
	return r.list(ctx, userID, period)
}

func (r harvestRepositoryStub) GetHarvest(context.Context, string, string) (ProtectedHarvestRecord, error) {
	return ProtectedHarvestRecord{}, ErrHarvestNotFound
}

func (r harvestRepositoryStub) UpdateHarvest(context.Context, ProtectedHarvest, ProtectedHarvest, func() error) error {
	return nil
}

func TestHarvestServiceCreateGeneratesIDBeforeProtectingItems(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	date := time.Date(2026, time.November, 6, 0, 0, 0, 0, time.UTC)
	cipher, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(4)})
	require.NoError(t, err)
	rateAmount, err := cipher.Encrypt("user-a", "rate-a", 650)
	require.NoError(t, err)
	calls := []string{}
	var persistedID string

	repo := harvestRepositoryStub{
		check: func(gotCtx context.Context, userID, employeeID string) error {
			calls = append(calls, "funcionário")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "user-a", userID)
			assert.Equal(t, "00000000-0000-4000-8000-000000000001", employeeID)
			return nil
		},
		rate: func(gotCtx context.Context, userID string, boxType BoxType, gotDate time.Time) (ProtectedRate, error) {
			calls = append(calls, "tarifa")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "user-a", userID)
			assert.Equal(t, BoxTypeNormal, boxType)
			assert.Equal(t, date, gotDate)
			return ProtectedRate{ID: "rate-a", UserID: userID, BoxType: boxType, Amount: rateAmount}, nil
		},
		create: func(gotCtx context.Context, harvest ProtectedHarvest) error {
			calls = append(calls, "gravação")
			assert.Same(t, ctx, gotCtx)
			persistedID = harvest.ID
			require.Len(t, harvest.Items, 1)
			cents, err := cipher.DecryptHarvestItemRate(
				harvest.UserID, harvest.ID, harvest.EmployeeID, harvest.HarvestDate,
				harvest.Items[0].BoxType, harvest.Items[0].Quantity, harvest.Items[0].AppliedAmount,
			)
			require.NoError(t, err)
			assert.EqualValues(t, 650, cents)
			return nil
		},
	}

	got, err := NewHarvestService(repo, cipher).Create(ctx, "user-a", CreateHarvestInput{
		EmployeeID:  "00000000-0000-4000-8000-000000000001",
		HarvestDate: date,
		Items:       []HarvestItemInput{{BoxType: BoxTypeNormal, Quantity: 5}},
	})
	require.NoError(t, err)
	parsedID, err := googleuuid.Parse(persistedID)
	require.NoError(t, err)
	assert.Equal(t, googleuuid.Version(4), parsedID.Version())
	assert.Equal(t, persistedID, got.ID)
	assert.Equal(t, []string{"funcionário", "tarifa", "gravação"}, calls)
	require.Len(t, got.Items, 1)
	assert.EqualValues(t, 650, got.Items[0].AppliedRateCents)
}
