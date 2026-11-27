export type Period = { start_date: string; end_date: string }

const operationalDateFormatter = new Intl.DateTimeFormat('pt-BR', {
  timeZone: 'America/Sao_Paulo',
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
})

export function operationalDate(now = new Date()) {
  // Os períodos padrão seguem o mesmo dia operacional dos eventos de caixa.
  const parts = operationalDateFormatter.formatToParts(now)
  const day = (type: Intl.DateTimeFormatPartTypes) => parts.find((part) => part.type === type)?.value
  return `${day('year')}-${day('month')}-${day('day')}`
}

export function isCivilDate(value: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || value < '0001-01-01') return false
  const instant = new Date(`${value}T00:00:00Z`)
  return !Number.isNaN(instant.getTime()) && instant.toISOString().slice(0, 10) === value
}

export function isPeriod(period: Period) {
  return isCivilDate(period.start_date) && isCivilDate(period.end_date) &&
    period.start_date <= period.end_date
}

export function displayDate(date: string) {
  return date.split('-').reverse().join('/')
}

export function periodLabel(period: Period) {
  if (period.start_date === period.end_date) return displayDate(period.start_date)
  return `${displayDate(period.start_date)} a ${displayDate(period.end_date)}`
}
