import type { Period } from '@/shared/lib/period'

export type EmployeePayroll = {
  employee: { id: string; name: string }
  normal_boxes: number
  milling_boxes: number
  normal_amount_cents: number
  milling_amount_cents: number
  harvest_amount_cents: number
  daily_work_amount_cents: number
  due_amount_cents: number
  paid_amount_cents: number
  remaining_amount_cents: number
}

export type PayrollSummary = { period: Period; employees: EmployeePayroll[] }

type PaymentFields = {
  employee_id: string
  amount_cents: number
  notes?: string
}

export type PaymentInput = PaymentFields & Period
export type SavedPayment = PaymentInput & { id: string; paid_at: string }
export type DailyWorkInput = PaymentFields & { work_date: string }
export type SavedDailyWork = DailyWorkInput & { id: string }
