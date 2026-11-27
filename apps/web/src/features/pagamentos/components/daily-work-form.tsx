import { useState, type FormEvent } from 'react'
import { useEmployees } from '@/features/funcionarios/hooks/use-employees'
import {
  FieldInput, FieldSelect, FieldTextarea, FormActions, FormField, SectionCard,
} from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { parseCents } from '@/shared/lib/money'
import { isCivilDate, operationalDate, type Period } from '@/shared/lib/period'
import type { DailyWorkInput } from '../types'

type Props = {
  period: Period
  busy: boolean
  error: string | null
  onSave: (input: DailyWorkInput) => void
  onCancel: () => void
}

export function DailyWorkForm({ period, busy, error, onSave, onCancel }: Props) {
  const staff = useEmployees(true)
  const [employeeID, setEmployeeID] = useState('')
  const [date, setDate] = useState(() => {
    const today = operationalDate()
    return today >= period.start_date && today <= period.end_date ? today : period.end_date
  })
  const [amount, setAmount] = useState('')
  const [notes, setNotes] = useState('')
  const [attempted, setAttempted] = useState(false)
  const activeEmployee = staff.employees.some((employee) => employee.id === employeeID)
  const cents = parseCents(amount)

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy || staff.loading || staff.error) return
    setAttempted(true)
    if (!activeEmployee || !isCivilDate(date) || cents === null) return
    onSave({ employee_id: employeeID, work_date: date, amount_cents: cents, notes: notes.trim() })
  }

  return (
    <SectionCard title="Registrar diária" className="payment-editor-card">
      {staff.loading && <p role="status">Carregando funcionários…</p>}
      {staff.error && (
        <FormActions>
          <p role="alert">{staff.error}</p>
          <Button type="button" variant="outline" onClick={staff.reload}>Recarregar funcionários</Button>
        </FormActions>
      )}
      {!staff.loading && !staff.error && staff.employees.length === 0 && (
        <p>Nenhum funcionário ativo para registrar diária.</p>
      )}
      <form className="demo-form" aria-label="Registrar diária" onSubmit={save} noValidate>
        <div className="form-grid">
          <FormField label="Funcionário">
            <FieldSelect value={activeEmployee ? employeeID : ''} disabled={busy || staff.loading}
              required onChange={(event) => setEmployeeID(event.target.value)}>
              <option value="">Selecione um funcionário</option>
              {staff.employees.map((employee) => (
                <option key={employee.id} value={employee.id}>{employee.name}</option>
              ))}
            </FieldSelect>
          </FormField>
          <FormField label="Data do trabalho">
            <FieldInput type="date" value={date} required disabled={busy}
              onChange={(event) => setDate(event.target.value)} />
          </FormField>
        </div>
        {attempted && !activeEmployee && <p role="alert">Selecione um funcionário ativo.</p>}
        {attempted && !isCivilDate(date) && <p role="alert">Informe uma data de trabalho válida.</p>}
        <FormField label="Valor da diária (R$)">
          <FieldInput type="text" inputMode="decimal" value={amount} placeholder="0,00" required disabled={busy}
            onChange={(event) => setAmount(event.target.value)} />
        </FormField>
        {attempted && cents === null && <p role="alert">Informe um valor positivo com até duas casas decimais.</p>}
        <FormField label="Observação (opcional)">
          <FieldTextarea value={notes} disabled={busy} onChange={(event) => setNotes(event.target.value)} />
        </FormField>
        {error && <p role="alert">{error}</p>}
        <FormActions>
          <Button type="submit" disabled={busy || staff.loading || Boolean(staff.error) || staff.employees.length === 0}>
            Salvar diária
          </Button>
          <Button type="button" variant="outline" disabled={busy} onClick={onCancel}>Cancelar</Button>
        </FormActions>
      </form>
    </SectionCard>
  )
}
