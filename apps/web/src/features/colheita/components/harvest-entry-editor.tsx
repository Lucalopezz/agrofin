import { useState, type FormEvent } from 'react'
import { FormActions } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { useUpdateHarvestEntry } from '../hooks/use-update-harvest-entry'
import { validQuantity } from '../lib/entries'
import { money } from '../lib/rates'
import type { HarvestEntry } from '../types'
import { QuantityField } from './quantity-field'
import { RequestError } from './request-error'

type Props = { entry: HarvestEntry; onSaved: (entry: HarvestEntry) => void; onCancel: () => void }

export function HarvestEntryEditor({ entry, onSaved, onCancel }: Props) {
  // Somente tipos já fotografados aparecem na correção. Cada tarifa segue vinculada ao item.
  const [draft, setDraft] = useState(() => Object.fromEntries(
    entry.items.map((item) => [item.box_type, String(item.quantity)]),
  ))
  const update = useUpdateHarvestEntry()
  const items = entry.items.map(({ box_type }) => ({ box_type, quantity: Number(draft[box_type]) }))
  const valid = items.every(({ quantity }) => validQuantity(quantity))
  const changed = items.some((item, index) => item.quantity !== entry.items[index].quantity)

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!valid || !changed) return
    const saved = await update.submit({ id: entry.id, items })
    if (saved) onSaved(saved)
  }

  return (
    <form className="demo-form" onSubmit={submit} aria-label="Corrigir quantidades">
      <fieldset className="form-grid" disabled={update.isPending}>
        {entry.items.map(({ box_type, applied_rate_cents }) => (
          <QuantityField key={box_type} type={box_type} value={draft[box_type]} required
            hint={`Tarifa aplicada: ${money(applied_rate_cents)}`}
            onChange={(value) => setDraft({ ...draft, [box_type]: value })} />
        ))}
      </fieldset>
      <RequestError error={update.error} />
      <FormActions>
        <Button type="submit" disabled={!valid || !changed || update.isPending}>Salvar correção</Button>
        <Button type="button" variant="outline" onClick={onCancel} disabled={update.isPending}>Cancelar</Button>
      </FormActions>
    </form>
  )
}
