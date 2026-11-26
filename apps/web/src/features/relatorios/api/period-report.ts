import type { Period } from '@/features/colheita/types'
import { apiRequest } from '@/shared/lib/api-client'
import type { DetailResource, PeriodSummary, ReportDetails } from '../types'

const detailPaths: Record<DetailResource, string> = {
  harvest: '/harvest-entries',
  dailyWork: '/daily-work',
  sold: '/reports/sold-sales',
  received: '/sale-receipts',
  paid: '/employee-payments',
  expenses: '/expenses',
}

export function getPeriodSummary(period: Period, signal: AbortSignal) {
  return apiRequest<PeriodSummary>(`/reports/period-summary?${periodSearch(period)}`, { signal })
}

export function getPeriodDetails<R extends DetailResource>(resource: R, period: Period, signal: AbortSignal) {
  return apiRequest<ReportDetails[R]>(`${detailPaths[resource]}?${periodSearch(period)}`, { signal })
}


function periodSearch({ start_date, end_date }: Period) {
  return new URLSearchParams({ start_date, end_date })
}
