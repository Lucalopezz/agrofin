import { describe, expect, it } from 'vitest'
import { activeRate, money, parseCents } from './rates'
import type { Rate } from '../types'

const rate: Rate = {
  id: 'a', box_type: 'normal', amount_cents: 501, valid_from: '2026-10-01', valid_until: '2026-11-01',
}

describe('Tarifas', () => {
  it('usa vigência com início inclusivo, fim exclusivo e tipo da caixa', () => {
    expect(activeRate([rate], 'normal', '2026-09-30')).toBeUndefined()
    expect(activeRate([rate], 'normal', '2026-10-01')).toBe(rate)
    expect(activeRate([rate], 'normal', '2026-10-31')).toBe(rate)
    expect(activeRate([rate], 'normal', '2026-11-01')).toBeUndefined()
    expect(activeRate([rate], 'moagem', '2026-10-01')).toBeUndefined()
    expect(activeRate([{ ...rate, valid_until: null }], 'normal', '2027-01-01')).toBeDefined()
  })

  it('converte reais em centavos sem arredondar a entrada', () => {
    expect(parseCents('5,01')).toBe(501)
    expect(parseCents('0.1')).toBe(10)
    for (const value of ['', '-1', '0', '1.001', '1e2', '9007199254740993']) {
      expect(parseCents(value)).toBeNull()
    }
    expect(money(501)).toBe('R$ 5,01')
  })
})
