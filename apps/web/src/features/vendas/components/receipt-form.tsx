import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { FieldInput, FormActions, FormField } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { centsInput, money, parseCents } from '@/shared/lib/money'

type Props = {
  balance: number
  busy: boolean
  canReceive: boolean
  error: string | null
  onSave: (cents: number) => void
  onCancel: () => void
}

export function ReceiptForm({ balance, busy, canReceive, error, onSave, onCancel }: Props) {
  const [amount, setAmount] = useState(() => centsInput(balance))
  const [attempted, setAttempted] = useState(false)
  const field = useRef<HTMLInputElement>(null)
  const errorId = useId()
  const cents = parseCents(amount)
  const amountError = cents === null
    ? 'Informe um valor positivo com até duas casas decimais.'
    : cents > balance ? 'O valor excede o saldo a receber.' : null
  const showError = attempted && amountError !== null

  useEffect(() => {
    field.current?.focus()
    field.current?.select()
  }, [])

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy || !canReceive) return
    setAttempted(true)
    if (amountError || cents === null) return
    onSave(cents)
  }

  return (
    <form className="demo-form" aria-label="Registrar recebimento" onSubmit={save} noValidate>
      <FormField label="Valor recebido (R$)">
        <FieldInput
          ref={field} type="text" inputMode="decimal" value={amount} required disabled={busy}
          aria-invalid={showError || undefined}
          aria-describedby={showError ? errorId : undefined}
          onChange={(event) => setAmount(event.target.value)}
        />
      </FormField>
      {showError && <p id={errorId} role="alert">{amountError}</p>}
      {cents !== null && !amountError && Number.isSafeInteger(balance) && (
        <div className="payment-preview">
          <span>Saldo previsto após o recebimento</span>
          <strong>{money(balance - cents)}</strong>
        </div>
      )}
      {error && <p role="alert">{error}</p>}
      <FormActions>
        <Button type="submit" disabled={busy || !canReceive || balance <= 0}>Confirmar recebimento</Button>
        <Button type="button" variant="outline" disabled={busy} onClick={onCancel}>Cancelar</Button>
      </FormActions>
    </form>
  )
}
