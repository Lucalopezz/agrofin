// Testa normalização e validação das entidades de diária.
package diarias

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validEmployeeID = "a1b2c3d4-1234-4567-89ab-0123456789ab"

func TestNormalizeDailyWork(t *testing.T) {
	inputDate := time.Date(2026, time.November, 4, 23, 15, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	work := DailyWork{
		UserID: "user-id", EmployeeID: strings.ToUpper(validEmployeeID),
		WorkDate: inputDate, AmountCents: 8000, Notes: "  Trabalho no talhão 2  ",
	}

	got, err := normalizeDailyWork(work)
	require.NoError(t, err)
	assert.Equal(t, validEmployeeID, got.EmployeeID)
	assert.Equal(t, time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC), got.WorkDate)
	assert.Equal(t, int64(8000), got.AmountCents)
	assert.Equal(t, "Trabalho no talhão 2", got.Notes)
	assert.Equal(t, inputDate, work.WorkDate, "a normalização não altera a entrada do chamador")
}

func TestNormalizeDailyWorkRejectsInvalidFields(t *testing.T) {
	validDate := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		work        DailyWork
		expectedErr error
	}{
		{name: "sem gerente", work: DailyWork{EmployeeID: validEmployeeID, WorkDate: validDate, AmountCents: 1}, expectedErr: ErrInvalidDailyWorkOwner},
		{name: "gerente com espaço", work: DailyWork{UserID: " user-id", EmployeeID: validEmployeeID, WorkDate: validDate, AmountCents: 1}, expectedErr: ErrInvalidDailyWorkOwner},
		{name: "uuid inválido", work: DailyWork{UserID: "user-id", EmployeeID: "employee", WorkDate: validDate, AmountCents: 1}, expectedErr: ErrInvalidEmployeeID},
		{name: "uuid com espaço", work: DailyWork{UserID: "user-id", EmployeeID: " " + validEmployeeID, WorkDate: validDate, AmountCents: 1}, expectedErr: ErrInvalidEmployeeID},
		{name: "data zero", work: DailyWork{UserID: "user-id", EmployeeID: validEmployeeID, AmountCents: 1}, expectedErr: ErrInvalidDailyWorkDate},
		{name: "valor zero", work: DailyWork{UserID: "user-id", EmployeeID: validEmployeeID, WorkDate: validDate}, expectedErr: ErrInvalidDailyWorkAmount},
		{name: "valor negativo", work: DailyWork{UserID: "user-id", EmployeeID: validEmployeeID, WorkDate: validDate, AmountCents: -1}, expectedErr: ErrInvalidDailyWorkAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeDailyWork(tt.work)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
