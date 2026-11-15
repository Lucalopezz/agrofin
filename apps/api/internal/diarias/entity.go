// Define diárias, períodos de consulta e regras de normalização.
package diarias

import (
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type DailyWork struct {
	ID          string
	UserID      string
	EmployeeID  string
	WorkDate    time.Time
	AmountCents int64
	Notes       string
}

type CreateDailyWorkInput struct {
	EmployeeID  string
	WorkDate    time.Time
	AmountCents int64
	Notes       string
}

// DailyWorkPeriod delimita uma consulta por datas de calendário, incluindo ambos os limites.
type DailyWorkPeriod struct {
	StartDate time.Time
	EndDate   time.Time
}

type ProtectedAmount = financialcipher.Amount[dailyWorkPurpose]

type ProtectedDailyWork struct {
	ID         string
	UserID     string
	EmployeeID string
	WorkDate   time.Time
	Amount     ProtectedAmount
	Notes      string
}

// ProtectedDailyWorkRecord inclui o nome atual do funcionário para projeções de leitura.
type ProtectedDailyWorkRecord struct {
	Work         ProtectedDailyWork
	EmployeeName string
}

// DailyWorkSummary agrega somente os valores de diárias de uma pessoa no período.
type DailyWorkSummary struct {
	EmployeeID           string
	EmployeeName         string
	DailyWorkAmountCents int64
}

func normalizeDailyWorkEmployeeID(value string) (string, error) {
	id, err := uuid.Normalize(value)
	if err != nil {
		return "", ErrInvalidEmployeeID
	}
	return id, nil
}
