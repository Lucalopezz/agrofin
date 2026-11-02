package colheitas

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateRepositoryIntegration(t *testing.T) {
	// A integração só usa um banco exclusivo de teste e já migrado. Nunca
	// recorremos a DATABASE_URL para não gravar dados na base de desenvolvimento.
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("configure TEST_DATABASE_URL com um banco de teste migrado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()
	userIDs := []string{}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		// IDs e logins únicos isolam inclusive execuções simultâneas do teste.
		// Só removemos as contas criadas aqui e suas tarifas, sem tocar nos demais testes.
		_, err := pool.Exec(cleanupCtx, "DELETE FROM harvest_rates WHERE user_id::text = ANY($1)", userIDs)
		assert.NoError(t, err)
		_, err = pool.Exec(cleanupCtx, "DELETE FROM users WHERE id::text = ANY($1)", userIDs)
		assert.NoError(t, err)
	}()
	for range 3 {
		var id string
		err = pool.QueryRow(ctx, `
			INSERT INTO users (name, login_name, password_hash)
			VALUES ('Gerente de teste', 'teste-tarifas-' || gen_random_uuid()::text, 'hash-de-teste')
			RETURNING id::text
		`).Scan(&id)
		require.NoError(t, err)
		userIDs = append(userIDs, id)
	}
	repo := NewRepository(pool)
	// Chaves determinísticas são apenas fixtures locais, não configuração do runtime.
	key := bytes.Repeat([]byte{1}, 32)
	protector, err := NewRateCipher("test-v1", map[string][]byte{"test-v1": key})
	require.NoError(t, err)
	service := NewService(repo, protector)
	start := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 8)
	before := start.AddDate(0, 0, -1)
	after := end.AddDate(0, 0, 2)
	create := func(userID string, boxType BoxType, cents int64, until *time.Time) Rate {
		t.Helper()
		rate, err := service.CreateRate(ctx, userID, CreateRateInput{BoxType: boxType, AmountCents: cents, ValidFrom: start, ValidUntil: until})
		require.NoError(t, err)
		return rate
	}
	load := func(rate Rate) ProtectedRate {
		t.Helper()
		var stored ProtectedRate
		var boxType string
		var general bool
		err := pool.QueryRow(ctx, `
			SELECT id::text, user_id::text, box_type, rate_ciphertext, rate_nonce,
			       rate_key_version, valid_from, valid_until, employee_id IS NULL
			FROM harvest_rates WHERE id = $1 AND user_id = $2
		`, rate.ID, rate.UserID).Scan(&stored.ID, &stored.UserID, &boxType,
			&stored.Amount.Ciphertext, &stored.Amount.Nonce, &stored.Amount.KeyVersion,
			&stored.ValidFrom, &stored.ValidUntil, &general)
		require.NoError(t, err)
		stored.BoxType = BoxType(boxType)
		assert.Equal(t, rate.ID, stored.ID)
		assert.Equal(t, rate.UserID, stored.UserID)
		assert.Equal(t, rate.BoxType, stored.BoxType)
		assert.Equal(t, rate.ValidFrom, stored.ValidFrom)
		assert.Equal(t, rate.ValidUntil, stored.ValidUntil)
		assert.True(t, general, "a persistência não pode criar tarifa individual")
		return stored
	}

	firstID, err := repo.NextRateID(ctx)
	require.NoError(t, err)
	secondID, err := repo.NextRateID(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, firstID, secondID)
	var canonical string
	err = pool.QueryRow(ctx, "SELECT $1::uuid::text", firstID).Scan(&canonical)
	require.NoError(t, err)
	assert.Equal(t, firstID, canonical)
	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM harvest_rates WHERE id::text = ANY($1)", []string{firstID, secondID}).Scan(&count)
	require.NoError(t, err)
	assert.Zero(t, count, "gerar IDs não pode inserir tarifas parciais")

	normal := create(userIDs[0], BoxTypeNormal, 500, &end)
	open := create(userIDs[0], BoxTypeMoagem, 200, nil)
	other := create(userIDs[1], BoxTypeNormal, 700, &end)
	stored := load(normal)
	assert.Nil(t, load(open).ValidUntil)
	load(other)
	assert.Len(t, stored.Amount.Ciphertext, 24)
	assert.Len(t, stored.Amount.Nonce, 12)
	var plaintext [8]byte
	binary.BigEndian.PutUint64(plaintext[:], 500)
	assert.False(t, bytes.Contains(stored.Amount.Ciphertext, plaintext[:]))
	// Recuperar o envelope do banco com uma nova instância simula a leitura de
	// dados antigos após trocar a versão ativa, mantendo a chave histórica fora da base.
	recovered, err := NewRateCipher("test-v2", map[string][]byte{"test-v1": key, "test-v2": bytes.Repeat([]byte{2}, 32)})
	require.NoError(t, err)
	cents, err := recovered.Decrypt(stored.UserID, stored.ID, stored.Amount)
	require.NoError(t, err)
	assert.EqualValues(t, 500, cents)
	wrongKey, err := NewRateCipher("test-v1", map[string][]byte{"test-v1": bytes.Repeat([]byte{3}, 32)})
	require.NoError(t, err)
	_, err = wrongKey.Decrypt(stored.UserID, stored.ID, stored.Amount)
	assert.ErrorIs(t, err, ErrInvalidProtectedRateAmount)
	_, err = recovered.Decrypt(other.UserID, stored.ID, stored.Amount)
	assert.ErrorIs(t, err, ErrInvalidProtectedRateAmount)

	t.Run("sobreposição respeita conta tipo e limites exclusivos", func(t *testing.T) {
		cases := []struct {
			name     string
			period   RatePeriod
			expected bool
		}{
			{name: "mesmo intervalo", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeNormal, ValidFrom: start, ValidUntil: &end}, expected: true},
			{name: "atravessa início", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeNormal, ValidFrom: before, ValidUntil: &end}, expected: true},
			{name: "atravessa fim", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeNormal, ValidFrom: start, ValidUntil: &after}, expected: true},
			{name: "sem fim engloba existente", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeNormal, ValidFrom: before}, expected: true},
			{name: "adjacente antes", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeNormal, ValidFrom: before, ValidUntil: &start}},
			{name: "adjacente depois", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeNormal, ValidFrom: end, ValidUntil: &after}},
			{name: "adjacente depois sem fim", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeNormal, ValidFrom: end}},
			{name: "vigência existente aberta", period: RatePeriod{UserID: userIDs[0], BoxType: BoxTypeMoagem, ValidFrom: after}, expected: true},
			{name: "tipo da outra conta não interfere", period: RatePeriod{UserID: userIDs[1], BoxType: BoxTypeMoagem, ValidFrom: start}},
			{name: "outra conta com dados próprios", period: RatePeriod{UserID: userIDs[1], BoxType: BoxTypeNormal, ValidFrom: start, ValidUntil: &end}, expected: true},
			{name: "conta sem tarifas", period: RatePeriod{UserID: userIDs[2], BoxType: BoxTypeNormal, ValidFrom: start}},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				got, err := repo.HasRateOverlap(ctx, tt.period)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			})
		}
	})

	_, err = service.CreateRate(ctx, normal.UserID, CreateRateInput{BoxType: BoxTypeNormal, AmountCents: 600, ValidFrom: start, ValidUntil: &end})
	assert.ErrorIs(t, err, ErrRateOverlap)
	// Um ID duplicado é outro erro, mesmo que o SQLSTATE também seja de integridade.
	duplicate := stored
	duplicate.ValidFrom, duplicate.ValidUntil = end, &after
	err = repo.CreateRate(ctx, duplicate)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrRateOverlap)
	var databaseError *pgconn.PgError
	require.ErrorAs(t, err, &databaseError)
	assert.Equal(t, "23505", databaseError.Code)
	_, err = service.CreateRate(ctx, normal.UserID, CreateRateInput{BoxType: BoxTypeNormal, AmountCents: 600, ValidFrom: end, ValidUntil: &after})
	require.NoError(t, err, "o fim exclusivo permite cadastrar o período adjacente")

	t.Run("banco decide disputa depois das duas consultas", func(t *testing.T) {
		concurrentCtx, concurrentCancel := context.WithTimeout(ctx, 5*time.Second)
		defer concurrentCancel()
		checked := make(chan struct{}, 2)
		release := make(chan struct{})
		concurrentRepo := rateRepositoryBarrier{Repository: repo, checked: checked, release: release}
		concurrentService := NewService(concurrentRepo, protector)
		results := make(chan error, 2)
		for _, amount := range []int64{500, 600} {
			go func(cents int64) {
				_, err := concurrentService.CreateRate(concurrentCtx, userIDs[2], CreateRateInput{BoxType: BoxTypeNormal, AmountCents: cents, ValidFrom: start})
				results <- err
			}(amount)
		}
		// A barreira garante que ambas viram ausência antes de inserir, sem sleeps
		// ou dependência da ordem em que o escalonador executa as goroutines.
		for range 2 {
			select {
			case <-checked:
			case <-concurrentCtx.Done():
				t.Fatal("as consultas concorrentes não chegaram à barreira")
			}
		}
		close(release)
		successes, conflicts := 0, 0
		for range 2 {
			select {
			case err := <-results:
				if err == nil {
					successes++
				} else if errors.Is(err, ErrRateOverlap) {
					conflicts++
				} else {
					t.Errorf("erro inesperado na disputa: %v", err)
				}
			case <-concurrentCtx.Done():
				t.Fatal("a disputa concorrente não terminou")
			}
		}
		assert.Equal(t, 1, successes)
		assert.Equal(t, 1, conflicts)
		err = pool.QueryRow(ctx, "SELECT count(*) FROM harvest_rates WHERE user_id = $1", userIDs[2]).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("cancelamento e prazo chegam ao driver", func(t *testing.T) {
		canceledCtx, canceled := context.WithCancel(ctx)
		canceled()
		expiredCtx, expired := context.WithDeadline(ctx, time.Unix(1, 0))
		defer expired()
		for _, tt := range []struct {
			ctx context.Context
			err error
		}{{canceledCtx, context.Canceled}, {expiredCtx, context.DeadlineExceeded}} {
			_, err := repo.HasRateOverlap(tt.ctx, RatePeriod{UserID: normal.UserID, BoxType: BoxTypeNormal, ValidFrom: start, ValidUntil: &end})
			assert.ErrorIs(t, err, tt.err)
			id, err := repo.NextRateID(tt.ctx)
			assert.ErrorIs(t, err, tt.err)
			assert.Empty(t, id)
			err = repo.CreateRate(tt.ctx, stored)
			assert.ErrorIs(t, err, tt.err)
		}
	})
}

// rateRepositoryBarrier só coordena o teste; as consultas e escritas continuam
// usando o Repository e as restrições reais do PostgreSQL.
type rateRepositoryBarrier struct {
	*Repository
	checked chan<- struct{}
	release <-chan struct{}
}

func (r rateRepositoryBarrier) HasRateOverlap(ctx context.Context, period RatePeriod) (bool, error) {
	overlaps, err := r.Repository.HasRateOverlap(ctx, period)
	if err != nil || overlaps {
		return overlaps, err
	}
	select {
	case r.checked <- struct{}{}:
	case <-ctx.Done():
		return false, ctx.Err()
	}
	select {
	case <-r.release:
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
