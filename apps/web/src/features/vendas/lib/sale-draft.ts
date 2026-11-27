import { validQuantity } from '@/features/colheita/lib/entries'
import { parseCents } from '@/shared/lib/money'
import { saleLabels, saleTypes, type SaleDiscount, type SaleItemInput } from '../types'

type LineDraft = { quantity: string; price: string }
export type SaleLines = Record<typeof saleTypes[number], LineDraft>

export function emptySaleLines(): SaleLines {
  return {
    exportacao: { quantity: '', price: '' },
    mercado: { quantity: '', price: '' },
    moagem: { quantity: '', price: '' },
  }
}

export function saleDraftTotals(lines: SaleLines, discountText: string, reason: string) {
  const items: SaleItemInput[] = []
  const errors: string[] = []
  let gross = 0n
  for (const type of saleTypes) {
    const line = lines[type]
    if (!line.quantity.trim() && !line.price.trim()) continue

    const quantity = Number(line.quantity)
    const price = parseCents(line.price)
    if (!/^\d+$/.test(line.quantity.trim()) || !validQuantity(quantity) || price === null) {
      errors.push(`${saleLabels[type]} exige quantidade inteira positiva e preço por caixa válido.`)
      continue
    }

    items.push({ sale_type: type, quantity, unit_price_cents: price })
    // BigInt mantém a prévia exata ao multiplicar quantidade por centavos.
    gross += BigInt(quantity) * BigInt(price)
  }
  if (items.length === 0 && errors.length === 0) errors.push('Informe as caixas e o preço de pelo menos uma classificação.')
  if (gross > (1n << 63n) - 1n) errors.push('O valor bruto excede o limite aceito pela API.')

  let discount: SaleDiscount | undefined
  if (discountText.trim()) {
    const cents = parseCents(discountText)
    if (cents === null) errors.push('Informe um desconto positivo com até duas casas decimais.')
    else if (BigInt(cents) > gross) errors.push('O desconto não pode exceder o valor bruto.')
    else if (!reason.trim()) errors.push('Informe o motivo do desconto.')
    else discount = { amount_cents: cents, reason: reason.trim() }
  }

  const discountCents = BigInt(discount?.amount_cents ?? 0)
  return { items, errors, discount, gross, discountCents, net: gross - discountCents }
}
