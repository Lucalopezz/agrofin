import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { renderWithQuery } from '@/render-with-query'
import type { Sale } from './types'

const prefix = '/api/v1'
const day = '2026-11-18'
const saleId = 'venda-1'
const json = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status })

const example: Sale = {
  id: saleId,
  sold_at: day,
  buyer_name: 'Barracão São José',
  items: [
    { sale_type: 'exportacao', quantity: 10, unit_price_cents: 4000, line_amount_cents: 40000 },
    { sale_type: 'mercado', quantity: 5, unit_price_cents: 3000, line_amount_cents: 15000 },
  ],
  discount: { amount_cents: 2000, reason: 'Ajuste de qualidade' },
  gross_amount_cents: 55000,
  net_amount_cents: 53000,
  received_amount_cents: 0,
  receivable_amount_cents: 53000,
}

function openSales(read: (url: URL, init: RequestInit) => Response | Promise<Response>) {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (url, init = {}) =>
    read(new URL(String(url), 'http://localhost'), init))
  window.history.replaceState(null, '', '#/demo/vendas')
  const { queryClient } = renderWithQuery(<SessionProvider><App /></SessionProvider>)
  return { fetchMock, queryClient }
}

function session() {
  return json({ user: { id: 'cliente', name: 'Cliente' } })
}

async function saleForm() {
  return within(await screen.findByRole('form', { name: 'Registrar venda' }))
}

function change(form: ReturnType<typeof within>, label: string, value: string) {
  fireEvent.change(form.getByLabelText(label), { target: { value } })
}

afterEach(() => vi.restoreAllMocks())

it('registra venda com desconto, muda para a semana do acerto e baixa o saldo em duas entradas', async () => {
  let registered = false
  let received = 0
  const { fetchMock, queryClient } = openSales((url, init) => {
    if (url.pathname === `${prefix}/auth/me`) return session()
    if (url.pathname === `${prefix}/sales/${saleId}/receipts`) {
      const { amount_cents } = JSON.parse(String(init.body))
      received += amount_cents
      return json({ id: 'entrada-1', sale_id: saleId, amount_cents, received_at: '2026-11-29T02:30:00Z' }, 201)
    }
    if (init.method === 'POST') {
      registered = true
      return json(example, 201)
    }
    const visible = registered && url.searchParams.get('start_date')! <= day && url.searchParams.get('end_date')! >= day
    return json(visible ? [{ ...example, received_amount_cents: received, receivable_amount_cents: 53000 - received }] : [])
  })
  expect(await screen.findByText('Nenhuma venda neste período.')).toBeVisible()
  const report = [...reportsKey('cliente'), 'summary', day, day]
  queryClient.setQueryData(report, { example: true })

  const form = await saleForm()
  change(form, 'Data da venda', day)
  change(form, 'Barracão (opcional)', 'Barracão São José')
  change(form, 'Caixas · Exportação', '10')
  change(form, 'Preço por caixa (R$) · Exportação', '40,00')
  change(form, 'Caixas · Mercado', '5')
  change(form, 'Preço por caixa (R$) · Mercado', '30,00')
  change(form, 'Desconto (R$, opcional)', '20,00')
  change(form, 'Motivo do desconto', 'Ajuste de qualidade')
  expect(form.getByText('R$ 550,00')).toBeVisible()
  expect(form.getByText('R$ 530,00')).toBeVisible()

  fireEvent.click(form.getByRole('button', { name: 'Registrar venda' }))
  expect(await screen.findByText('Venda registrada. Líquido vendido de R$ 530,00.')).toBeVisible()
  expect(await screen.findByText('Motivo do desconto · Ajuste de qualidade')).toBeVisible()
  expect(screen.getByText('Vendida em 18/11/2026')).toBeVisible()
  expect(fetchMock).toHaveBeenCalledWith(`${prefix}/sales`, expect.objectContaining({
    method: 'POST',
    body: JSON.stringify({
      sold_at: day, buyer_name: example.buyer_name, notes: '',
      items: [
        { sale_type: 'exportacao', quantity: 10, unit_price_cents: 4000 },
        { sale_type: 'mercado', quantity: 5, unit_price_cents: 3000 },
      ],
      discount: { amount_cents: 2000, reason: 'Ajuste de qualidade' },
    }),
  }))
  expect(fetchMock).toHaveBeenCalledWith(`${prefix}/sales?start_date=2026-11-16&end_date=2026-11-22`, expect.anything())
  expect(queryClient.getQueryState(report)?.isInvalidated).toBe(true)

  fireEvent.click(screen.getByRole('button', { name: 'Registrar recebimento' }))
  const first = within(screen.getByRole('form', { name: 'Registrar recebimento' }))
  expect(first.getByLabelText('Valor recebido (R$)')).toHaveValue('530,00')
  change(first, 'Valor recebido (R$)', '300,00')
  fireEvent.click(first.getByRole('button', { name: 'Confirmar recebimento' }))
  expect(await screen.findByText('Recebimento registrado em 28/11/2026.')).toBeVisible()
  expect(screen.getByText('R$ 230,00')).toBeVisible()
  expect(fetchMock).toHaveBeenCalledWith(`${prefix}/sales/${saleId}/receipts`, expect.objectContaining({
    method: 'POST', body: JSON.stringify({ amount_cents: 30000 }),
  }))

  fireEvent.click(await screen.findByRole('button', { name: 'Registrar recebimento' }))
  const last = within(screen.getByRole('form', { name: 'Registrar recebimento' }))
  expect(last.getByLabelText('Valor recebido (R$)')).toHaveValue('230,00')
  fireEvent.click(last.getByRole('button', { name: 'Confirmar recebimento' }))
  await waitFor(() => expect(screen.queryByRole('form', { name: 'Registrar recebimento' })).not.toBeInTheDocument())
  expect(screen.queryByRole('button', { name: 'Registrar recebimento' })).not.toBeInTheDocument()
})

it('impede desconto acima do bruto e exige motivo quando há desconto', async () => {
  const { fetchMock } = openSales((url) => url.pathname.endsWith('/auth/me') ? session() : json([]))
  const form = await saleForm()
  change(form, 'Caixas · Exportação', '1')
  change(form, 'Preço por caixa (R$) · Exportação', '20,00')
  change(form, 'Desconto (R$, opcional)', '25,00')
  fireEvent.click(form.getByRole('button', { name: 'Registrar venda' }))
  expect(form.getByText('O desconto não pode exceder o valor bruto.')).toBeVisible()
  change(form, 'Desconto (R$, opcional)', '10,00')
  expect(form.getByText('Informe o motivo do desconto.')).toBeVisible()
  expect(fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(0)
})

it('no conflito de recebimento atualiza o saldo e preserva o valor sem repetir o envio', async () => {
  let received = 0
  let resolveReceipt!: () => void
  const pending = new Promise<Response>((resolve) => {
    resolveReceipt = () => {
      received = 40000
      resolve(json({ error: 'O recebimento excede o saldo' }, 409))
    }
  })
  const { fetchMock } = openSales((url, init) => {
    if (url.pathname.endsWith('/auth/me')) return session()
    if (init.method === 'POST') return pending
    return json([{ ...example, received_amount_cents: received, receivable_amount_cents: 53000 - received }])
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Ver detalhes' }))
  fireEvent.click(screen.getByRole('button', { name: 'Registrar recebimento' }))
  const form = screen.getByRole('form', { name: 'Registrar recebimento' })
  const fields = within(form)
  change(fields, 'Valor recebido (R$)', '300,00')
  fireEvent.submit(form)
  fireEvent.submit(form)
  await waitFor(() => expect(fields.getByRole('button', { name: 'Confirmar recebimento' })).toBeDisabled())
  resolveReceipt()

  expect(await fields.findByText('O recebimento excede o saldo')).toBeVisible()
  expect(screen.getByText('R$ 130,00')).toBeVisible()
  expect(fields.getByLabelText('Valor recebido (R$)')).toHaveValue('300,00')
  fireEvent.click(fields.getByRole('button', { name: 'Confirmar recebimento' }))
  expect(fields.getByText('O valor excede o saldo a receber.')).toBeVisible()
  expect(fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
})

it('volta ao login se a sessão expirar ao consultar vendas', async () => {
  let expired = false
  openSales((url) => {
    if (url.pathname.endsWith('/auth/me') && !expired) return session()
    expired = true
    return json({ error: 'Sessão expirada' }, 401)
  })
  expect(await screen.findByLabelText('Nome de acesso')).toBeVisible()
})
