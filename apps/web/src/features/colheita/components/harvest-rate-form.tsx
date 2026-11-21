import { useState, type FormEvent } from 'react'
import {
  Feedback,
  FieldInput,
  FieldSelect,
  FormField,
} from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { useCreateHarvestRate } from '../hooks/use-create-harvest-rate'
import { localDate, parseCents } from '../lib/rates'
import { boxLabels, boxTypes, type BoxType } from '../types'
import { RequestError } from './request-error'

export function HarvestRateForm() {
  const create = useCreateHarvestRate()
  const [type, setType] = useState<BoxType>('normal')
  const [amount, setAmount] = useState('')
  const [from, setFrom] = useState(() => localDate())
  const [until, setUntil] = useState('')
  const cents = parseCents(amount)
  const invalidPeriod = Boolean(until && until <= from)
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!cents || !from || invalidPeriod) return
    const saved = await create.submit({
      box_type: type,
      amount_cents: cents,
      valid_from: from,
      valid_until: until || null,
    })
    if (saved) setAmount('')
  }

  return (
    <form className="demo-form" onSubmit={submit}>
      <fieldset className="form-grid" disabled={create.isPending}>
        <FormField label="Tipo de caixa">
          <FieldSelect value={type} onChange={(e) => setType(e.target.value as BoxType)}>
            {boxTypes.map((box) => (
              <option key={box} value={box}>
                {boxLabels[box]}
              </option>
            ))}
          </FieldSelect>
        </FormField>
        <FormField label="Valor por caixa (R$)">
          <FieldInput
            inputMode="decimal"
            required
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </FormField>
        <FormField label="Início da vigência">
          <FieldInput
            type="date"
            required
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
        </FormField>
        <FormField
          label="Fim da vigência"
          hint="Primeiro dia sem esta tarifa. Sem fim, ela impede outra tarifa futura do mesmo tipo."
        >
          <FieldInput type="date" value={until} onChange={(e) => setUntil(e.target.value)} />
        </FormField>
      </fieldset>
      {amount && !cents && <p>Informe um valor positivo com até duas casas decimais.</p>}
      {invalidPeriod && <p>O fim deve ser posterior ao início da vigência.</p>}
      <Button disabled={!cents || !from || invalidPeriod || create.isPending}>
        Cadastrar tarifa
      </Button>
      <RequestError error={create.error} />
      {create.isSuccess && <Feedback>Tarifa cadastrada.</Feedback>}
    </form>
  )
}
