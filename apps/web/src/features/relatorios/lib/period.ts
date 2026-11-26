import { weekOf } from '@/features/colheita/lib/entries'
import { displayDate } from '@/features/colheita/lib/rates'
import type { Period } from '@/features/colheita/types'

export function reportWeek(now = new Date()): Period {
  // O início da semana usa o mesmo dia operacional do relatório, mesmo em viagem.
  const parts = new Intl.DateTimeFormat('pt-BR', {
    timeZone: 'America/Sao_Paulo', day: '2-digit', month: '2-digit', year: 'numeric',
  }).formatToParts(now)
  const day = (type: Intl.DateTimeFormatPartTypes) => parts.find((part) => part.type === type)?.value
  return weekOf(`${day('year')}-${day('month')}-${day('day')}`)
}

function isCivilDate(value: string) {
  const instant = new Date(`${value}T00:00:00Z`)
  return /^\d{4}-\d{2}-\d{2}$/.test(value) &&
    value >= '0001-01-01' &&
    !Number.isNaN(instant.getTime()) &&
    instant.toISOString().slice(0, 10) === value
}

export function isReportPeriod(period: Period) {
  return isCivilDate(period.start_date) && isCivilDate(period.end_date) &&
    period.start_date <= period.end_date
}

export function periodFromHash(hash: string): Period | undefined {
  const params = new URLSearchParams(hash.split('?')[1])
  const period = { start_date: params.get('start_date') ?? '', end_date: params.get('end_date') ?? '' }
  return isReportPeriod(period) ? period : undefined
}

export function periodLabel(period: Period) {
  if (period.start_date === period.end_date) return displayDate(period.start_date)
  return `${displayDate(period.start_date)} a ${displayDate(period.end_date)}`
}
