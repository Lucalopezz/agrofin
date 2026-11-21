import type { HarvestEntry, Period } from '../types'
import { localDate, money } from './rates'

export function weekOf(day = localDate()): Period {
  // O calendário usa dias locais, inclusive ao atravessar o fim do mês.
  const monday = new Date(day + 'T12:00:00')
  monday.setDate(monday.getDate() - ((monday.getDay() + 6) % 7))
  const sunday = new Date(monday)
  sunday.setDate(monday.getDate() + 6)
  return { start_date: localDate(monday), end_date: localDate(sunday) }
}

export function validQuantity(value: number) {
  return Number.isInteger(value) && value > 0 && value <= 2147483647
}

export function entryTotal(entry: HarvestEntry) {
  // A API retorna a tarifa fotografada por item. O histórico nunca usa a tarifa atual.
  let total = 0n
  for (const item of entry.items) {
    if (!Number.isSafeInteger(item.applied_rate_cents)) return 'Valor fora do limite de exibição'
    total += BigInt(item.quantity) * BigInt(item.applied_rate_cents)
  }
  return money(total)
}
