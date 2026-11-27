import { apiRequest } from '@/shared/lib/api-client'
import type { Period } from '@/shared/lib/period'
import type {
  DailyWorkInput, PaymentInput, PayrollSummary, SavedDailyWork, SavedPayment,
} from '../types'

export function getPayrollSummary({ start_date, end_date }: Period, signal: AbortSignal) {
  const params = new URLSearchParams({ start_date, end_date })
  return apiRequest<PayrollSummary>(`/payroll-periods/summary?${params}`, { signal })
}

export function registerPayment(input: PaymentInput) {
  return apiRequest<SavedPayment>('/employee-payments', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function registerDailyWork(input: DailyWorkInput) {
  return apiRequest<SavedDailyWork>('/daily-work', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
