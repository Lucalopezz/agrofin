import { entryTotal } from '@/features/colheita/lib/entries'
import { boxLabels } from '@/features/colheita/types'
import { useReportDetails } from '@/features/relatorios/hooks/use-period-summary'
import { Button } from '@/shared/components/ui/button'
import { money } from '@/shared/lib/money'
import { displayDate, type Period } from '@/shared/lib/period'

export function PayrollDetails({ employeeId, period }: { employeeId: string; period: Period }) {
  const harvest = useReportDetails('harvest', period)
  const daily = useReportDetails('dailyWork', period)
  const harvestEntries = harvest.data?.filter((row) => row.employee_id === employeeId) ?? []
  const dailyEntries = daily.data?.filter((row) => row.employee_id === employeeId) ?? []

  return (
    <div className="payroll-entry-form">
      <h4>Colheitas do funcionário</h4>
      {harvest.isPending && <p role="status">Carregando colheitas…</p>}
      {harvest.isError && <DetailError kind="colheitas" retry={() => void harvest.refetch()} />}
      {harvest.isSuccess && (harvestEntries.length ? (
        <ul className="report-list">
          {harvestEntries.map((row) => (
            <li key={row.id}>
              <strong>{displayDate(row.harvest_date)} · {entryTotal(row)}</strong>
              {row.items.map((item) => (
                <span key={item.box_type}>{boxLabels[item.box_type]} · {item.quantity} caixas a {money(item.applied_rate_cents)}</span>
              ))}
            </li>
          ))}
        </ul>
      ) : <p>Nenhuma colheita no período.</p>)}

      <h4>Diárias do funcionário</h4>
      {daily.isPending && <p role="status">Carregando diárias…</p>}
      {daily.isError && <DetailError kind="diárias" retry={() => void daily.refetch()} />}
      {daily.isSuccess && (dailyEntries.length ? (
        <ul className="report-list">
          {dailyEntries.map((row) => (
            <li key={row.id}>
              <strong>{displayDate(row.work_date)} · {money(row.amount_cents)}</strong>
              {row.notes && <span>{row.notes}</span>}
            </li>
          ))}
        </ul>
      ) : <p>Nenhuma diária no período.</p>)}
    </div>
  )
}

function DetailError({ kind, retry }: { kind: string; retry: () => void }) {
  return (
    <>
      <p role="alert">Não foi possível carregar as {kind}.</p>
      <Button type="button" variant="outline" onClick={retry}>Tentar novamente</Button>
    </>
  )
}
