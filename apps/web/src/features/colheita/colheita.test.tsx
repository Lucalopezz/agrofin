import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'
import { renderWithQuery } from '@/render-with-query'
import { localDate } from './lib/rates'
import { weekOf } from './lib/entries'
import type { HarvestEntry, Rate } from './types'

const today = localDate()
const person = { id: 'employee-1', name: 'Maria Silva', active: true }
const rate: Rate = {
  id: 'rate', box_type: 'normal', amount_cents: 501, valid_from: '2000-01-01', valid_until: null,
}
const entry: HarvestEntry = {
  id: 'entry', employee_id: person.id, harvest_date: today,
  items: [{ box_type: 'normal', quantity: 2, applied_rate_cents: 500 }],
}
const json = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status })
type Override = (path: string, init: RequestInit) => Response | Promise<Response> | undefined

function openPage(override?: Override) {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (url, init = {}) => {
    const path = String(url).replace('/api/v1', '')
    const result = override?.(path, init)
    if (result) return result
    if (path === '/auth/me') return json({ user: { id: 'cliente', name: 'Cliente' } })
    if (path.startsWith('/employees')) return json([person])
    if (path === '/harvest-rates' && !init.method) return json([rate])
    if (path.startsWith('/harvest-entries?')) return json([])
    throw new Error(`Requisição inesperada: ${path}`)
  })
  window.history.replaceState(null, '', '#/demo/colheitas')
  renderWithQuery(<SessionProvider><App /></SessionProvider>)
  return fetchMock
}

function input(label: string | RegExp, value: string) {
  fireEvent.change(screen.getByLabelText(label), { target: { value } })
}

async function fillEntry(id = person.id) {
  await screen.findByRole('option', { name: person.name })
  input('Funcionário', id)
  input(/Caixas normais/, '2')
}

afterEach(() => vi.restoreAllMocks())

describe('Tarifas e colheitas', () => {
  it('abre o cadastro de funcionários a partir da lista vazia', async () => {
    openPage((path) => path.startsWith('/employees') ? json([]) : undefined)
    expect(await screen.findByText('Nenhum funcionário ativo.')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Gerenciar funcionários' }))
    expect(await screen.findByRole('heading', { name: 'Cadastrar funcionário' })).toBeVisible()
  })

  it('bloqueia o lançamento apenas quando falta tarifa para um tipo informado', async () => {
    openPage()
    await fillEntry()
    const save = screen.getByRole('button', { name: 'Registrar colheita' })
    await waitFor(() => expect(save).toBeEnabled())
    input(/Caixas de moagem/, '1')
    expect(save).toBeDisabled()
    input(/Caixas de moagem/, '')
    expect(save).toBeEnabled()
    input('Data da colheita', '1999-12-31')
    expect(save).toBeDisabled()
  })

  it('permite configurar a conta vazia e usa a tarifa confirmada na prévia', async () => {
    let stored: Rate[] = []
    const fetchMock = openPage((path, init) => {
      if (path !== '/harvest-rates') return
      if (!init.method) return json(stored)
      stored = [{ ...JSON.parse(String(init.body)), id: 'new' }]
      return json(stored[0], 201)
    })
    await fillEntry()
    expect(screen.getByRole('button', { name: 'Registrar colheita' })).toBeDisabled()
    fireEvent.click(screen.getByText('Consultar e cadastrar tarifas'))
    input('Valor por caixa (R$)', '4,05')
    input('Início da vigência', '2020-01-01')
    input(/Fim da vigência/, '2019-12-31')
    const submit = screen.getByRole('button', { name: 'Cadastrar tarifa' })
    expect(submit).toBeDisabled()
    input(/Fim da vigência/, '')
    fireEvent.click(submit)
    expect(await screen.findByText('Tarifa cadastrada.')).toBeVisible()
    expect(screen.getByText('R$ 8,10')).toBeVisible()
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/harvest-rates', expect.objectContaining({
      method: 'POST', body: JSON.stringify({
        box_type: 'normal', amount_cents: 405, valid_from: '2020-01-01', valid_until: null,
      }),
    }))
  })

  it('mantém a tarifa digitada quando o servidor rejeita uma sobreposição', async () => {
    openPage((path, init) => path === '/harvest-rates' && init.method === 'POST'
      ? json({ error: 'Vigência sobreposta' }, 409) : undefined)
    fireEvent.click(await screen.findByText('Consultar e cadastrar tarifas'))
    input('Tipo de caixa', 'moagem')
    input('Valor por caixa (R$)', '3,50')
    fireEvent.click(screen.getByRole('button', { name: 'Cadastrar tarifa' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Vigência sobreposta')
    expect(screen.getByLabelText('Valor por caixa (R$)')).toHaveValue('3,50')
  })

  it('envia uma vez com o ID escolhido e exibe a tarifa aplicada pelo servidor', async () => {
    let confirm!: (value: Response) => void
    const pending = new Promise<Response>((resolve) => { confirm = resolve })
    let stored: HarvestEntry[] = []
    const twin = { ...person, id: 'employee-2' }
    const fetchMock = openPage((path, init) => {
      if (path.startsWith('/employees')) return json([person, twin])
      if (path === '/harvest-entries' && init.method === 'POST') return pending
      if (path.startsWith('/harvest-entries?')) return json(stored)
    })
    const select = await screen.findByRole('combobox', { name: 'Funcionário' })
    await waitFor(() => expect(within(select).getAllByText(person.name)).toHaveLength(2))
    input('Funcionário', twin.id)
    input(/Caixas normais/, '2')
    const save = screen.getByRole('button', { name: 'Registrar colheita' })
    fireEvent.submit(save.closest('form')!)
    fireEvent.submit(save.closest('form')!)
    await waitFor(() => expect(save).toBeDisabled())
    const writes = fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')
    expect(writes).toHaveLength(1)
    expect(JSON.parse(String(writes[0][1]?.body))).toEqual({
      employee_id: twin.id, harvest_date: today, items: [{ box_type: 'normal', quantity: 2 }],
    })
    stored = [{ ...entry, employee_id: twin.id }]
    confirm(json(stored[0], 201))
    expect(await screen.findByText('Colheita registrada. Valor devido: R$ 10,00.')).toBeVisible()
    expect(screen.getByText('Total devido: R$ 10,00')).toBeVisible()
    expect(screen.getByLabelText('Data da colheita')).toHaveValue(today)
    expect(screen.getByLabelText(/Caixas normais/)).toHaveValue(null)
  })

  it('mantém o rascunho quando a API recusa a colheita', async () => {
    openPage((path, init) => path === '/harvest-entries' && init.method === 'POST'
      ? json({ error: 'Não há tarifa vigente para a data informada' }, 409) : undefined)
    await fillEntry()
    fireEvent.click(screen.getByRole('button', { name: 'Registrar colheita' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Não há tarifa vigente')
    expect(screen.getByLabelText(/Caixas normais/)).toHaveValue(2)
  })

  it('consulta o histórico por período e permite recuperar uma falha', async () => {
    const week = weekOf(today)
    let failed = true
    const fetchMock = openPage((path) => {
      if (!path.startsWith('/harvest-entries?')) return
      if (failed) return json({ error: 'Histórico indisponível' }, 500)
      return json([entry])
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('Histórico indisponível')
    failed = false
    fireEvent.click(screen.getByRole('button', { name: 'Tentar novamente' }))
    expect(await screen.findByText('Total devido: R$ 10,00')).toBeVisible()
    expect(fetchMock).toHaveBeenCalledWith(
      `/api/v1/harvest-entries?start_date=${week.start_date}&end_date=${week.end_date}`, expect.anything(),
    )
    input('Fim do período', '2020-01-10')
    expect(screen.getByText(/Informe um período/)).toBeVisible()
    input('Início do período', '2020-01-01')
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/harvest-entries?start_date=2020-01-01&end_date=2020-01-10', expect.anything(),
    ))
  })

  it('pede login se a sessão expirar durante uma escrita', async () => {
    let expired = false
    openPage((path, init) => {
      if (path === '/harvest-entries' && init.method === 'POST') expired = true
      if (expired) return json({ error: 'Sessão expirada' }, 401)
    })
    await fillEntry()
    fireEvent.click(screen.getByRole('button', { name: 'Registrar colheita' }))
    expect(await screen.findByRole('button', { name: 'Entrar' })).toBeVisible()
  })
})
