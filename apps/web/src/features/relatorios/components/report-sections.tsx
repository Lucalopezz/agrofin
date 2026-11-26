import { useId, useState, type ReactNode } from 'react'
import { money } from '@/features/colheita/lib/rates'
import { SectionCard } from '@/features/painel/components/page-ui'
import { sentenceCase } from '@/features/painel/lib/sentence-case'
import { Button } from '@/shared/components/ui/button'
import { saleLabels, type PeriodSummary, type SaleType } from '../types'
import {
  DailyWorkDetails, ExpenseDetails, HarvestDetails, PaymentDetails,
  ReceiptDetails, SoldDetails,
} from './report-details'

function count(value: number) {
  return value.toLocaleString('pt-BR')
}

function Total({ label, value }: { label: string; value: string }) {
  return <div><dt>{label}</dt><dd>{value}</dd></div>
}

function DetailToggle({ label, children }: { label: string; children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const id = useId()
  return (
    <div className="report-disclosure">
      <Button type="button" variant="outline" aria-expanded={open} aria-controls={id}
        onClick={() => setOpen(!open)}>
        {open ? 'Ocultar' : 'Ver'} {label}
      </Button>
      <div id={id} hidden={!open}>{open && children}</div>
    </div>
  )
}

export function ReportSections({ summary }: { summary: PeriodSummary }) {
  const { harvest, sales, payroll, cash, time_zone: timeZone } = summary
  const empty = harvest.total_boxes === 0 && sales.total_boxes === 0 &&
    payroll.due_amount_cents === 0 && payroll.paid_amount_cents === 0 &&
    cash.received_amount_cents === 0 && cash.paid_expenses_amount_cents === 0

  return (
    <div className="report-sections">
      {empty && <p role="status">Nenhum lançamento neste período.</p>}
      <SectionCard title="Colheita">
        <dl className="report-values">
          <Total label="Caixas colhidas" value={count(harvest.total_boxes)} />
          <Total label="Caixas normais" value={count(harvest.normal_boxes)} />
          <Total label="Caixas de moagem" value={count(harvest.milling_boxes)} />
        </dl>
        <DetailToggle label="caixas por funcionário e lançamentos">
          {harvest.by_employee.length > 0 && (
            <ul className="report-list">
              {harvest.by_employee.map((person) => (
                <li key={person.employee_id}>
                  <strong>{person.employee_name}</strong>
                  <span>Normais · {count(person.normal_boxes)} | Moagem · {count(person.milling_boxes)}</span>
                </li>
              ))}
            </ul>
          )}
          <HarvestDetails summary={summary} />
        </DetailToggle>
      </SectionCard>

      <SectionCard title="Vendas">
        <dl className="report-values">
          <Total label="Caixas vendidas" value={count(sales.total_boxes)} />
          {(Object.keys(saleLabels) as SaleType[]).map((type) => (
            <Total key={type} label={saleLabels[type]} value={count(sales.boxes_by_type[type])} />
          ))}
          <Total label="Valor bruto vendido" value={money(sales.gross_amount_cents)} />
          <Total label="Descontos" value={money(sales.discount_amount_cents)} />
          <Total label="Valor líquido vendido" value={money(sales.net_amount_cents)} />
          <Total label="Preço médio líquido por caixa" value={sales.average_net_price_cents === null
            ? 'Sem vendas no período'
            : money(sales.average_net_price_cents)} />
        </dl>
        <DetailToggle label="vendas e descontos"><SoldDetails period={summary} /></DetailToggle>
      </SectionCard>

      <SectionCard title="Equipe">
        <dl className="report-values">
          <Total label="Valor devido à equipe" value={money(payroll.due_amount_cents)} />
          <Total label="Valor pago à equipe" value={money(payroll.paid_amount_cents)} />
        </dl>
        <DetailToggle label="valores devidos por funcionário">
          {payroll.by_employee.length > 0 && (
            <ul className="report-list">
              {payroll.by_employee.map((person) => (
                <li key={person.employee_id}>
                  <strong>{person.employee_name} · {money(person.due_amount_cents)}</strong>
                  <span>Colheita · {money(person.harvest_amount_cents)}</span>
                  <span>Diárias · {money(person.daily_work_amount_cents)}</span>
                </li>
              ))}
            </ul>
          )}
          <HarvestDetails summary={summary} />
          <DailyWorkDetails period={summary} />
        </DetailToggle>
        <DetailToggle label="pagamentos à equipe"><PaymentDetails period={summary} timeZone={timeZone} /></DetailToggle>
      </SectionCard>

      <SectionCard title="Caixa">
        <dl className="report-values">
          <Total label="Valor recebido" value={money(cash.received_amount_cents)} />
          <Total label="Despesas pagas" value={money(cash.paid_expenses_amount_cents)} />
          {Object.entries(cash.paid_expenses_by_category).map(([category, amount]) => (
            <Total key={category} label={sentenceCase(category)} value={money(amount)} />
          ))}
          <Total label="Saldo de caixa do período" value={money(cash.balance_amount_cents)} />
        </dl>
        <p>Recebimentos menos pagamentos à equipe e despesas pagas no período.</p>
        <DetailToggle label="recebimentos"><ReceiptDetails period={summary} timeZone={timeZone} /></DetailToggle>
        <DetailToggle label="despesas pagas"><ExpenseDetails period={summary} /></DetailToggle>
        <DetailToggle label="lançamentos do saldo de caixa">
          <ReceiptDetails period={summary} timeZone={timeZone} />
          <PaymentDetails period={summary} timeZone={timeZone} />
          <ExpenseDetails period={summary} />
        </DetailToggle>
      </SectionCard>
    </div>
  )
}
