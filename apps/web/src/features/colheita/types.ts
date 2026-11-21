export const boxTypes = ['normal', 'moagem'] as const
export type BoxType = typeof boxTypes[number]
export const boxLabels = { normal: 'Normal', moagem: 'Moagem' }

export type RateInput = {
  box_type: BoxType
  amount_cents: number
  valid_from: string
  valid_until: string | null
}

export type Rate = RateInput & { id: string }
