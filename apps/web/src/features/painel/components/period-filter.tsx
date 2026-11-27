import { useState } from 'react'
import { FieldInput, FormField } from './page-ui'
import { Button } from '@/shared/components/ui/button'
import { isPeriod, type Period } from '@/shared/lib/period'

type Props = { period: Period; onApply: (period: Period) => void; disabled?: boolean }

export function PeriodFilter({ period, onApply, disabled }: Props) {
  const [draft, setDraft] = useState(period)
  const [error, setError] = useState(false)

  return (
    <form className="period-filter" onSubmit={(event) => {
      event.preventDefault()
      if (!isPeriod(draft)) {
        setError(true)
        return
      }
      setError(false)
      onApply(draft)
    }}>
      <FormField label="Início">
        <FieldInput type="date" disabled={disabled} value={draft.start_date} required
          onChange={(event) => setDraft({ ...draft, start_date: event.target.value })} />
      </FormField>
      <FormField label="Fim">
        <FieldInput type="date" disabled={disabled} value={draft.end_date} required
          onChange={(event) => setDraft({ ...draft, end_date: event.target.value })} />
      </FormField>
      <Button type="submit" disabled={disabled}>Aplicar período</Button>
      {error && <p className="period-filter-error" role="alert">Informe datas válidas com o fim no mesmo dia ou depois do início.</p>}
    </form>
  )
}
