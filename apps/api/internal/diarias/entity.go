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
	period.StartDate = dailyWorkDate(period.StartDate)
	period.EndDate = dailyWorkDate(period.EndDate)
	if period.StartDate.IsZero() || period.EndDate.IsZero() || period.EndDate.Before(period.StartDate) {
		return DailyWorkPeriod{}, ErrInvalidDailyWorkPeriod
	}
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

	if work.AmountCents <= 0 {
		return DailyWork{}, ErrInvalidDailyWorkAmount
	}

	// A operação é um dia no calendário, não um instante com horário.
	work.WorkDate = dailyWorkDate(work.WorkDate)
	work.Notes = strings.TrimSpace(work.Notes)

	return work, nil
}

func dailyWorkDate(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	// DATE não carrega fuso nem horário; reconstruir meia-noite UTC mantém o
	// mesmo contexto autenticado quando o lançamento sai e volta do PostgreSQL.
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
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
