import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import {
  FieldInput, FieldSelect, FormActions, FormField,
} from '@/features/painel/components/page-ui'
import { sentenceCase } from '@/features/painel/lib/sentence-case'
import { Button } from '@/shared/components/ui/button'
import { parseCents } from '@/shared/lib/money'
import { isCivilDate, operationalDate } from '@/shared/lib/period'
import type { Expense, ExpenseInput } from '../types'

const categories = [
  'Energia e irrigação',
  'Fertilizantes',
  'Defensivos',
  'Manutenção',
  'Outras despesas',
]

type Props = {
  expense?: Expense
  busy: boolean
  error: string | null
  onSave: (input: ExpenseInput) => Promise<boolean>
  onCancel?: () => void
}

function initialAmount(expense?: Expense) {
  // Um valor da API que perdeu precisão ao ser lido precisa ser redigitado
  // para evitar que a edição grave uma aproximação.
  if (!expense || !Number.isSafeInteger(expense.amount_cents)) return ''
  const cents = BigInt(expense.amount_cents)
  return `${cents / 100n},${String(cents % 100n).padStart(2, '0')}`
}

export function ExpenseForm({ expense, busy, error, onSave, onCancel }: Props) {
  const [date, setDate] = useState(() => expense?.paid_at ?? operationalDate())
  const [category, setCategory] = useState(() => expense?.category ?? categories[0]!)
  const [description, setDescription] = useState(() => expense?.description ?? '')
  const [amount, setAmount] = useState(() => initialAmount(expense))
  const [attempted, setAttempted] = useState(false)
  const dateField = useRef<HTMLInputElement>(null)
  const dateErrorId = useId()
  const amountErrorId = useId()
  const editing = Boolean(expense)
  const cents = parseCents(amount)
  const validDate = isCivilDate(date)
  const dateError = attempted && !validDate
  const amountError = attempted && cents === null

  useEffect(() => {
    if (editing) dateField.current?.focus()
  }, [editing])

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setAttempted(true)
    if (!validDate || cents === null) return

    const saved = await onSave({
      paid_at: date,
      category,
      description: description.trim(),
      amount_cents: cents,
    })
    if (!expense && saved) {
      setAmount('')
      setDescription('')
      setAttempted(false)
    }
  }

  return (
    <form className="demo-form" aria-label={expense ? 'Editar despesa' : 'Nova despesa'} onSubmit={save} noValidate>
      <div className="form-grid">
        <div>
          <FormField label="Data de pagamento">
            <FieldInput
              ref={dateField}
              type="date"
              value={date}
              required
              disabled={busy}
              aria-invalid={dateError || undefined}
              aria-describedby={dateError ? dateErrorId : undefined}
              onChange={(event) => setDate(event.target.value)}
            />
          </FormField>
          {dateError && <p id={dateErrorId} role="alert">Informe uma data de pagamento válida.</p>}
        </div>
        <FormField label="Categoria">
          <FieldSelect value={category} disabled={busy} onChange={(event) => setCategory(event.target.value)}>
            {expense && !categories.includes(expense.category) && (
              <option value={expense.category}>{sentenceCase(expense.category)}</option>
            )}
            {categories.map((option) => <option key={option}>{option}</option>)}
          </FieldSelect>
        </FormField>
      </div>
      <FormField label="Descrição (opcional)">
        <FieldInput
          value={description}
          disabled={busy}
          onChange={(event) => setDescription(event.target.value)}
        />
      </FormField>
      <div>
        <FormField label="Valor (R$)">
          <FieldInput
            type="text"
            inputMode="decimal"
            value={amount}
            placeholder="0,00"
            required
            disabled={busy}
            aria-invalid={amountError || undefined}
            aria-describedby={amountError ? amountErrorId : undefined}
            onChange={(event) => setAmount(event.target.value)}
          />
        </FormField>
        {amountError && <p id={amountErrorId} role="alert">Informe um valor positivo com até duas casas decimais.</p>}
      </div>
      {error && <p role="alert">{error}</p>}
      <FormActions>
        <Button type="submit" disabled={busy}>{expense ? 'Salvar alteração' : 'Registrar despesa'}</Button>
        {onCancel && <Button type="button" variant="outline" disabled={busy} onClick={onCancel}>Cancelar</Button>}
      </FormActions>
    </form>
  )
}
