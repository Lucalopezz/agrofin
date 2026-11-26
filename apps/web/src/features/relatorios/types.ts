import type { HarvestEntry, Period } from '@/features/colheita/types'

export type SaleType = 'exportacao' | 'mercado' | 'moagem'

export const saleLabels: Record<SaleType, string> = {
  exportacao: 'Exportação',
  mercado: 'Mercado',
  moagem: 'Moagem',
}

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
  sold: {
    id: string
    sold_at: string
    buyer_name?: string
    items: {
      sale_type: SaleType
      quantity: number
      unit_price_cents: number
      line_amount_cents: number
    }[]
    discount?: { reason: string; amount_cents: number }
    gross_amount_cents: number
    net_amount_cents: number
  }[]
  received: (Amount & { sale_id: string; sold_at: string; received_at: string })[]
  paid: (Amount & Employee & Period & { paid_at: string; notes?: string })[]
  expenses: (Amount & { paid_at: string; category: string; description?: string })[]
}

export type DetailResource = keyof ReportDetails
