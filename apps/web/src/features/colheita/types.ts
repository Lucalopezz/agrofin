export const boxTypes = ['normal', 'moagem'] as const
export type BoxType = (typeof boxTypes)[number]
export const boxLabels = { normal: 'Normal', moagem: 'Moagem' }

export type RateInput = {
  box_type: BoxType
  amount_cents: number
  valid_from: string
  valid_until: string | null
}

export type Rate = RateInput & { id: string }

export type HarvestItemInput = { box_type: BoxType; quantity: number }
export type HarvestInput = {
  employee_id: string
  harvest_date: string
  items: HarvestItemInput[]
}
export type HarvestEntry = Omit<HarvestInput, 'items'> & {
  id: string
  items: (HarvestItemInput & { applied_rate_cents: number })[]
}
export type { Period } from '@/shared/lib/period'
