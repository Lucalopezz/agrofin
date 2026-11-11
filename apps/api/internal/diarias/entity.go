// Define diárias, períodos de consulta e regras de normalização.
package diarias

import (
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/money"
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

func normalizeDailyWorkPeriod(period DailyWorkPeriod) (DailyWorkPeriod, error) {
	start, end, err := calendarDate.NormalizeInclusivePeriod(period.StartDate, period.EndDate)
	if err != nil {
		return DailyWorkPeriod{}, ErrInvalidDailyWorkPeriod
	}
	period.StartDate, period.EndDate = start, end
	return period, nil
}

func normalizeDailyWork(work DailyWork) (DailyWork, error) {
	if work.UserID == "" || strings.TrimSpace(work.UserID) != work.UserID {
		return DailyWork{}, ErrInvalidDailyWorkOwner
	}

	employeeID, err := normalizeDailyWorkEmployeeID(work.EmployeeID)
	if err != nil {
		return DailyWork{}, err
	}
	work.EmployeeID = employeeID

	if work.WorkDate.IsZero() {
		return DailyWork{}, ErrInvalidDailyWorkDate
	}

	if !money.IsPositiveCents(work.AmountCents) {
		return DailyWork{}, ErrInvalidDailyWorkAmount
	}

	// A operação é um dia no calendário, não um instante com horário.
	work.WorkDate = calendarDate.Normalize(work.WorkDate)
	work.Notes = strings.TrimSpace(work.Notes)

	return work, nil
}

func normalizeDailyWorkEmployeeID(value string) (string, error) {
	id, err := uuid.Normalize(value)
	if err != nil {
		return "", ErrInvalidEmployeeID
	}
	return id, nil
}
