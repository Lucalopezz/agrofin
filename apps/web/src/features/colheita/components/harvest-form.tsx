import { useState, type FormEvent } from 'react'
import { useEmployees } from '@/features/funcionarios/hooks/use-employees'
import {
  Feedback,
  FieldInput,
  FieldSelect,
  FormField,
  SectionCard,
} from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { useCreateHarvestEntry } from '../hooks/use-create-harvest-entry'
import { useHarvestRates } from '../hooks/use-harvest-rates'
import { entryTotal, validQuantity } from '../lib/entries'
import { activeRate, localDate, money } from '../lib/rates'
import { boxTypes } from '../types'
import { QuantityField } from './quantity-field'
import { RequestError } from './request-error'

type Props = { onOpenEmployees: () => void; onSaved: (day: string) => void }

export function HarvestForm({ onOpenEmployees, onSaved }: Props) {
  const staff = useEmployees(true)
  const rates = useHarvestRates()
  const create = useCreateHarvestEntry()
  const [date, setDate] = useState(() => localDate())
  const [employeeId, setEmployeeId] = useState('')
  const [quantities, setQuantities] = useState({ normal: '', moagem: '' })
  const employee = staff.employees.find(({ id }) => id === employeeId)
  const lines = boxTypes.map((box_type) => ({
    box_type,
    quantity: Number(quantities[box_type]),
    rate: activeRate(rates.data ?? [], box_type, date),
  }))
  const selected = lines.filter(({ quantity }) => quantity !== 0)
  const canSave =
    employee &&
    date &&
    !staff.error &&
    !staff.loading &&
    rates.isSuccess &&
    selected.length > 0 &&
    selected.every(
      ({ quantity, rate }) =>
        validQuantity(quantity) && rate && Number.isSafeInteger(rate.amount_cents),
    )
  const preview = canSave
    ? selected.reduce(
        (sum, line) => sum + BigInt(line.quantity) * BigInt(line.rate!.amount_cents),
        0n,
      )
    : null

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!canSave) return
    const entry = await create.submit({
      employee_id: employeeId,
      harvest_date: date,
      items: selected.map(({ box_type, quantity }) => ({ box_type, quantity })),
    })
    if (entry) {
      setQuantities({ normal: '', moagem: '' })
      onSaved(entry.harvest_date)
    }
  }

  return (
    <SectionCard title="Dados da colheita">
      <form className="demo-form" onSubmit={submit}>
        <fieldset className="form-grid" disabled={create.isPending}>
          <FormField label="Data da colheita">
            <FieldInput
              type="date"
              required
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </FormField>
          <FormField label="Funcionário">
            <FieldSelect
              value={employee?.id ?? ''}
              required
              disabled={!staff.employees.length}
              onChange={(e) => setEmployeeId(e.target.value)}
            >
              <option value="">Selecione um funcionário</option>
              {staff.employees.map(({ id, name }) => (
                <option key={id} value={id}>
                  {name}
                </option>
              ))}
            </FieldSelect>
          </FormField>
          {lines.map(({ box_type, rate }) => (
            <QuantityField
              key={box_type}
              type={box_type}
              value={quantities[box_type]}
              hint={
                rates.isSuccess
                  ? rate
                    ? `${money(rate.amount_cents)} por caixa`
                    : 'Sem tarifa vigente'
                  : undefined
              }
              onChange={(value) => setQuantities({ ...quantities, [box_type]: value })}
            />
          ))}
        </fieldset>
        {staff.loading && <p role="status">Carregando funcionários...</p>}
        {staff.error && (
          <div>
            <p role="alert">{staff.error}</p>
            <Button type="button" onClick={staff.reload}>
              Recarregar funcionários
            </Button>
          </div>
        )}
        {!staff.loading && !staff.error && !staff.employees.length && (
          <div>
            <p>Nenhum funcionário ativo.</p>
            <Button type="button" variant="outline" onClick={onOpenEmployees}>
              Gerenciar funcionários
            </Button>
          </div>
        )}
        {rates.isPending && <p role="status">Carregando tarifas...</p>}
        <RequestError error={rates.error} retry={() => void rates.refetch()} />
        {rates.isSuccess && lines.some(({ rate }) => !rate) && (
          <p>
            Cadastre em Tarifas de colheita a vigência que falta para o tipo de caixa desejado.
          </p>
        )}
        {preview !== null && (
          <p>
            Valor estimado: <strong>{money(preview)}</strong>
          </p>
        )}
        <Button type="submit" disabled={!canSave || create.isPending}>
          Registrar colheita
        </Button>
        <RequestError error={create.error} />
        {create.isSuccess && (
          <Feedback>Colheita registrada. Valor devido: {entryTotal(create.data)}.</Feedback>
        )}
      </form>
    </SectionCard>
  )
}
