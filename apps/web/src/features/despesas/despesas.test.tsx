import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { renderWithQuery } from '@/render-with-query'
import { operationalDate } from '@/shared/lib/period'
import { expenseMonth } from './lib/expense-period'
import type { Expense } from './types'

const prefix = '/api/v1'
const paidAt = operationalDate()
const existing: Expense = {
  id: 'despesa-1',
  paid_at: paidAt,
  category: 'Energia e irrigação',
  description: 'Bomba do poço',
  amount_cents: 12345,
}
const json = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status })

function openExpenses(read: (url: URL, request: RequestInit) => Response | Promise<Response>) {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (url, init = {}) => {
    const target = new URL(String(url), 'http://localhost')
    if (target.pathname === `${prefix}/auth/me`) {
      return json({ user: { id: 'cliente', name: 'Cliente' } })
    }
    return read(target, init)
  })
  window.history.replaceState(null, '', '#/demo/despesas')
  const rendered = renderWithQuery(<SessionProvider><App /></SessionProvider>)
  return { fetchMock, queryClient: rendered.queryClient }
}

async function newExpenseForm() {
  return within(await screen.findByRole('form', { name: 'Nova despesa' }))
}

afterEach(() => vi.restoreAllMocks())

it('registra, edita e exclui despesas, atualizando a lista e o relatório da conta', async () => {
  let expenses: Expense[] = []
  const { fetchMock, queryClient } = openExpenses(async (url, init) => {
    switch (init.method) {
      case 'POST':
        expenses = [{ id: existing.id, ...JSON.parse(String(init.body)) }]
        return json(expenses[0], 201)
      case 'PUT':
        expenses = [{ id: existing.id, ...JSON.parse(String(init.body)) }]
        return json(expenses[0])
      case 'DELETE':
        expenses = []
        return new Response(null, { status: 204 })
      default:
        expect(url.pathname).toBe(`${prefix}/expenses`)
        return json(expenses)
    }
  })
  const report = [...reportsKey('cliente'), 'summary', paidAt, paidAt]
  queryClient.setQueryData(report, { fake: 'resumo em cache' })
  expect(await screen.findByText('Nenhuma despesa paga neste período.')).toBeVisible()

  const form = await newExpenseForm()
  expect(form.getByLabelText('Data de pagamento')).toHaveValue(paidAt)
  fireEvent.change(form.getByLabelText('Valor (R$)'), { target: { value: '123,45' } })
  fireEvent.click(form.getByRole('button', { name: 'Registrar despesa' }))

  expect(await screen.findByText('Despesa registrada.')).toBeVisible()
  expect(screen.getByText('Total pago no período · R$ 123,45')).toBeVisible()
  expect(form.getByLabelText('Valor (R$)')).toHaveValue('')
  expect(fetchMock).toHaveBeenCalledWith(`${prefix}/expenses`, expect.objectContaining({
    method: 'POST',
    body: JSON.stringify({ paid_at: paidAt, category: existing.category, description: '', amount_cents: 12345 }),
  }))
  expect(queryClient.getQueryState(report)?.isInvalidated).toBe(true)

  fireEvent.click(screen.getByRole('button', { name: 'Editar' }))
  const edit = within(screen.getByRole('form', { name: 'Editar despesa' }))
  expect(edit.getByLabelText('Valor (R$)')).toHaveValue('123,45')
  fireEvent.change(edit.getByLabelText('Descrição (opcional)'), { target: { value: 'Conta corrigida' } })
  fireEvent.change(edit.getByLabelText('Valor (R$)'), { target: { value: '52,90' } })
  fireEvent.click(edit.getByRole('button', { name: 'Salvar alteração' }))
  expect(await screen.findByText('Despesa atualizada.')).toBeVisible()
  expect(screen.getByText('Conta corrigida')).toBeVisible()
  expect(screen.getByText('Total pago no período · R$ 52,90')).toBeVisible()
  expect(fetchMock).toHaveBeenCalledWith(`${prefix}/expenses/${existing.id}`, expect.objectContaining({
    method: 'PUT',
    body: JSON.stringify({ paid_at: paidAt, category: existing.category, description: 'Conta corrigida', amount_cents: 5290 }),
  }))

  fireEvent.click(screen.getByRole('button', { name: 'Excluir' }))
  expect(fetchMock).not.toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ method: 'DELETE' }))
  fireEvent.click(screen.getByRole('button', { name: 'Confirmar exclusão' }))
  expect(await screen.findByText('Despesa excluída.')).toBeVisible()
  expect(screen.getByText('Nenhuma despesa paga neste período.')).toBeVisible()
  expect(fetchMock).toHaveBeenCalledWith(`${prefix}/expenses/${existing.id}`, expect.objectContaining({ method: 'DELETE' }))
})

it('aplica somente um período válido e consulta as duas datas inclusivas', async () => {
  const { fetchMock } = openExpenses(() => json([]))
  expect(await screen.findByText('Nenhuma despesa paga neste período.')).toBeVisible()
  expect(fetchMock).toHaveBeenCalledWith(
    `${prefix}/expenses?${new URLSearchParams(expenseMonth())}`, expect.anything(),
  )

  fireEvent.change(screen.getByLabelText('Início'), { target: { value: '2030-04-14' } })
  fireEvent.change(screen.getByLabelText('Fim'), { target: { value: '2030-04-13' } })
  fireEvent.click(screen.getByRole('button', { name: 'Aplicar período' }))
  expect(screen.getByRole('alert')).toHaveTextContent('Informe datas válidas')
  expect(fetchMock).toHaveBeenCalledTimes(2)

  fireEvent.change(screen.getByLabelText('Fim'), { target: { value: '2030-04-14' } })
  fireEvent.click(screen.getByRole('button', { name: 'Aplicar período' }))
  await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
    `${prefix}/expenses?start_date=2030-04-14&end_date=2030-04-14`, expect.anything(),
  ))
})

it('rejeita valor inválido e guarda o rascunho após uma falha sem reenviar automaticamente', async () => {
  let resolveWrite!: (response: Response) => void
  const writeResponse = new Promise<Response>((resolve) => { resolveWrite = resolve })
  const { fetchMock } = openExpenses((_, init) => init.method === 'POST' ? writeResponse : json([]))
  const form = await newExpenseForm()
  const amount = form.getByLabelText('Valor (R$)')
  const description = form.getByLabelText('Descrição (opcional)')
  const save = form.getByRole('button', { name: 'Registrar despesa' })

  fireEvent.change(amount, { target: { value: '1,999' } })
  fireEvent.click(save)
  expect(form.getByRole('alert')).toHaveTextContent('Informe um valor positivo')
  expect(fetchMock).not.toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ method: 'POST' }))

  fireEvent.change(amount, { target: { value: '57,10' } })
  fireEvent.change(description, { target: { value: 'Energia do poço' } })
  const formElement = screen.getByRole('form', { name: 'Nova despesa' })
  fireEvent.submit(formElement)
  fireEvent.submit(formElement)
  await waitFor(() => expect(save).toBeDisabled())
  resolveWrite(json({ error: 'Falha ao gravar' }, 500))

  expect(await form.findByRole('alert')).toHaveTextContent('Atualize a lista antes de tentar novamente')
  expect(amount).toHaveValue('57,10')
  expect(description).toHaveValue('Energia do poço')
  const writes = fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')
  expect(writes).toHaveLength(1)
})

it('mostra uma despesa indisponível sem confirmar a exclusão', async () => {
  openExpenses((_, init) => init.method === 'DELETE'
    ? json({ error: 'Despesa não encontrada' }, 404)
    : json([existing]))
  expect(await screen.findByText('Bomba do poço')).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name: 'Excluir' }))
  fireEvent.click(screen.getByRole('button', { name: 'Confirmar exclusão' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Despesa indisponível')
  expect(screen.queryByText('Despesa excluída.')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Atualizar lista' })).toBeEnabled()
})
