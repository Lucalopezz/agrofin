import { useState } from 'react'
import { Feedback, PageHeading, SectionCard } from '@/features/painel/components/page-ui'
import { PeriodFilter } from '@/features/painel/components/period-filter'
import { ExpenseForm } from '../components/expense-form'
import { ExpenseHistory } from '../components/expense-history'
import { useExpenseWrite } from '../hooks/use-expense-write'
import { expenseMonth } from '../lib/expense-period'
import type { ExpenseInput } from '../types'

export function DespesasPage() {
  const [period, setPeriod] = useState(expenseMonth)
  const [notice, setNotice] = useState('')
  const write = useExpenseWrite()

  function confirmWrite(message: string, input: ExpenseInput) {
    // Mantém o filtro aplicado quando a data digitada pertencer a outro período.
    const outside = input.paid_at < period.start_date || input.paid_at > period.end_date
    setNotice(outside ? `${message} A data de pagamento está fora do período exibido.` : message)
  }

  async function create(input: ExpenseInput) {
    setNotice('')
    const saved = await write.submit({ kind: 'create', input })
    if (saved) confirmWrite('Despesa registrada.', input)
    return saved
  }

  return (
    <>
      <PageHeading eyebrow="Financeiro" title="Despesas pagas" description="Registre e confira os gastos pagos." />
      {notice && <Feedback>{notice}</Feedback>}
      <div className="form-page-grid expense-page-grid">
        <SectionCard title="Nova despesa">
          <ExpenseForm busy={write.busy} error={write.errorFor('create')} onSave={create} />
        </SectionCard>
      </div>
      <PeriodFilter
        period={period}
        disabled={write.busy}
        onApply={(next) => {
          setNotice('')
          setPeriod(next)
        }}
      />
      <ExpenseHistory
        key={`${period.start_date}/${period.end_date}`}
        period={period}
        write={write}
        onUpdate={(input) => confirmWrite('Despesa atualizada.', input)}
        onDelete={() => setNotice('Despesa excluída.')}
      />
    </>
  )
}
