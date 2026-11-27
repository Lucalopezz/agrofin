export const saleTypes = ['exportacao', 'mercado', 'moagem'] as const
export type SaleType = typeof saleTypes[number]

export const saleLabels: Record<SaleType, string> = {
  exportacao: 'Exportação',
  mercado: 'Mercado',
  moagem: 'Moagem',
}

export type SaleItemInput = { sale_type: SaleType; quantity: number; unit_price_cents: number }
export type SaleDiscount = { reason: string; amount_cents: number }

export type SaleInput = {
  sold_at: string
  buyer_name?: string
  notes?: string
  items: SaleItemInput[]
  discount?: SaleDiscount
}

export type Sale = Omit<SaleInput, 'items'> & {
  id: string
  items: (SaleItemInput & { line_amount_cents: number })[]
  gross_amount_cents: number
  net_amount_cents: number
  received_amount_cents: number
  receivable_amount_cents: number
}

export type Receipt = { id: string; sale_id: string; received_at: string; amount_cents: number }
