package diarias

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
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

type ProtectedAmount = financialcipher.Amount[dailyWorkPurpose]

type ProtectedDailyWork struct {
	ID         string
	UserID     string
	EmployeeID string
	WorkDate   time.Time
	Amount     ProtectedAmount
	Notes      string
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

	if work.AmountCents <= 0 {
		return DailyWork{}, ErrInvalidDailyWorkAmount
	}

	// A operação é um dia no calendário, não um instante com horário.
	date := work.WorkDate
	work.WorkDate = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	work.Notes = strings.TrimSpace(work.Notes)

	return work, nil
}

func normalizeDailyWorkEmployeeID(value string) (string, error) {
	if len(value) != 36 ||
		value[8] != '-' || value[13] != '-' ||
		value[18] != '-' || value[23] != '-' ||
		strings.TrimSpace(value) != value {
		return "", ErrInvalidEmployeeID
	}

	decoded, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	if err != nil || len(decoded) != 16 {
		return "", ErrInvalidEmployeeID
	}

	return strings.ToLower(value), nil
}
