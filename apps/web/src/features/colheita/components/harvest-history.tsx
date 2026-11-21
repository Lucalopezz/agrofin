import { useState } from 'react'
import { useEmployees } from '@/features/funcionarios/hooks/use-employees'
import {
  Feedback,
  FieldInput,
  FormField,
  SectionCard,
} from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { useHarvestEntries } from '../hooks/use-harvest-entries'
import { entryTotal } from '../lib/entries'
import { displayDate, money } from '../lib/rates'
import { boxLabels, type Period } from '../types'
import { HarvestEntryEditor } from './harvest-entry-editor'
import { RequestError } from './request-error'

type Props = { period: Period; onChange: (period: Period) => void }

export function HarvestHistory({ period, onChange }: Props) {
  const [editing, setEditing] = useState<string | null>(null)
  const [notice, setNotice] = useState('')
  const entries = useHarvestEntries(period)
  // Inclui inativos para identificar também quem deixou de trabalhar na conta.
  const staff = useEmployees()
  const names = new Map(staff.employees.map(({ id, name }) => [id, name]))
  const editingVisible = editing !== null && entries.data?.some((row) => row.id === editing)
  const valid = period.start_date && period.end_date && period.start_date <= period.end_date

  function changePeriod(next: Period) {
    setEditing(null)
    setNotice('')
    onChange(next)
  }

  return (
    <SectionCard title="Histórico de colheitas">
      {notice && <Feedback>{notice}</Feedback>}
      <div className="form-grid">
        <FormField label="Início do período">
          <FieldInput
            type="date"
            value={period.start_date}
            onChange={(e) => changePeriod({ ...period, start_date: e.target.value })}
          />
        </FormField>
        <FormField label="Fim do período">
          <FieldInput
            type="date"
            value={period.end_date}
            onChange={(e) => changePeriod({ ...period, end_date: e.target.value })}
          />
        </FormField>
      </div>
      <Button
        type="button"
        variant="outline"
        disabled={!valid || entries.isFetching}
        onClick={() => void entries.refetch()}
      >
        Atualizar histórico
      </Button>
      {!valid && <p>Informe um período com fim igual ou posterior ao início.</p>}
      {valid && entries.isPending && <p role="status">Carregando histórico...</p>}
      <RequestError error={entries.error} retry={() => void entries.refetch()} />
      {staff.error && (
        <div>
          <p role="alert">{staff.error}</p>
          <Button onClick={staff.reload}>Recarregar nomes</Button>
        </div>
      )}
      {entries.isSuccess &&
        (entries.data.length ? (
          <ul className="employee-list">
            {entries.data.map((entry) => (
              <li key={entry.id}>
                <div>
                  <strong>
                    {names.get(entry.employee_id) ?? `Funcionário · ${entry.employee_id}`}
                  </strong>
                  <p>{displayDate(entry.harvest_date)}</p>
                  {entry.items.map((item) => (
                    <p key={item.box_type}>
                      {boxLabels[item.box_type]} · {item.quantity} caixas a{' '}
                      {money(item.applied_rate_cents)}
                    </p>
                  ))}
                  <strong>Total devido: {entryTotal(entry)}</strong>
                </div>
                {editing === entry.id ? (
                  <HarvestEntryEditor
                    entry={entry}
                    onCancel={() => setEditing(null)}
                    onSaved={(saved) => {
                      setEditing(null)
                      setNotice(`Quantidades atualizadas. Valor devido: ${entryTotal(saved)}.`)
                    }}
                  />
                ) : (
                  <Button
                    type="button"
                    variant="outline"
                    disabled={editingVisible}
                    onClick={() => {
                      setEditing(entry.id)
                      setNotice('')
                    }}
                  >
                    Corrigir quantidades
                  </Button>
                )}
              </li>
            ))}
          </ul>
        ) : (
          <p>Nenhuma colheita neste período.</p>
        ))}
    </SectionCard>
  )
}
