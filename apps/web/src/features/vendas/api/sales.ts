import { apiRequest } from '@/shared/lib/api-client'
import type { Period } from '@/shared/lib/period'
import type { Receipt, Sale, SaleInput } from '../types'

export function listSales({ start_date, end_date }: Period, signal: AbortSignal) {
  const dates = new URLSearchParams({ start_date, end_date })
  return apiRequest<Sale[]>(`/sales?${dates}`, { signal })
}

export function createSale(input: SaleInput) {
  return apiRequest<Sale>('/sales', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function createReceipt(saleId: string, cents: number) {
  return apiRequest<Receipt>(`/sales/${encodeURIComponent(saleId)}/receipts`, {
    method: 'POST',
    body: JSON.stringify({ amount_cents: cents }),
  })
}
