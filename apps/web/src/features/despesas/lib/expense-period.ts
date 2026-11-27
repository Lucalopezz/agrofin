import { operationalDate, type Period } from '@/shared/lib/period'

export function expenseMonth(day = operationalDate()): Period {
  const first = `${day.slice(0, 7)}-01`
  const end = new Date(`${first}T00:00:00Z`)
  // O dia zero do mês seguinte é o último dia do mês exibido.
  end.setUTCMonth(end.getUTCMonth() + 1, 0)
  return { start_date: first, end_date: end.toISOString().slice(0, 10) }
}
