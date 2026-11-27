import { useId, useState } from 'react'
import { FormActions, SectionCard } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { money } from '@/shared/lib/money'
import { periodLabel, type Period } from '@/shared/lib/period'
import type { EmployeePayroll, PaymentInput, PayrollSummary } from '../types'
import { PaymentForm } from './payment-form'
import { PayrollDetails } from './payroll-details'

type PayrollTeamProps = {
  summary: PayrollSummary
  busy: boolean
  canPay: boolean
  error: string | null
  onSave: (input: PaymentInput) => Promise<boolean>
  clearError: () => void
}

type TotalField = 'due_amount_cents' | 'paid_amount_cents' | 'remaining_amount_cents'

function total(employees: EmployeePayroll[], field: TotalField) {
  let sum = 0n
  for (const employee of employees) {
    const cents = employee[field]
    if (!Number.isSafeInteger(cents)) return 'Valor fora do limite de exibição'
    sum += BigInt(cents)
  }
  return money(sum)
}

export function PayrollTeam({ summary, busy, canPay, error, onSave, clearError }: PayrollTeamProps) {
  const [selected, setSelected] = useState<string | null>(null)
  const { period, employees } = summary

  function select(id: string | null) {
    clearError()
    setSelected(id)
  }

  async function save(input: PaymentInput) {
    if (await onSave(input)) setSelected(null)
  }

  return (
    <SectionCard title="Equipe do período" subtitle={`Período de ${periodLabel(period)}`}>
      {employees.length === 0 ? (
        <p role="status">Nenhum lançamento de colheita ou diária neste período.</p>
      ) : (
        <>
          <dl className="payroll-totals">
            <div><dt>Total devido</dt><dd><strong>{total(employees, 'due_amount_cents')}</strong></dd></div>
            <div><dt>Total pago</dt><dd><strong>{total(employees, 'paid_amount_cents')}</strong></dd></div>
            <div><dt>Total pendente</dt><dd><strong>{total(employees, 'remaining_amount_cents')}</strong></dd></div>
          </dl>
          <ul className="payroll-employee-list">
            {employees.map((employee) => (
              <PayrollEmployee
                key={employee.employee.id}
                employee={employee}
                period={period}
                isPaying={selected === employee.employee.id}
                busy={busy}
                canPay={canPay}
                error={error}
                onPay={() => select(employee.employee.id)}
                onCancel={() => select(null)}
                onSave={(input) => void save(input)}
              />
            ))}
          </ul>
        </>
      )}
    </SectionCard>
  )
}

type EmployeeProps = {
  employee: EmployeePayroll
  period: Period
  isPaying: boolean
  busy: boolean
  canPay: boolean
  error: string | null
  onPay: () => void
  onCancel: () => void
  onSave: (input: PaymentInput) => void
}

function PayrollEmployee({
  employee, period, isPaying, busy, canPay, error, onPay, onCancel, onSave,
}: EmployeeProps) {
  const [details, setDetails] = useState(false)
  const detailId = useId()
  return (
    <li>
      <h3>{employee.employee.name}</h3>
      <dl className="report-values">
        <div><dt>Caixas normais</dt><dd>{employee.normal_boxes} · {money(employee.normal_amount_cents)}</dd></div>
        <div><dt>Caixas de moagem</dt><dd>{employee.milling_boxes} · {money(employee.milling_amount_cents)}</dd></div>
        <div><dt>Valor das caixas</dt><dd>{money(employee.harvest_amount_cents)}</dd></div>
        <div><dt>Diárias</dt><dd>{money(employee.daily_work_amount_cents)}</dd></div>
        <div><dt>Devido</dt><dd>{money(employee.due_amount_cents)}</dd></div>
        <div><dt>Pago</dt><dd>{money(employee.paid_amount_cents)}</dd></div>
        <div><dt>Saldo pendente</dt><dd>{money(employee.remaining_amount_cents)}</dd></div>
      </dl>
      <FormActions>
        <Button type="button" variant="outline" aria-expanded={details} aria-controls={detailId}
          onClick={() => setDetails(!details)}>
          {details ? 'Ocultar lançamentos' : 'Ver lançamentos'}
        </Button>
        {!isPaying && employee.remaining_amount_cents > 0 && (
          <Button type="button" disabled={busy || !canPay} onClick={onPay}>Pagar</Button>
        )}
        {employee.remaining_amount_cents === 0 && <strong>Quitado</strong>}
      </FormActions>
      {isPaying && (
        <PaymentForm
          employee={employee}
          period={period}
          busy={busy}
          canPay={canPay}
          error={error}
          onSave={onSave}
          onCancel={onCancel}
        />
      )}
      <div id={detailId} hidden={!details}>
        {details && <PayrollDetails employeeId={employee.employee.id} period={period} />}
      </div>
    </li>
  )
}
