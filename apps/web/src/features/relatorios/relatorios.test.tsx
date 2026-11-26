import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'
import { renderWithQuery } from '@/render-with-query'
import { reportWeek } from './lib/period'
import type { PeriodSummary } from './types'

const report: PeriodSummary = {
  start_date: '2026-11-23',
  end_date: '2026-11-29',
  time_zone: 'America/Sao_Paulo',
  harvest: { total_boxes: 0, normal_boxes: 0, milling_boxes: 0, by_employee: [] },
  sales: {
    total_boxes: 0,
    boxes_by_type: { exportacao: 0, mercado: 0, moagem: 0 },
    gross_amount_cents: 0,
    discount_amount_cents: 0,
    net_amount_cents: 0,
    average_net_price_cents: null,
  },
  payroll: { due_amount_cents: 0, paid_amount_cents: 6000, by_employee: [] },
  cash: {
    received_amount_cents: 25000,
    paid_expenses_amount_cents: 19025,
    paid_expenses_by_category: { Energia: 19025 },
    balance_amount_cents: -25,
  },
}

const json = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status })
const filters = '?start_date=2026-11-23&end_date=2026-11-29'

function openPage(hash: string, override?: (path: string) => Response | undefined) {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (url) => {
    const path = String(url).replace('/api/v1', '')
    const result = override?.(path)
    if (result) return result
    if (path === '/auth/me') return json({ user: { id: 'cliente', name: 'Cliente' } })
    if (path.startsWith('/reports/period-summary?')) {
      const params = new URLSearchParams(path.split('?')[1])
      return json({ ...report, ...Object.fromEntries(params) })
    }
    if (path === `/sale-receipts${filters}`) return json([{
      id: 'recebimento-1',
      sale_id: 'venda-1',
      sold_at: '2026-11-19',
      received_at: '2026-11-24T00:30:00Z',
      amount_cents: 25000,
    }])
    throw new Error(`Requisição inesperada: ${path}`)
  })
  window.history.replaceState(null, '', hash)
  renderWithQuery(<SessionProvider><App /></SessionProvider>)
  return fetchMock
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

it('mantém os lançamentos de caixa em uma consulta sob demanda com o período aplicado', async () => {
  const fetchMock = openPage(`#/demo/relatorios${filters}`)
  expect(await screen.findByText('Sem vendas no período')).toBeVisible()
  expect(screen.getByText('-R$ 0,25')).toBeVisible()
  expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining('/sale-receipts'), expect.anything())

  fireEvent.change(screen.getByLabelText('Fim'), { target: { value: '2026-11-30' } })
  fireEvent.click(screen.getByRole('button', { name: 'Ver recebimentos' }))
  expect(await screen.findByText('Venda de 19/11/2026')).toBeVisible()
  expect(screen.getByText(/23\/11\/2026.*21:30/)).toBeVisible()
  expect(fetchMock).toHaveBeenCalledWith(`/api/v1/sale-receipts${filters}`, expect.anything())

  fireEvent.click(screen.getByRole('button', { name: 'Aplicar período' }))
  expect(await screen.findByText('Período de 23/11/2026 a 30/11/2026')).toBeVisible()
  await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
    '/api/v1/reports/period-summary?start_date=2026-11-23&end_date=2026-11-30',
    expect.anything(),
  ))
  expect(screen.queryByText('Venda de 19/11/2026')).not.toBeInTheDocument()
  expect(window.location.hash).toBe('#/demo/relatorios?start_date=2026-11-23&end_date=2026-11-30')
})

it('exporta os totais e as datas retornadas pela API', async () => {
  openPage(`#/demo/relatorios${filters}`)
  const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:resumo')
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  fireEvent.click(await screen.findByRole('button', { name: 'Exportar CSV' }))
  const [csv] = createObjectURL.mock.calls[0]!
  expect(csv).toBeInstanceOf(Blob)
  const raw = await (csv as Blob).text()
  expect(raw).toContain('Início;2026-11-23')
  expect(raw).toContain('Fim;2026-11-29')
  expect(raw).toContain('Saldo de caixa do período;-R$ 0,25')
})

it('leva do início ao relatório com a mesma semana', async () => {
  const fetchMock = openPage('#/demo/inicio')
  expect(await screen.findByText('-R$ 0,25')).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name: 'Ver relatório' }))
  expect(await screen.findByRole('heading', { name: 'Produção e caixa' })).toBeVisible()
  const week = reportWeek()
  expect(screen.getByLabelText('Início')).toHaveValue(week.start_date)
  expect(screen.getByLabelText('Fim')).toHaveValue(week.end_date)
  expect(fetchMock).toHaveBeenCalledWith(
    `/api/v1/reports/period-summary?${new URLSearchParams(week)}`, expect.anything(),
  )
})

it('distingue falha da consulta de uma sessão que expirou', async () => {
  let expired = false
  openPage(`#/demo/relatorios${filters}`, (path) => {
    if (path === '/auth/me' && expired) return json({ error: 'Sessão expirada' }, 401)
    if (!path.startsWith('/reports/period-summary?')) return
    if (expired) return json({ error: 'Sessão expirada' }, 401)
    expired = true
    return json({ error: 'Falha na leitura' }, 500)
  })
  expect(await screen.findByRole('alert')).toHaveTextContent('Não foi possível carregar o resumo')
  expect(screen.queryByText('Nenhum lançamento neste período.')).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Tentar novamente' }))
  expect(await screen.findByLabelText('Nome de acesso')).toBeVisible()
})
