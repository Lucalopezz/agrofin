import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'
import { weekOf } from '@/features/colheita/lib/entries'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { renderWithQuery } from '@/render-with-query'
import { operationalDate } from '@/shared/lib/period'
import type { EmployeePayroll, PayrollSummary } from './types'

const prefix = '/api/v1'
const period = { start_date: '2026-11-16', end_date: '2026-11-22' }
const employee = { id: 'func-1', name: 'Maria', active: true }
const json = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status })

function payroll(daily = 0, paid = 0): EmployeePayroll {
  return {
    employee, normal_boxes: 10, milling_boxes: 4,
    normal_amount_cents: 5000, milling_amount_cents: 800, harvest_amount_cents: 5800,
    daily_work_amount_cents: daily, due_amount_cents: 5800 + daily,
    paid_amount_cents: paid, remaining_amount_cents: 5800 + daily - paid,
  }
}

function responsePeriod(url: URL) {
  return { start_date: url.searchParams.get('start_date')!, end_date: url.searchParams.get('end_date')! }
}

function openPayroll(read: (url: URL, init: RequestInit) => Response | Promise<Response>) {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (url, init = {}) => {
    const target = new URL(String(url), 'http://localhost')
    if (target.pathname === `${prefix}/auth/me`) return json({ user: { id: 'cliente', name: 'Cliente' } })
    return read(target, init)
  })
  window.history.replaceState(null, '', '#/demo/pagamentos')
  const { queryClient } = renderWithQuery(<SessionProvider><App /></SessionProvider>)
  return { fetchMock, queryClient }
}

async function paymentForm() {
  fireEvent.click(await screen.findByRole('button', { name: 'Pagar' }))
  return within(screen.getByRole('form', { name: 'Registrar pagamento' }))
}

afterEach(() => vi.restoreAllMocks())

it('usa o período aplicado, registra diária e pagamento parcial, mostra a data do servidor e permite quitar', async () => {
  let daily = 0
  let paid = 0
  const { fetchMock, queryClient } = openPayroll((url, init) => {
    switch (url.pathname) {
      case `${prefix}/payroll-periods/summary`:
        return json({ period: responsePeriod(url), employees: [payroll(daily, paid)] } satisfies PayrollSummary)
      case `${prefix}/employees`:
        expect(url.search).toBe('?active=true')
        return json([employee])
      case `${prefix}/harvest-entries`:
        return json([{
          id: 'colheita', employee_id: employee.id, harvest_date: period.start_date,
          items: [{ box_type: 'normal', quantity: 10, applied_rate_cents: 500 }],
        }])
      case `${prefix}/daily-work`:
        if (init.method === 'POST') {
          daily = 8000
          return json({ id: 'diaria', ...JSON.parse(String(init.body)) }, 201)
        }
        return json([{ id: 'diaria', employee_id: employee.id, work_date: period.start_date, amount_cents: daily, notes: 'Irrigação' }])
      case `${prefix}/employee-payments`: {
        const input = JSON.parse(String(init.body))
        paid += input.amount_cents
        return json({ id: 'pagamento', ...input, paid_at: '2026-11-29T02:30:00Z' }, 201)
      }
      default:
        throw new Error(`Rota inesperada: ${url.pathname}`)
    }
  })
  await screen.findByRole('heading', { name: 'Maria' })
  expect(fetchMock).toHaveBeenCalledWith(
    `${prefix}/payroll-periods/summary?${new URLSearchParams(weekOf(operationalDate()))}`, expect.anything(),
  )
  fireEvent.change(screen.getByLabelText('Início'), { target: { value: period.start_date } })
  fireEvent.change(screen.getByLabelText('Fim'), { target: { value: period.end_date } })
  fireEvent.click(screen.getByRole('button', { name: 'Aplicar período' }))
  await screen.findByRole('heading', { name: 'Maria' })

  const report = [...reportsKey('cliente'), 'summary', period.start_date, period.end_date]
  queryClient.setQueryData(report, { example: true })
  fireEvent.click(screen.getByRole('button', { name: 'Registrar diária' }))
  const dailyForm = within(screen.getByRole('form', { name: 'Registrar diária' }))
  await dailyForm.findByRole('option', { name: 'Maria' })
  fireEvent.change(dailyForm.getByLabelText('Funcionário'), { target: { value: employee.id } })
  fireEvent.change(dailyForm.getByLabelText('Data do trabalho'), { target: { value: period.start_date } })
  fireEvent.change(dailyForm.getByLabelText('Valor da diária (R$)'), { target: { value: '80,00' } })
  fireEvent.change(dailyForm.getByLabelText('Observação (opcional)'), { target: { value: 'Irrigação' } })
  fireEvent.click(dailyForm.getByRole('button', { name: 'Salvar diária' }))
  expect(await screen.findByText('Diária registrada.')).toBeVisible()
  expect(screen.getAllByText('R$ 138,00').length).toBeGreaterThan(0)
  expect(queryClient.getQueryState(report)?.isInvalidated).toBe(true)

  fireEvent.click(screen.getByRole('button', { name: 'Ver lançamentos' }))
  expect(await screen.findByText('Irrigação')).toBeVisible()
  expect(await screen.findByText('Normal · 10 caixas a R$ 5,00')).toBeVisible()

  const partial = await paymentForm()
  expect(partial.getByLabelText('Valor do pagamento (R$)')).toHaveValue('138,00')
  fireEvent.change(partial.getByLabelText('Valor do pagamento (R$)'), { target: { value: '100,00' } })
  fireEvent.click(partial.getByRole('button', { name: 'Confirmar pagamento' }))
  expect(await screen.findByText('Pagamento registrado em 28/11/2026.')).toBeVisible()
  expect(fetchMock).toHaveBeenCalledWith(`${prefix}/employee-payments`, expect.objectContaining({
    method: 'POST',
    body: JSON.stringify({ ...period, employee_id: employee.id, amount_cents: 10000, notes: '' }),
  }))

  const full = await paymentForm()
  expect(full.getByLabelText('Valor do pagamento (R$)')).toHaveValue('38,00')
  fireEvent.click(full.getByRole('button', { name: 'Confirmar pagamento' }))
  expect(await screen.findByText('Quitado')).toBeVisible()
})

it('protege contra envio repetido, atualiza saldo em conflito e conserva o rascunho', async () => {
  let paid = 0
  let failPayment!: () => void
  const paymentResponse = new Promise<Response>((resolve) => {
    failPayment = () => {
      paid = 2000
      resolve(json({ error: 'O valor excede o saldo devido no período' }, 409))
    }
  })
  const { fetchMock } = openPayroll((url) => url.pathname === `${prefix}/employee-payments`
    ? paymentResponse
    : json({ period: responsePeriod(url), employees: [payroll(0, paid)] } satisfies PayrollSummary))

  const payment = await paymentForm()
  fireEvent.change(payment.getByLabelText('Valor do pagamento (R$)'), { target: { value: '50,00' } })
  fireEvent.change(payment.getByLabelText('Observação (opcional)'), { target: { value: 'Acerto parcial' } })
  const form = screen.getByRole('form', { name: 'Registrar pagamento' })
  fireEvent.submit(form)
  fireEvent.submit(form)
  await waitFor(() => expect(payment.getByRole('button', { name: 'Confirmar pagamento' })).toBeDisabled())
  failPayment()
  expect(await payment.findByText('O valor excede o saldo devido no período')).toBeVisible()
  expect(screen.getAllByText('R$ 38,00').length).toBeGreaterThan(0)
  expect(payment.getByLabelText('Valor do pagamento (R$)')).toHaveValue('50,00')
  expect(payment.getByLabelText('Observação (opcional)')).toHaveValue('Acerto parcial')
  fireEvent.click(payment.getByRole('button', { name: 'Confirmar pagamento' }))
  expect(payment.getByText('O valor excede o saldo pendente.')).toBeVisible()
  expect(fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
})

it('mantém a lista vazia e oferece a diária quando o período não tem trabalho', async () => {
  openPayroll((url) => json({ period: responsePeriod(url), employees: [] } satisfies PayrollSummary))
  expect(await screen.findByText('Nenhum lançamento de colheita ou diária neste período.')).toBeVisible()
  expect(screen.getByRole('button', { name: 'Registrar diária' })).toBeEnabled()
})
