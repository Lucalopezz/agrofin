import { useEffect, useId, useRef, useState } from 'react'
import { FormActions, SectionCard } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { money } from '@/shared/lib/money'
import { displayDate, periodLabel, type Period } from '@/shared/lib/period'
import type { useSaleWrite } from '../hooks/use-sale-write'
import { useSales } from '../hooks/use-sales'
import { saleLabels, type Sale } from '../types'
import { ReceiptForm } from './receipt-form'

type Props = {
  period: Period
  write: ReturnType<typeof useSaleWrite>
  selected: string | null
  newlySaved: string | null
  onSelect: (id: string | null) => void
  onReceipt: (saleId: string, cents: number) => Promise<boolean>
}

export function SaleHistory({ period, write, selected, newlySaved, onSelect, onReceipt }: Props) {
  const query = useSales(period)
  return (
    <SectionCard
      title="Vendas do período"
      subtitle={`Período de ${periodLabel(period)}`}
      action={<Button variant="outline" disabled={query.isFetching || write.busy}
        onClick={() => void query.refetch()}>Atualizar vendas</Button>}
    >
      {query.isPending && <p role="status">Carregando vendas…</p>}
      {query.isError && <p role="alert">Não foi possível carregar as vendas do período.</p>}
      {query.isSuccess && query.data.length === 0 && <p role="status">Nenhuma venda neste período.</p>}
      {query.data && query.data.length > 0 && (
        <ul className="sale-history">
          {query.data.map((sale) => (
            <SaleRow
              key={sale.id}
              sale={sale}
              expanded={selected === sale.id}
              focus={newlySaved === sale.id}
              write={write}
              canReceive={!query.isError}
              onSelect={() => onSelect(selected === sale.id ? null : sale.id)}
              onReceipt={(cents) => onReceipt(sale.id, cents)}
            />
          ))}
        </ul>
      )}
    </SectionCard>
  )
}

type RowProps = {
  sale: Sale
  expanded: boolean
  focus: boolean
  write: ReturnType<typeof useSaleWrite>
  canReceive: boolean
  onSelect: () => void
  onReceipt: (cents: number) => Promise<boolean>
}

function SaleRow({ sale, expanded, focus, write, canReceive, onSelect, onReceipt }: RowProps) {
  const row = useRef<HTMLLIElement>(null)
  const detailId = useId()
  useEffect(() => {
    // A venda recém-criada fica acessível para receber logo após o cadastro.
    if (focus) row.current?.focus()
  }, [focus])

  return (
    <li ref={row} tabIndex={-1}>
      <h3>{sale.buyer_name || 'Venda ao barracão'}</h3>
      <p>Vendida em {displayDate(sale.sold_at)}</p>
      <dl className="report-values">
        <div><dt>Líquido vendido</dt><dd>{money(sale.net_amount_cents)}</dd></div>
        <div><dt>Recebido da venda</dt><dd>{money(sale.received_amount_cents)}</dd></div>
        <div><dt>A receber</dt><dd>{money(sale.receivable_amount_cents)}</dd></div>
      </dl>
      <FormActions>
        <Button type="button" variant="outline" disabled={write.busy}
          aria-controls={detailId} aria-expanded={expanded} onClick={onSelect}>
          {expanded ? 'Ocultar detalhes' : 'Ver detalhes'}
        </Button>
      </FormActions>
      <div id={detailId} hidden={!expanded}>
        {expanded && <SaleDetail sale={sale} write={write} canReceive={canReceive} onReceipt={onReceipt} />}
      </div>
    </li>
  )
}

function SaleDetail({ sale, write, canReceive, onReceipt }: Pick<RowProps, 'sale' | 'write' | 'canReceive' | 'onReceipt'>) {
  const [receiving, setReceiving] = useState(false)

  function toggleReceipt(open: boolean) {
    write.reset()
    setReceiving(open)
  }

  async function save(cents: number) {
    if (await onReceipt(cents)) setReceiving(false)
  }

  return (
    <div className="sale-detail">
      <h4>Itens da venda</h4>
      <ul className="report-list">
        {sale.items.map((item) => (
          <li key={item.sale_type}>
            <strong>{saleLabels[item.sale_type]}</strong>
            <span>{item.quantity} caixas a {money(item.unit_price_cents)} · {money(item.line_amount_cents)}</span>
          </li>
        ))}
      </ul>
      <dl className="report-values">
        <div><dt>Valor bruto</dt><dd>{money(sale.gross_amount_cents)}</dd></div>
        <div><dt>Desconto</dt><dd>{money(sale.discount?.amount_cents ?? 0)}</dd></div>
      </dl>
      {sale.discount && <p>Motivo do desconto · {sale.discount.reason}</p>}
      {sale.notes && <p>{sale.notes}</p>}
      {sale.receivable_amount_cents > 0 && !receiving && (
        <Button disabled={write.busy || !canReceive} onClick={() => toggleReceipt(true)}>Registrar recebimento</Button>
      )}
      {receiving && (
        <ReceiptForm
          balance={sale.receivable_amount_cents}
          busy={write.busy}
          canReceive={canReceive}
          error={write.errorFor('receipt', sale.id)}
          onSave={(cents) => void save(cents)}
          onCancel={() => toggleReceipt(false)}
        />
      )}
    </div>
  )
}
