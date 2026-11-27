import { useState } from 'react'
import { weekOf } from '@/features/colheita/lib/entries'
import { Feedback, PageHeading } from '@/features/painel/components/page-ui'
import { PeriodFilter } from '@/features/painel/components/period-filter'
import { Button } from '@/shared/components/ui/button'
import { displayDate, operationalDate } from '@/shared/lib/period'
import { DailyWorkForm } from '../components/daily-work-form'
import { PayrollTeam } from '../components/payroll-team'
import { usePayroll } from '../hooks/use-payroll'
import { usePayrollWrite } from '../hooks/use-payroll-write'
import type { DailyWorkInput, PaymentInput } from '../types'

export function PagamentosPage() {
  const [period, setPeriod] = useState(() => weekOf(operationalDate()))
  const [dailyOpen, setDailyOpen] = useState(false)
  const [notice, setNotice] = useState('')
  const query = usePayroll(period)
  const write = usePayrollWrite()

  function toggleDaily(open: boolean) {
    write.reset()
    setDailyOpen(open)
  }

  async function saveDaily(input: DailyWorkInput) {
    setNotice('')
    const result = await write.submit({ kind: 'dailyWork', input })
    if (result?.kind !== 'dailyWork') return
    setDailyOpen(false)
    const day = result.saved.work_date
    const outside = day < period.start_date || day > period.end_date
    setNotice(outside ? 'Diária registrada. A data do trabalho está fora do período exibido.' : 'Diária registrada.')
  }

  async function savePayment(input: PaymentInput) {
    setNotice('')
    const result = await write.submit({ kind: 'payment', input })
    if (result?.kind !== 'payment') return false
    const paidOn = displayDate(operationalDate(new Date(result.saved.paid_at)))
    setNotice(`Pagamento registrado em ${paidOn}.`)
    return true
  }

  return (
    <>
      <PageHeading
        eyebrow="Produção"
        title="Pagamentos"
        description="Confira o que a equipe trabalhou e registre os pagamentos do período."
        action={!dailyOpen && (
          <Button variant="outline" disabled={write.busy} onClick={() => toggleDaily(true)}>
            Registrar diária
          </Button>
        )}
      />
      {notice && <Feedback>{notice}</Feedback>}
      <PeriodFilter
        period={period}
        onApply={(next) => {
          setNotice('')
          write.reset()
          setPeriod(next)
        }}
        disabled={write.busy}
      />
      {dailyOpen && (
        <DailyWorkForm
          period={period}
          busy={write.busy}
          error={write.errorFor('dailyWork')}
          onSave={(input) => void saveDaily(input)}
          onCancel={() => toggleDaily(false)}
        />
      )}
      {query.isPending && <p role="status">Carregando pagamentos…</p>}
      {query.isError && (
        <div role="alert">
          <p>Não foi possível carregar o resumo da equipe.</p>
          <Button variant="outline" onClick={() => void query.refetch()}>Tentar novamente</Button>
        </div>
      )}
      {query.data && (
        <PayrollTeam
          key={`${period.start_date}/${period.end_date}`}
          summary={query.data}
          busy={write.busy}
          canPay={!query.isError}
          error={write.errorFor('payment')}
          onSave={savePayment}
          clearError={write.reset}
        />
      )}
    </>
  )
}
