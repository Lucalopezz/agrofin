import { useState } from 'react'
import { FormActions, SectionCard } from '@/features/painel/components/page-ui'
import { sentenceCase } from '@/features/painel/lib/sentence-case'
import { Button } from '@/shared/components/ui/button'
import { money } from '@/shared/lib/money'
import { displayDate, periodLabel, type Period } from '@/shared/lib/period'
import { useExpenses } from '../hooks/use-expenses'
import type { useExpenseWrite } from '../hooks/use-expense-write'
import type { Expense, ExpenseInput } from '../types'
import { ExpenseForm } from './expense-form'

type Selection = { id: string; kind: 'edit' | 'delete' }

type Props = {
  period: Period
  write: ReturnType<typeof useExpenseWrite>
  onUpdate: (input: ExpenseInput) => void
  onDelete: () => void
}

function totalLabel(expenses: Expense[]) {
  let total = 0n
  for (const expense of expenses) {
    if (!Number.isSafeInteger(expense.amount_cents)) return 'Valor fora do limite de exibição'
    total += BigInt(expense.amount_cents)
  }
  return money(total)
}

export function ExpenseHistory({ period, write, onUpdate, onDelete }: Props) {
  const list = useExpenses(period)
  const [selection, setSelection] = useState<Selection | null>(null)

  function select(action: Selection | null) {
    write.reset()
    setSelection(action)
  }

  async function saveExpense(id: string, input: ExpenseInput) {
    const saved = await write.submit({ kind: 'update', id, input })
    if (saved) {
      setSelection(null)
      onUpdate(input)
    }
    return saved
  }

  async function confirmDelete(id: string) {
    if (await write.submit({ kind: 'delete', id })) {
      setSelection(null)
      onDelete()
    }
  }

  return (
    <SectionCard
      title="Despesas do período"
      subtitle={`Período de ${periodLabel(period)}`}
      action={
        <Button variant="outline" disabled={write.busy || list.isFetching} onClick={() => void list.refetch()}>
          Atualizar lista
        </Button>
      }
    >
      {list.isPending && <p role="status">Carregando despesas…</p>}
      {list.isError && <p role="alert">Não foi possível carregar as despesas do período.</p>}
      {list.isSuccess && list.data.length === 0 && <p role="status">Nenhuma despesa paga neste período.</p>}
      {list.isSuccess && list.data.length > 0 && (
        <>
          <p className="expense-total">Total pago no período · {totalLabel(list.data)}</p>
          <ul className="expense-history" aria-label="Despesas pagas no período">
            {list.data.map((expense) => {
              const action = selection?.id === expense.id ? selection.kind : null
              const deleteError = write.errorFor('delete', expense.id)
              return (
                <li key={expense.id}>
                  <div className="expense-history-heading">
                    <strong>{sentenceCase(expense.category)}</strong>
                    <strong>{money(expense.amount_cents)}</strong>
                  </div>
                  <p>Pago em {displayDate(expense.paid_at)}</p>
                  {expense.description && <p>{expense.description}</p>}
                  {action === 'edit' && (
                    <ExpenseForm
                      expense={expense}
                      busy={write.busy}
                      error={write.errorFor('update', expense.id)}
                      onSave={(input) => saveExpense(expense.id, input)}
                      onCancel={() => select(null)}
                    />
                  )}
                  {action === 'delete' && (
                    <div className="expense-confirm">
                      <p>Excluir esta despesa paga?</p>
                      {deleteError && <p role="alert">{deleteError}</p>}
                      <FormActions>
                        <Button variant="destructive" disabled={write.busy} onClick={() => void confirmDelete(expense.id)}>
                          Confirmar exclusão
                        </Button>
                        <Button variant="outline" disabled={write.busy} onClick={() => select(null)}>Cancelar</Button>
                      </FormActions>
                    </div>
                  )}
                  {action === null && (
                    <FormActions>
                      <Button variant="outline" disabled={write.busy} onClick={() => select({ id: expense.id, kind: 'edit' })}>
                        Editar
                      </Button>
                      <Button variant="outline" disabled={write.busy} onClick={() => select({ id: expense.id, kind: 'delete' })}>
                        Excluir
                      </Button>
                    </FormActions>
                  )}
                </li>
              )
            })}
          </ul>
        </>
      )}
    </SectionCard>
  )
}
