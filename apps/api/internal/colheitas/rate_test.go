// Testa validação e normalização das tarifas de colheita.
package colheitas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRate(t *testing.T) {
	start := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 8)
	previousDay := start.AddDate(0, 0, -1)
	sameDayLater := start.Add(23 * time.Hour)
	zeroDate := time.Time{}
	zeroDayLater := zeroDate.Add(time.Hour)
	tests := []struct {
		name        string
		input       Rate
		expectedErr error
	}{
		{name: "normal sem fim", input: Rate{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start}},
		{name: "moagem com fim", input: Rate{BoxType: BoxTypeMoagem, AmountCents: 200, ValidFrom: start, ValidUntil: &end}},
		{name: "um centavo", input: Rate{BoxType: BoxTypeNormal, AmountCents: 1, ValidFrom: start}},
		{name: "tipo ausente", input: Rate{AmountCents: 500, ValidFrom: start}, expectedErr: ErrInvalidBoxType},
		// Os tipos da venda ao barracão não podem ser usados no pagamento da colheita.
		{name: "exportação não é tipo de colheita", input: Rate{BoxType: "exportacao", AmountCents: 500, ValidFrom: start}, expectedErr: ErrInvalidBoxType},
		{name: "mercado não é tipo de colheita", input: Rate{BoxType: "mercado", AmountCents: 500, ValidFrom: start}, expectedErr: ErrInvalidBoxType},
		{name: "tipo desconhecido", input: Rate{BoxType: "outro", AmountCents: 500, ValidFrom: start}, expectedErr: ErrInvalidBoxType},
		{name: "valor zero", input: Rate{BoxType: BoxTypeNormal, ValidFrom: start}, expectedErr: ErrInvalidRateAmount},
		{name: "valor negativo", input: Rate{BoxType: BoxTypeNormal, AmountCents: -1, ValidFrom: start}, expectedErr: ErrInvalidRateAmount},
		{name: "início ausente", input: Rate{BoxType: BoxTypeNormal, AmountCents: 500}, expectedErr: ErrInvalidRatePeriod},
		{name: "início zero após remover horário", input: Rate{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: zeroDayLater}, expectedErr: ErrInvalidRatePeriod},
		{name: "fim zero não equivale a ausência", input: Rate{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start, ValidUntil: &zeroDate}, expectedErr: ErrInvalidRatePeriod},
		{name: "fim anterior", input: Rate{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start, ValidUntil: &previousDay}, expectedErr: ErrInvalidRatePeriod},
		{name: "fim igual ao início", input: Rate{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start, ValidUntil: &start}, expectedErr: ErrInvalidRatePeriod},
		// Horários distintos no mesmo dia não podem criar uma vigência de um dia.
		{name: "mesmo dia com horário posterior", input: Rate{BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start, ValidUntil: &sameDayLater}, expectedErr: ErrInvalidRatePeriod},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// IDs são vínculos técnicos vindos do chamador confiável; a validação
			// local da tarifa não deve alterá-los ou depender de persistência.
			tt.input.ID = "rate-id"
			tt.input.UserID = "user-id"
			got, err := normalizeRate(tt.input)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				assert.Equal(t, Rate{}, got, "não devolver entidade parcialmente validada")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.input, got)
		})
	}
}

func TestNormalizeRatePreservesCalendarDays(t *testing.T) {
	// Em instantes absolutos, este fim vem antes do início. Como o domínio usa
	// dias, 03/11 deve ser aceito depois de 02/11, sem conversão entre fusos.
	start := time.Date(2026, time.November, 2, 23, 30, 15, 1, time.FixedZone("UTC-3", -3*60*60))
	end := time.Date(2026, time.November, 3, 0, 30, 15, 1, time.FixedZone("UTC+9", 9*60*60))
	input := Rate{ID: "rate-id", UserID: "user-id", BoxType: BoxTypeNormal, AmountCents: 500, ValidFrom: start, ValidUntil: &end}
	expectedStart := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	expectedEnd := expectedStart.AddDate(0, 0, 1)

	got, err := normalizeRate(input)
	require.NoError(t, err)
	assert.Equal(t, expectedStart, got.ValidFrom)
	require.NotNil(t, got.ValidUntil)
	assert.Equal(t, expectedEnd, *got.ValidUntil)
	assert.Equal(t, start, input.ValidFrom, "normalizar não altera a entrada")
	assert.Equal(t, end, *input.ValidUntil, "normalizar não altera a data apontada pelo chamador")

	// Mesmo mudando o fim original para um dia inválido, a entidade já validada
	// mantém sua própria cópia e continua com o intervalo que foi conferido.
	end = start.AddDate(0, 0, -1)
	assert.Equal(t, expectedEnd, *got.ValidUntil)
}
