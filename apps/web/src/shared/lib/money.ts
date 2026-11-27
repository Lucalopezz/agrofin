export function parseCents(value: string) {
  const match = /^(\d+)(?:[.,](\d{1,2}))?$/.exec(value.trim())
  if (!match) return null
  // Monta centavos a partir dos dígitos para preservar os decimais digitados.
  const cents = Number(match[1] + (match[2] ?? '').padEnd(2, '0'))
  return Number.isSafeInteger(cents) && cents > 0 ? cents : null
}

export function money(cents: number | bigint) {
  if (typeof cents === 'number' && !Number.isSafeInteger(cents))
    return 'Valor fora do limite de exibição'
  const value = BigInt(cents)
  // Mantém o sinal mesmo para saldos negativos com menos de um real.
  const sign = value < 0n ? '-' : ''
  const absolute = value < 0n ? -value : value
  return `${sign}R$ ${(absolute / 100n).toLocaleString('pt-BR')},${String(absolute % 100n).padStart(2, '0')}`
}
