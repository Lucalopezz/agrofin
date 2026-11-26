import type { ReactNode } from 'react'
import { displayDate, money } from '@/features/colheita/lib/rates'
import { entryTotal } from '@/features/colheita/lib/entries'
import { boxLabels, type Period } from '@/features/colheita/types'
import { sentenceCase } from '@/features/painel/lib/sentence-case'
import { Button } from '@/shared/components/ui/button'
import { useReportDetails } from '../hooks/use-period-summary'
import { saleLabels, type DetailResource, type PeriodSummary, type ReportDetails } from '../types'
import { periodLabel } from '../lib/period'

function DetailRows<R extends DetailResource>({
  resource, period, title, render,
}: {
  resource: R
  period: Period
  title: string
  render: (rows: ReportDetails[R]) => ReactNode
}) {
  const query = useReportDetails(resource, period)

  return (
    <div className="report-detail">
      <h3>{title}</h3>
      {query.isPending && <p role="status">Carregando lançamentos…</p>}
      {query.isError && (
        <div role="alert">
          <p>Não foi possível consultar estes lançamentos.</p>
          <Button type="button" variant="outline" onClick={() => void query.refetch()}>Tentar novamente</Button>
        </div>
      )}
      {query.isSuccess && (query.data.length > 0
        ? <ul className="report-list">{render(query.data)}</ul>
        : <p>Nenhum lançamento neste período.</p>)}
    </div>
  )
}

export function HarvestDetails({ summary }: { summary: PeriodSummary }) {
  const names = new Map(summary.harvest.by_employee.map((person) => [person.employee_id, person.employee_name]))
  return (
    <DetailRows resource="harvest" period={summary} title="Lançamentos de colheita" render={(rows) =>
      rows.map((row) => (
        <li key={row.id}>
          <strong>{names.get(row.employee_id) ?? 'Funcionário'}</strong>
          <span>{displayDate(row.harvest_date)} · {entryTotal(row)}</span>
          <ul>
            {row.items.map((item) => (
              <li key={item.box_type}>
                {boxLabels[item.box_type]} · {item.quantity} caixas a {money(item.applied_rate_cents)}
              </li>
            ))}
          </ul>
        </li>
      ))}
    />
  )
}

export function DailyWorkDetails({ period }: { period: Period }) {
  return (
    <DetailRows resource="dailyWork" period={period} title="Diárias do período" render={(rows) =>
      rows.map((row) => (
        <li key={row.id}>
          <strong>{row.employee_name} · {money(row.amount_cents)}</strong>
          <span>{displayDate(row.work_date)}</span>
          {row.notes && <span>{sentenceCase(row.notes)}</span>}
        </li>
      ))}
    />
  )
}

export function SoldDetails({ period }: { period: Period }) {
  return (
    <DetailRows resource="sold" period={period} title="Vendas do período" render={(rows) =>
      rows.map((row) => (
        <li key={row.id}>
          <strong>{row.buyer_name || 'Venda ao barracão'} · {displayDate(row.sold_at)}</strong>
          <ul>
            {row.items.map((item) => (
              <li key={item.sale_type}>
                {saleLabels[item.sale_type]} · {item.quantity} caixas a {money(item.unit_price_cents)}
                {' · '}{money(item.line_amount_cents)}
              </li>
            ))}
          </ul>
          <span>Bruto {money(row.gross_amount_cents)} · Líquido {money(row.net_amount_cents)}</span>
          {row.discount && <span>Desconto de {money(row.discount.amount_cents)} · {row.discount.reason}</span>}
        </li>
      ))}
    />
  )
}

function transactionDate(instant: string, timeZone: string) {
  return new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short', timeStyle: 'short', timeZone })
    .format(new Date(instant))
}

export function ReceiptDetails({ period, timeZone }: { period: Period; timeZone: string }) {
  return (
    <DetailRows resource="received" period={period} title="Recebimentos do período" render={(rows) =>
      rows.map((row) => (
        <li key={row.id}>
          <strong>{money(row.amount_cents)} · {transactionDate(row.received_at, timeZone)}</strong>
          <span>Venda de {displayDate(row.sold_at)}</span>
          <small className="report-reference">Referência da venda {row.sale_id}</small>
        </li>
      ))}
    />
  )
}

export function PaymentDetails({ period, timeZone }: { period: Period; timeZone: string }) {
  return (
    <DetailRows resource="paid" period={period} title="Pagamentos à equipe no período" render={(rows) =>
      rows.map((row) => (
        <li key={row.id}>
          <strong>{row.employee_name} · {money(row.amount_cents)}</strong>
          <span>Pago em {transactionDate(row.paid_at, timeZone)}</span>
          <span>Período trabalhado de {periodLabel(row)}</span>
          {row.notes && <span>{sentenceCase(row.notes)}</span>}
        </li>
      ))}
    />
  )
}

export function ExpenseDetails({ period }: { period: Period }) {
  return (
    <DetailRows resource="expenses" period={period} title="Despesas pagas no período" render={(rows) =>
      rows.map((row) => (
        <li key={row.id}>
          <strong>{sentenceCase(row.category)} · {money(row.amount_cents)}</strong>
          <span>Pago em {displayDate(row.paid_at)}</span>
          {row.description && <span>{sentenceCase(row.description)}</span>}
        </li>
      ))}
    />
  )
}
