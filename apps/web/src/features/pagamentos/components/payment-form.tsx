import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { FieldInput, FieldTextarea, FormActions, FormField } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { centsInput, money, parseCents } from '@/shared/lib/money'
import type { Period } from '@/shared/lib/period'
import type { EmployeePayroll, PaymentInput } from '../types'

type Props = {
  employee: EmployeePayroll
  period: Period
  busy: boolean
  canPay: boolean
  error: string | null
  onSave: (input: PaymentInput) => void
  onCancel: () => void
}

export function PaymentForm({ employee, period, busy, canPay, error, onSave, onCancel }: Props) {
  const remaining = employee.remaining_amount_cents
  const [amount, setAmount] = useState(() => centsInput(remaining))
  const [notes, setNotes] = useState('')
  const [attempted, setAttempted] = useState(false)
  const amountField = useRef<HTMLInputElement>(null)
  const errorId = useId()
  const cents = parseCents(amount)
  const amountError = cents === null
    ? 'Informe um valor positivo com até duas casas decimais.'
    : cents > remaining ? 'O valor excede o saldo pendente.' : null
  const showAmountError = attempted && amountError !== null

  useEffect(() => {
    amountField.current?.focus()
    amountField.current?.select()
  }, [])

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy || !canPay) return
    setAttempted(true)
    if (amountError || cents === null) return

    // A API usa o mesmo intervalo da folha e gera a data efetiva do pagamento.
    onSave({ ...period, employee_id: employee.employee.id, amount_cents: cents, notes: notes.trim() })
  }

  return (
    <form className="demo-form payroll-entry-form" aria-label="Registrar pagamento" onSubmit={save} noValidate>
      <FormField label="Valor do pagamento (R$)">
        <FieldInput
          ref={amountField}
          value={amount}
          onChange={(event) => setAmount(event.target.value)}
          type="text"
          inputMode="decimal"
          required
          disabled={busy}
          aria-invalid={showAmountError || undefined}
          aria-describedby={showAmountError ? errorId : undefined}
        />
      </FormField>
      {showAmountError && <p id={errorId} role="alert">{amountError}</p>}
      <FormField label="Observação (opcional)">
        <FieldTextarea value={notes} disabled={busy} onChange={(event) => setNotes(event.target.value)} />
      </FormField>
      {cents !== null && !amountError && Number.isSafeInteger(remaining) && (
        <div className="payment-preview">
          <span>Saldo previsto após o pagamento</span>
          <strong>{money(remaining - cents)}</strong>
        </div>
      )}
      {error && <p role="alert">{error}</p>}
      <FormActions>
        <Button type="submit" disabled={busy || !canPay || remaining <= 0}>Confirmar pagamento</Button>
        <Button type="button" variant="outline" disabled={busy} onClick={onCancel}>Cancelar</Button>
      </FormActions>
    </form>
  )
}
