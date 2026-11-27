import { useState } from 'react'
import { weekOf } from '@/features/colheita/lib/entries'
import { Feedback, PageHeading } from '@/features/painel/components/page-ui'
import { PeriodFilter } from '@/features/painel/components/period-filter'
import { money } from '@/shared/lib/money'
import { displayDate, operationalDate } from '@/shared/lib/period'
import { SaleForm } from '../components/sale-form'
import { SaleHistory } from '../components/sale-history'
import { useSaleWrite } from '../hooks/use-sale-write'
import type { SaleInput } from '../types'

export function VendasPage() {
  const [period, setPeriod] = useState(() => weekOf(operationalDate()))
  const [selected, setSelected] = useState<string | null>(null)
  const [newlySaved, setNewlySaved] = useState<string | null>(null)
  const [notice, setNotice] = useState('')
  const write = useSaleWrite()

  async function saveSale(input: SaleInput) {
    setNotice('')
    const result = await write.submit({ kind: 'sale', input })
    if (result?.kind !== 'sale') return false
    const sale = result.saved

    // Abre o lançamento confirmado, inclusive quando o cliente informa
    // uma data fora do filtro atual para registrar o acerto retroativo.
    if (sale.sold_at < period.start_date || sale.sold_at > period.end_date) {
      setPeriod(weekOf(sale.sold_at))
    }
    setSelected(sale.id)
    setNewlySaved(sale.id)
    setNotice(`Venda registrada. Líquido vendido de ${money(sale.net_amount_cents)}.`)
    return true
  }

  async function saveReceipt(saleId: string, cents: number) {
    setNotice('')
    const result = await write.submit({ kind: 'receipt', saleId, cents })
    if (result?.kind !== 'receipt') return false
    const date = displayDate(operationalDate(new Date(result.saved.received_at)))
    setNotice(`Recebimento registrado em ${date}.`)
    return true
  }

  return (
    <>
      <PageHeading eyebrow="Financeiro" title="Vendas" description="Registre vendas ao barracão e acompanhe os recebimentos." />
      {notice && <Feedback>{notice}</Feedback>}
      <div className="form-page-grid sales-entry-grid">
        <SaleForm busy={write.busy} error={write.errorFor('sale')} onSave={saveSale} />
      </div>
      <PeriodFilter
        key={`${period.start_date}/${period.end_date}`}
        period={period}
        disabled={write.busy}
        onApply={(next) => {
          setPeriod(next)
          setSelected(null)
          setNewlySaved(null)
          setNotice('')
          write.reset()
        }}
      />
      <SaleHistory
        period={period}
        write={write}
        selected={selected}
        newlySaved={newlySaved}
        onSelect={(id) => {
          setNewlySaved(null)
          setSelected(id)
          write.reset()
        }}
        onReceipt={saveReceipt}
      />
    </>
  )
}
