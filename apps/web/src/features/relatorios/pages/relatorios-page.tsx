import { useState } from 'react'
import { Download } from 'lucide-react'
import type { Period } from '@/features/colheita/types'
import { PageHeading } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { PeriodFilter } from '../components/period-filter'
import { ReportSections } from '../components/report-sections'
import { usePeriodSummary } from '../hooks/use-period-summary'
import { exportPeriodSummary } from '../lib/export-summary'
import { periodLabel, reportWeek } from '../lib/period'

type Props = { selectedPeriod?: Period; onApplyPeriod: (period: Period) => void }

export function RelatoriosPage({ selectedPeriod, onApplyPeriod }: Props) {
  const [defaultPeriod] = useState(reportWeek)
  const period = selectedPeriod ?? defaultPeriod
  const query = usePeriodSummary(period)
  const summary = query.isSuccess ? query.data : undefined

  return (
    <>
      <PageHeading
        eyebrow="Relatórios"
        title="Produção e caixa"
        description={`Período de ${periodLabel(period)}`}
        action={summary && (
          <Button variant="outline" onClick={() => exportPeriodSummary(summary)}>
            <Download size={16} aria-hidden="true" /> Exportar CSV
          </Button>
        )}
      />
      <PeriodFilter period={period} onApply={onApplyPeriod} />
      {query.isPending && <p role="status">Carregando resumo…</p>}
      {query.isError && (
        <div role="alert">
          <p>Não foi possível carregar o resumo do período.</p>
          <Button variant="outline" onClick={() => void query.refetch()}>Tentar novamente</Button>
        </div>
      )}
      {summary && <ReportSections summary={summary} />}
    </>
  )
}
