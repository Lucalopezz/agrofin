import { apiRequest } from '@/shared/lib/api-client'
import type { Period } from '@/shared/lib/period'
import type { Expense, ExpenseInput } from '../types'

export function listExpenses({ start_date, end_date }: Period, signal: AbortSignal) {
  const params = new URLSearchParams({ start_date, end_date })
  return apiRequest<Expense[]>(`/expenses?${params}`, { signal })
}

export function createExpense(input: ExpenseInput) {
  return apiRequest<Expense>('/expenses', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateExpense(id: string, input: ExpenseInput) {
  return apiRequest<Expense>(`/expenses/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deleteExpense(id: string) {
  return apiRequest<void>(`/expenses/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
