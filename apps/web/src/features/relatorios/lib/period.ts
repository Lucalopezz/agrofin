import { weekOf } from '@/features/colheita/lib/entries'
import { isPeriod, operationalDate, type Period } from '@/shared/lib/period'

export function reportWeek(now = new Date()): Period {
  return weekOf(operationalDate(now))
}

export function periodFromHash(hash: string): Period | undefined {
  const params = new URLSearchParams(hash.split('?')[1])
  const period = { start_date: params.get('start_date') ?? '', end_date: params.get('end_date') ?? '' }
  return isPeriod(period) ? period : undefined
}

export { periodLabel } from '@/shared/lib/period'
