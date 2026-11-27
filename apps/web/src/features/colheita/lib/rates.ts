import type { BoxType, Rate } from '../types'

export function localDate(date = new Date()) {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

export function activeRate(rates: Rate[], type: BoxType, date: string) {
  // O início pertence à vigência e o fim é o primeiro dia sem essa tarifa.
  return rates.find(
    (rate) =>
      rate.box_type === type &&
      rate.valid_from <= date &&
      (!rate.valid_until || date < rate.valid_until),
  )
}

export { money, parseCents } from '@/shared/lib/money'
export { displayDate } from '@/shared/lib/period'
