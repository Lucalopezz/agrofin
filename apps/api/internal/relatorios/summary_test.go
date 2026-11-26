// Testa o arredondamento da média e a janela de datas do caixa.
package relatorios

import (
	"testing"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAveragePrice(t *testing.T) {
	assert.Nil(t, averagePrice(0, 0))
	for _, check := range []struct {
		net, boxes, want int64
	}{
		{0, 2, 0}, {401, 4, 100}, {402, 4, 101}, {302, 3, 101},
	} {
		got := averagePrice(check.net, check.boxes)
		require.NotNil(t, got)
		assert.Equal(t, check.want, *got)
	}
}

func TestCashWindowUsesLocalMidnight(t *testing.T) {
	location, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	day, err := date.Parse("2026-11-29")
	require.NoError(t, err)
	start, end := (Period{StartDate: day, EndDate: day}).cashWindow(location)
	assert.Equal(t, "2026-11-29T03:00:00Z", start.Format(time.RFC3339))
	assert.Equal(t, "2026-11-30T03:00:00Z", end.Format(time.RFC3339))
}

func TestCashWindowOnSkippedMidnight(t *testing.T) {
	location, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	day, err := date.Parse("2018-11-04")
	require.NoError(t, err)
	start, end := (Period{StartDate: day, EndDate: day}).cashWindow(location)
	assert.Equal(t, "2018-11-04T03:00:00Z", start.Format(time.RFC3339))
	assert.Equal(t, 23*time.Hour, end.Sub(start))
}
