import type { HarvestEntry, Period } from '@/features/colheita/types'

import type { Sale, SaleType } from '@/features/vendas/types'

export { saleLabels, type SaleType } from '@/features/vendas/types'

type Employee = { employee_id: string; employee_name: string }

export type PeriodSummary = Period & {
  time_zone: string
  harvest: {
    total_boxes: number
    normal_boxes: number
    milling_boxes: number
    by_employee: (Employee & { normal_boxes: number; milling_boxes: number })[]
  }
  sales: {
    total_boxes: number
    boxes_by_type: Record<SaleType, number>
    gross_amount_cents: number
    discount_amount_cents: number
    net_amount_cents: number
    average_net_price_cents: number | null
  }
  payroll: {
    due_amount_cents: number
    paid_amount_cents: number
    by_employee: (Employee & {
      harvest_amount_cents: number
      daily_work_amount_cents: number
      due_amount_cents: number
    })[]
  }
  cash: {
    received_amount_cents: number
    paid_expenses_amount_cents: number
    paid_expenses_by_category: Record<string, number>
    balance_amount_cents: number
  }
}

type Amount = { id: string; amount_cents: number }

export type ReportDetails = {
  harvest: HarvestEntry[]
  dailyWork: (Amount & Employee & { work_date: string; notes?: string })[]
  sold: Pick<Sale, 'id' | 'sold_at' | 'buyer_name' | 'items' | 'discount' | 'gross_amount_cents' | 'net_amount_cents'>[]
  received: (Amount & { sale_id: string; sold_at: string; received_at: string })[]
  paid: (Amount & Employee & Period & { paid_at: string; notes?: string })[]
  expenses: (Amount & { paid_at: string; category: string; description?: string })[]
}

export type DetailResource = keyof ReportDetails
