import { useState } from 'react'
import { FieldInput, FormField } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import type { Period } from '@/features/colheita/types'
import { isReportPeriod } from '../lib/period'

export function PeriodFilter({ period, onApply }: { period: Period; onApply: (period: Period) => void }) {
  const [draft, setDraft] = useState(period)
  const [error, setError] = useState(false)

  return (
    <form className="report-period-filter" onSubmit={(event) => {
      event.preventDefault()
      if (!isReportPeriod(draft)) {
        setError(true)
        return
      }
      setError(false)
      onApply(draft)
    }}>
      <FormField label="Início">
        <FieldInput type="date" value={draft.start_date} required
          onChange={(event) => setDraft({ ...draft, start_date: event.target.value })} />
      </FormField>
      <FormField label="Fim">
        <FieldInput type="date" value={draft.end_date} required
          onChange={(event) => setDraft({ ...draft, end_date: event.target.value })} />
      </FormField>
      <Button type="submit">Aplicar período</Button>
      {error && <p className="report-period-error" role="alert">Informe datas válidas com o fim no mesmo dia ou depois do início.</p>}
    </form>
  )
}
