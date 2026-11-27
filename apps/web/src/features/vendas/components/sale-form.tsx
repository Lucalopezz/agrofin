import { useState, type FormEvent } from 'react'
import {
  FieldInput, FieldTextarea, FormActions, FormField, SectionCard,
} from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { money } from '@/shared/lib/money'
import { isCivilDate, operationalDate } from '@/shared/lib/period'
import { emptySaleLines, saleDraftTotals, type SaleLines } from '../lib/sale-draft'
import { saleLabels, saleTypes, type SaleInput, type SaleType } from '../types'

type Props = {
  busy: boolean
  error: string | null
  onSave: (input: SaleInput) => Promise<boolean>
}

export function SaleForm({ busy, error, onSave }: Props) {
  const [date, setDate] = useState(operationalDate)
  const [buyer, setBuyer] = useState('')
  const [notes, setNotes] = useState('')
  const [lines, setLines] = useState(emptySaleLines)
  const [discount, setDiscount] = useState('')
  const [reason, setReason] = useState('')
  const [attempted, setAttempted] = useState(false)
  const preview = saleDraftTotals(lines, discount, reason)

  function changeLine(type: SaleType, field: keyof SaleLines[SaleType], value: string) {
    setLines((current) => ({ ...current, [type]: { ...current[type], [field]: value } }))
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setAttempted(true)
    if (!isCivilDate(date) || preview.errors.length > 0) return

    if (await onSave({
      sold_at: date,
      buyer_name: buyer.trim(),
      notes: notes.trim(),
      items: preview.items,
      discount: preview.discount,
    })) {
      // A data e o barracão podem ser reaproveitados no próximo lançamento.
      setLines(emptySaleLines())
      setDiscount('')
      setReason('')
      setNotes('')
      setAttempted(false)
    }
  }

  return (
    <SectionCard title="Nova venda">
      <form className="demo-form" aria-label="Registrar venda" onSubmit={save} noValidate>
        <div className="form-grid">
          <FormField label="Data da venda">
            <FieldInput
              type="date" value={date} disabled={busy} required
              aria-invalid={attempted && !isCivilDate(date) || undefined}
              onChange={(event) => setDate(event.target.value)}
            />
          </FormField>
          <FormField label="Barracão (opcional)">
            <FieldInput value={buyer} disabled={busy} onChange={(event) => setBuyer(event.target.value)} />
          </FormField>
        </div>
        {attempted && !isCivilDate(date) && <p role="alert">Informe uma data de venda válida.</p>}

        <div className="sale-lines">
          {saleTypes.map((type) => (
            <fieldset key={type} disabled={busy}>
              <legend>{saleLabels[type]}</legend>
              <div className="form-grid">
                <FormField label={`Caixas · ${saleLabels[type]}`}>
                  <FieldInput
                    type="text" inputMode="numeric" value={lines[type].quantity}
                    onChange={(event) => changeLine(type, 'quantity', event.target.value)}
                  />
                </FormField>
                <FormField label={`Preço por caixa (R$) · ${saleLabels[type]}`}>
                  <FieldInput
                    type="text" inputMode="decimal" value={lines[type].price} placeholder="0,00"
                    onChange={(event) => changeLine(type, 'price', event.target.value)}
                  />
                </FormField>
              </div>
            </fieldset>
          ))}
        </div>

        <div className="form-grid">
          <FormField label="Desconto (R$, opcional)">
            <FieldInput type="text" inputMode="decimal" value={discount} disabled={busy}
              onChange={(event) => setDiscount(event.target.value)} />
          </FormField>
          {discount.trim() && (
            <FormField label="Motivo do desconto">
              <FieldInput value={reason} disabled={busy} onChange={(event) => setReason(event.target.value)} />
            </FormField>
          )}
        </div>
        <FormField label="Observação (opcional)">
          <FieldTextarea value={notes} disabled={busy} onChange={(event) => setNotes(event.target.value)} />
        </FormField>
        {preview.errors.length === 0 && (
          <div>
            <h3>Prévia da venda</h3>
            <dl className="report-values">
              <div><dt>Bruto previsto</dt><dd>{money(preview.gross)}</dd></div>
              <div><dt>Desconto previsto</dt><dd>{money(preview.discountCents)}</dd></div>
              <div><dt>Líquido previsto</dt><dd>{money(preview.net)}</dd></div>
            </dl>
          </div>
        )}
        {attempted && preview.errors.length > 0 && (
          <ul role="alert">{preview.errors.map((message) => <li key={message}>{message}</li>)}</ul>
        )}
        {error && <p role="alert">{error}</p>}
        <FormActions>
          <Button type="submit" disabled={busy}>Registrar venda</Button>
        </FormActions>
      </form>
    </SectionCard>
  )
}
