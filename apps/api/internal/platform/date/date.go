// Package date normaliza e interpreta datas de calendário sem horário.
package date

import (
	"errors"
	"strings"
	"time"
)

// Layout é o formato de data usado pelos contratos HTTP do projeto.
const Layout = "2006-01-02"

// ErrInvalid indica que o texto não representa uma data civil válida no formato aceito.
var (
	ErrInvalid = errors.New("date: invalid calendar date")
	// ErrInvalidPeriod indica limites ausentes ou invertidos num intervalo inclusivo.
	ErrInvalidPeriod = errors.New("date: invalid inclusive period")
)

// Parse aceita somente uma data completa no formato AAAA-MM-DD.
// Retornar meia-noite UTC mantém o dia civil estável ao persistir em colunas DATE.
func Parse(value string) (time.Time, error) {
	if len(value) != len(Layout) || strings.TrimSpace(value) != value {
		return time.Time{}, ErrInvalid
	}

	parsed, err := time.Parse(Layout, value)
	if err != nil || parsed.Year() < 1 || parsed.Format(Layout) != value {
		return time.Time{}, ErrInvalid
	}
	return Normalize(parsed), nil
}

// Normalize remove horário e fuso sem deslocar o dia que o chamador informou.
// A data zero e anos fora do calendário civil ficam no zero para que o domínio
// possa rejeitá-los junto das demais regras do período.
func Normalize(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	year, month, day := value.Date()
	if year < 1 {
		return time.Time{}
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// NormalizeInclusivePeriod normaliza os limites de um intervalo inclusivo de dias civis.
// O mesmo dia é um intervalo válido; períodos com fim anterior ao início não são.
func NormalizeInclusivePeriod(start, end time.Time) (time.Time, time.Time, error) {
	start = Normalize(start)
	end = Normalize(end)
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return time.Time{}, time.Time{}, ErrInvalidPeriod
	}
	return start, end, nil
}
