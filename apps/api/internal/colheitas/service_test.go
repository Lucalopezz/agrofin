package colheitas

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cada teste configura somente as etapas esperadas. Uma chamada extra a uma
// função nil falha e revela que o serviço avançou além de uma validação ou erro.
type rateRepositoryStub struct {
	overlap func(context.Context, RatePeriod) (bool, error)
	nextID  func(context.Context) (string, error)
	create  func(context.Context, ProtectedRate) error
}

func (r rateRepositoryStub) HasRateOverlap(ctx context.Context, period RatePeriod) (bool, error) {
	return r.overlap(ctx, period)
}

func (r rateRepositoryStub) NextRateID(ctx context.Context) (string, error) {
	return r.nextID(ctx)
}

func (r rateRepositoryStub) CreateRate(ctx context.Context, rate ProtectedRate) error {
	return r.create(ctx, rate)
}

type rateAmountProtectorFunc func(string, string, int64) (ProtectedRateAmount, error)

func (f rateAmountProtectorFunc) Encrypt(userID, rateID string, cents int64) (ProtectedRateAmount, error) {
	return f(userID, rateID, cents)
}

func TestServiceCreateRate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Date(2026, time.November, 2, 19, 30, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	end := start.AddDate(0, 0, 8)
	expectedStart := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	expectedEnd := expectedStart.AddDate(0, 0, 8)
	protector, err := NewRateCipher("v1", map[string][]byte{"v1": testRateKey(1)})
	require.NoError(t, err)
	calls := []string{}
	repo := rateRepositoryStub{
		overlap: func(gotCtx context.Context, period RatePeriod) (bool, error) {
			calls = append(calls, "consulta")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, RatePeriod{UserID: "user-id", BoxType: BoxTypeNormal, ValidFrom: expectedStart, ValidUntil: &expectedEnd}, period)
			return false, nil
		},
		nextID: func(gotCtx context.Context) (string, error) {
			calls = append(calls, "id")
			assert.Same(t, ctx, gotCtx)
			return "rate-id", nil
		},
		create: func(gotCtx context.Context, rate ProtectedRate) error {
			calls = append(calls, "gravação")
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "rate-id", rate.ID)
			assert.Equal(t, "user-id", rate.UserID)
			assert.Equal(t, BoxTypeNormal, rate.BoxType)
			assert.Equal(t, expectedStart, rate.ValidFrom)
			require.NotNil(t, rate.ValidUntil)
			assert.Equal(t, expectedEnd, *rate.ValidUntil)
			// Usar a cifra real prova que o valor entregue à persistência está
			// protegido e vinculado exatamente ao gerente e ID gerado.
			cents, err := protector.Decrypt(rate.UserID, rate.ID, rate.Amount)
			require.NoError(t, err)
			assert.EqualValues(t, 500, cents)
			return nil
		},
	}
	observedProtector := rateAmountProtectorFunc(func(userID, rateID string, cents int64) (ProtectedRateAmount, error) {
		calls = append(calls, "cifra")
		return protector.Encrypt(userID, rateID, cents)
	})
	got, err := NewService(repo, observedProtector).CreateRate(ctx, "user-id", CreateRateInput{
		BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start, ValidUntil: &end,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"consulta", "id", "cifra", "gravação"}, calls)
	assert.Equal(t, Rate{ID: "rate-id", UserID: "user-id", BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: expectedStart, ValidUntil: &expectedEnd}, got)
}

func TestServiceCreateRateRejectsInvalidInput(t *testing.T) {
	start := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	valid := CreateRateInput{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start}
	tests := []struct {
		name        string
		userID      string
		input       CreateRateInput
		expectedErr error
	}{
		{name: "sem gerente", input: valid, expectedErr: ErrInvalidRateOwner},
		{name: "gerente com espaços", userID: " user-id", input: valid, expectedErr: ErrInvalidRateOwner},
		{name: "tipo de venda", userID: "user-id", input: CreateRateInput{BoxType: "mercado", AmountCents: 500, ValidFrom: start}, expectedErr: ErrInvalidBoxType},
		{name: "valor zero", userID: "user-id", input: CreateRateInput{BoxType: BoxTypeNormal, ValidFrom: start}, expectedErr: ErrInvalidRateAmount},
		{name: "sem início", userID: "user-id", input: CreateRateInput{BoxType: BoxTypeNormal, AmountCents: 500}, expectedErr: ErrInvalidRatePeriod},
		{name: "fim no mesmo dia", userID: "user-id", input: CreateRateInput{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start, ValidUntil: &start}, expectedErr: ErrInvalidRatePeriod},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Dependências sem operações configuradas garantem que os erros
			// locais sejam detectados antes de consulta, cifra ou escrita.
			got, err := NewService(rateRepositoryStub{}, nil).CreateRate(context.Background(), tt.userID, tt.input)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, Rate{}, got)
		})
	}
}

func TestServiceCreateRateStopsOnFailure(t *testing.T) {
	failure := errors.New("falha da dependência")
	tests := []struct {
		name               string
		overlap            bool
		checkErr           error
		idErr              error
		protectErr         error
		createErr          error
		cancelAfterID      bool
		cancelAfterProtect bool
		expectedErr        error
		expectedCalls      []string
	}{
		{name: "falha na consulta", checkErr: failure, expectedErr: failure, expectedCalls: []string{"consulta"}},
		{name: "vigência sobreposta", overlap: true, expectedErr: ErrRateOverlap, expectedCalls: []string{"consulta"}},
		{name: "falha no id", idErr: failure, expectedErr: failure, expectedCalls: []string{"consulta", "id"}},
		{name: "falha na proteção", protectErr: failure, expectedErr: failure, expectedCalls: []string{"consulta", "id", "cifra"}},
		{name: "falha na gravação", createErr: failure, expectedErr: failure, expectedCalls: []string{"consulta", "id", "cifra", "gravação"}},
		// A restrição do banco será a última defesa quando outra requisição
		// gravar um intervalo conflitante depois da consulta do serviço.
		{name: "conflito concorrente traduzido", createErr: ErrRateOverlap, expectedErr: ErrRateOverlap, expectedCalls: []string{"consulta", "id", "cifra", "gravação"}},
		{name: "cancelamento antes da cifra", cancelAfterID: true, expectedErr: context.Canceled, expectedCalls: []string{"consulta", "id"}},
		{name: "cancelamento antes da escrita", cancelAfterProtect: true, expectedErr: context.Canceled, expectedCalls: []string{"consulta", "id", "cifra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := []string{}
			repo := rateRepositoryStub{
				overlap: func(gotCtx context.Context, period RatePeriod) (bool, error) {
					assert.Same(t, ctx, gotCtx)
					calls = append(calls, "consulta")
					return tt.overlap, tt.checkErr
				},
				nextID: func(gotCtx context.Context) (string, error) {
					assert.Same(t, ctx, gotCtx)
					calls = append(calls, "id")
					if tt.cancelAfterID {
						cancel()
					}
					return "rate-id", tt.idErr
				},
				create: func(gotCtx context.Context, rate ProtectedRate) error {
					assert.Same(t, ctx, gotCtx)
					calls = append(calls, "gravação")
					return tt.createErr
				},
			}
			protector := rateAmountProtectorFunc(func(string, string, int64) (ProtectedRateAmount, error) {
				calls = append(calls, "cifra")
				if tt.cancelAfterProtect {
					cancel()
				}
				return ProtectedRateAmount{}, tt.protectErr
			})
			got, err := NewService(repo, protector).CreateRate(ctx, "user-id", CreateRateInput{
				BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC),
			})
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, Rate{}, got)
			assert.Equal(t, tt.expectedCalls, calls)
		})
	}
}

func TestServiceCreateRateCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewService(rateRepositoryStub{}, nil).CreateRate(ctx, "user-id", CreateRateInput{})
	assert.ErrorIs(t, err, context.Canceled)
	ctx, cancel = context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer cancel()
	_, err = NewService(rateRepositoryStub{}, nil).CreateRate(ctx, "user-id", CreateRateInput{})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
