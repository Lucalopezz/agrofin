import { useState, type FormEvent } from 'react'
import { Feedback, FieldInput, FieldSelect, FormField, SectionCard } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { useHarvestRates } from '../hooks/use-harvest-rates'
import { useCreateHarvestRate } from '../hooks/use-create-harvest-rate'
import { displayDate, localDate, money, parseCents } from '../lib/rates'
import { boxLabels, boxTypes, type BoxType } from '../types'
import { RequestError } from './request-error'

export function HarvestRates() {
  const rates = useHarvestRates()
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
      box_type: type, amount_cents: cents, valid_from: from, valid_until: until || null,
    })
    if (saved) setAmount('')
  }

  return (
    <SectionCard title="Tarifas de colheita">
      <details>
        <summary>Consultar e cadastrar tarifas</summary>
        <form className="demo-form" onSubmit={submit}>
          <fieldset className="form-grid" disabled={create.isPending}>
            <FormField label="Tipo de caixa">
              <FieldSelect value={type} onChange={(e) => setType(e.target.value as BoxType)}>
                {boxTypes.map((box) => <option key={box} value={box}>{boxLabels[box]}</option>)}
              </FieldSelect>
            </FormField>
            <FormField label="Valor por caixa (R$)">
              <FieldInput inputMode="decimal" required value={amount} onChange={(e) => setAmount(e.target.value)} />
            </FormField>
            <FormField label="Início da vigência">
              <FieldInput type="date" required value={from} onChange={(e) => setFrom(e.target.value)} />
            </FormField>
            <FormField label="Fim da vigência" hint="Primeiro dia sem esta tarifa. Deixe vazio para vigência aberta.">
              <FieldInput type="date" value={until} onChange={(e) => setUntil(e.target.value)} />
            </FormField>
          </fieldset>
          {amount && !cents && <p>Informe um valor positivo com até duas casas decimais.</p>}
          {invalidPeriod && <p>O fim deve ser posterior ao início da vigência.</p>}
          <Button disabled={!cents || !from || invalidPeriod || create.isPending}>Cadastrar tarifa</Button>
          <RequestError error={create.error} />
          {create.isSuccess && <Feedback>Tarifa cadastrada.</Feedback>}
        </form>
        <h3>Tarifas cadastradas</h3>
        {rates.isPending && <p role="status">Carregando tarifas...</p>}
        <RequestError error={rates.error} retry={() => void rates.refetch()} />
        {rates.isSuccess && (rates.data.length ? (
          <ul className="employee-list">
            {rates.data.map((rate) => (
              <li key={rate.id}>
                <strong>{boxLabels[rate.box_type]} · {money(rate.amount_cents)}</strong>
                <span>Desde {displayDate(rate.valid_from)} · {rate.valid_until
                  ? `Fim exclusivo em ${displayDate(rate.valid_until)}` : 'Sem fim definido'}</span>
              </li>
            ))}
          </ul>
        ) : <p>Nenhuma tarifa cadastrada.</p>)}
      </details>
    </SectionCard>
  )
}
