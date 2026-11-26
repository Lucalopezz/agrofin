import { money } from '@/features/colheita/lib/rates'
import type { PeriodSummary } from '../types'

export function exportPeriodSummary(summary: PeriodSummary) {
  const { harvest, sales, payroll, cash } = summary
  const rows = [
    ['Início', summary.start_date],
    ['Fim', summary.end_date],
    ['Indicador', 'Valor'],
    ['Caixas colhidas', harvest.total_boxes],
    ['Caixas normais colhidas', harvest.normal_boxes],
    ['Caixas de moagem colhidas', harvest.milling_boxes],
    ['Caixas vendidas', sales.total_boxes],
    ['Caixas vendidas para exportação', sales.boxes_by_type.exportacao],
    ['Caixas vendidas para mercado', sales.boxes_by_type.mercado],
    ['Caixas vendidas para moagem', sales.boxes_by_type.moagem],
    ['Valor bruto vendido', money(sales.gross_amount_cents)],
    ['Descontos', money(sales.discount_amount_cents)],
    ['Valor líquido vendido', money(sales.net_amount_cents)],
    ['Preço médio líquido por caixa', sales.average_net_price_cents === null ? '' : money(sales.average_net_price_cents)],
    ['Valor recebido', money(cash.received_amount_cents)],
    ['Devido à equipe', money(payroll.due_amount_cents)],
    ['Pago à equipe', money(payroll.paid_amount_cents)],
    ['Despesas pagas', money(cash.paid_expenses_amount_cents)],
    ['Saldo de caixa do período', money(cash.balance_amount_cents)],
  ]
  // Os rótulos são fixos, sem nomes ou descrições digitados por usuários.
  const csv = `\uFEFF${rows.map((row) => row.join(';')).join('\r\n')}`
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = `agrofin-resumo-${summary.start_date}-a-${summary.end_date}.csv`
  link.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}
