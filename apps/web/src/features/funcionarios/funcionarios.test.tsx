import { renderWithQuery } from '@/render-with-query'
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'
import type { Employee } from './types'

const prefix = '/api/v1'
const employeeId = 'e0184fc1-b810-4596-8013-bfbccffa7252'
const inactiveId = '84e48b38-b920-43f1-b2f8-b7cbe1ddbe5e'

function response(status: number, data?: unknown) {
  return new Response(data === undefined ? null : JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function sessionResponse() {
  return response(200, {
    authenticated: true,
    user: { id: '1', name: 'Cliente', login_name: 'cliente', active: true },
  })
}

function renderPanel() {
  window.history.replaceState(null, '', '#/demo/funcionarios')
  return renderWithQuery(<SessionProvider><App /></SessionProvider>)
}

describe('Funcionários no painel', () => {
  beforeEach(() => {
    window.history.replaceState(null, '', '#/')
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('exige sessão antes de acessar a rota de funcionários', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(response(401))
    renderPanel()

    expect(await screen.findByRole('button', { name: 'Entrar' })).toBeVisible()
    expect(screen.queryByRole('heading', { name: 'Funcionários' })).not.toBeInTheDocument()
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([`${prefix}/auth/me`])
  })

  it('cadastra, renomeia, desativa e reativa o mesmo funcionário usando os filtros da API', async () => {
    let employee: Employee | undefined
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const path = String(input).slice(prefix.length)
      if (path === '/auth/me') return sessionResponse()
      switch (init?.method) {
        case 'POST':
          employee = { id: employeeId, name: JSON.parse(String(init.body)).name, active: true }
          return response(201, employee)
        case 'PATCH':
          if (!employee) throw new Error('Funcionário ausente no teste')
          if (path.endsWith('/reactivate')) {
            employee.active = true
            return response(204)
          }
          employee.name = JSON.parse(String(init.body)).name
          return response(200, employee)
        case 'DELETE':
          if (!employee) throw new Error('Funcionário ausente no teste')
          employee.active = false
          return response(204)
        default: {
          const filter = new URLSearchParams(path.split('?')[1]).get('active')
          return response(200, employee && (filter === null || String(employee.active) === filter)
            ? [employee]
            : [])
        }
      }
    })
    renderPanel()

    expect(await screen.findByText('Nenhum funcionário encontrado.')).toBeVisible()
    const name = screen.getByRole('textbox', { name: 'Nome' })
    fireEvent.change(name, { target: { value: '   ' } })
    expect(screen.getByRole('button', { name: 'Cadastrar' })).toBeDisabled()
    fireEvent.change(name, { target: { value: '  Maria Silva  ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Cadastrar' }))

    expect(await screen.findByText('Funcionário cadastrado.')).toBeVisible()
    expect(name).toHaveValue('')
    fireEvent.click(await screen.findByRole('button', { name: 'Renomear Maria Silva' }))
    const editName = screen.getByRole('textbox', { name: 'Nome' })
    expect(editName).toHaveValue('Maria Silva')
    expect(editName).toHaveFocus()
    fireEvent.change(editName, { target: { value: '  Maria Souza ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Salvar nome' }))
    expect(await screen.findByText('Nome atualizado.')).toBeVisible()

    const filter = screen.getByRole('combobox', { name: 'Situação' })
    fireEvent.change(filter, { target: { value: 'ativos' } })
    fireEvent.click(await screen.findByRole('button', { name: 'Desativar Maria Souza' }))
    expect(await screen.findByText('Funcionário desativado.')).toBeVisible()
    expect(await screen.findByText('Nenhum funcionário encontrado.')).toBeVisible()

    fireEvent.change(filter, { target: { value: 'inativos' } })
    fireEvent.click(await screen.findByRole('button', { name: 'Reativar Maria Souza' }))
    expect(await screen.findByText('Funcionário reativado.')).toBeVisible()
    expect(await screen.findByText('Nenhum funcionário encontrado.')).toBeVisible()
    fireEvent.change(filter, { target: { value: 'ativos' } })
    expect(await screen.findByRole('button', { name: 'Desativar Maria Souza' })).toBeVisible()

    const writes = fetchMock.mock.calls.filter(([, init]) => init?.method)
    expect(writes.map(([url, init]) => [url, init?.method, init?.body])).toEqual([
      [`${prefix}/employees`, 'POST', JSON.stringify({ name: 'Maria Silva' })],
      [`${prefix}/employees/${employeeId}`, 'PATCH', JSON.stringify({ name: 'Maria Souza' })],
      [`${prefix}/employees/${employeeId}`, 'DELETE', undefined],
      [`${prefix}/employees/${employeeId}/reactivate`, 'PATCH', undefined],
    ])
    for (const [, init] of writes) {
      expect(init?.credentials).toBe('same-origin')
      expect(new Headers(init?.headers).get('X-Agrofin-Request')).toBe('1')
    }
    expect(fetchMock).toHaveBeenCalledWith(`${prefix}/employees?active=true`, expect.anything())
    expect(fetchMock).toHaveBeenCalledWith(`${prefix}/employees?active=false`, expect.anything())
  })

  it('bloqueia envio repetido e preserva o nome quando o cadastro falha', async () => {
    let finishRequest!: (response: Response) => void
    const request = new Promise<Response>((resolve) => { finishRequest = resolve })
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      if (String(input).endsWith('/auth/me')) return sessionResponse()
      return init?.method === 'POST' ? request : response(200, [])
    })
    renderPanel()

    const name = await screen.findByRole('textbox', { name: 'Nome' })
    fireEvent.change(name, { target: { value: 'Maria Silva' } })
    const save = screen.getByRole('button', { name: 'Cadastrar' })
    fireEvent.click(save)
    fireEvent.click(save)

    await waitFor(() => expect(save).toBeDisabled())
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
    finishRequest(response(500, { error: 'não foi possível cadastrar o funcionário' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Não foi possível cadastrar o funcionário')
    expect(name).toHaveValue('Maria Silva')
    expect(save).toBeEnabled()
  })

  it('permite tentar novamente quando a lista falha', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(sessionResponse())
      .mockRejectedValueOnce(new TypeError('offline'))
      .mockResolvedValueOnce(response(200, []))
    renderPanel()

    expect(await screen.findByRole('alert')).toHaveTextContent('Confira a conexão.')
    fireEvent.click(screen.getByRole('button', { name: 'Tentar novamente' }))
    expect(await screen.findByText('Nenhum funcionário encontrado.')).toBeVisible()
  })

  it('reutiliza um filtro consultado enquanto a lista está atual', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(sessionResponse())
      .mockResolvedValueOnce(response(200, [{ id: employeeId, name: 'Maria', active: true }]))
      .mockResolvedValueOnce(response(200, []))
    renderPanel()

    expect(await screen.findByText('Maria')).toBeVisible()
    const filter = screen.getByRole('combobox', { name: 'Situação' })
    fireEvent.change(filter, { target: { value: 'inativos' } })
    expect(await screen.findByText('Nenhum funcionário encontrado.')).toBeVisible()
    fireEvent.change(filter, { target: { value: 'todos' } })

    expect(screen.getByText('Maria')).toBeVisible()
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      `${prefix}/auth/me`,
      `${prefix}/employees`,
      `${prefix}/employees?active=false`,
    ])
  })

  it('ignora uma resposta do filtro anterior ao alternar rapidamente', async () => {
    let finishActive!: (response: Response) => void
    const activeRequest = new Promise<Response>((resolve) => { finishActive = resolve })
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(sessionResponse())
      .mockResolvedValueOnce(response(200, []))
      .mockReturnValueOnce(activeRequest)
      .mockResolvedValueOnce(response(200, [{ id: inactiveId, name: 'Joana', active: false }]))
    renderPanel()

    await screen.findByText('Nenhum funcionário encontrado.')
    const filter = screen.getByRole('combobox', { name: 'Situação' })
    fireEvent.change(filter, { target: { value: 'ativos' } })
    fireEvent.change(filter, { target: { value: 'inativos' } })
    expect(await screen.findByText('Joana')).toBeVisible()
    expect(fetchMock.mock.calls[2]?.[1]?.signal?.aborted).toBe(true)

    await act(async () => {
      finishActive(response(200, [{ id: employeeId, name: 'Maria', active: true }]))
    })
    expect(screen.queryByText('Maria')).not.toBeInTheDocument()
    expect(screen.getByText('Joana')).toBeVisible()
  })

  it.each(['consulta', 'escrita'])('volta ao login quando a sessão expira durante uma %s', async (operation) => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(sessionResponse())
    if (operation === 'escrita') fetchMock.mockResolvedValueOnce(response(200, []))
    fetchMock.mockResolvedValueOnce(response(401)).mockResolvedValueOnce(response(401))
    renderPanel()

    if (operation === 'escrita') {
      await screen.findByText('Nenhum funcionário encontrado.')
      fireEvent.change(screen.getByRole('textbox', { name: 'Nome' }), { target: { value: 'Maria' } })
      fireEvent.click(screen.getByRole('button', { name: 'Cadastrar' }))
    }

    expect(await screen.findByRole('button', { name: 'Entrar' })).toBeVisible()
    expect(fetchMock).toHaveBeenLastCalledWith(`${prefix}/auth/me`, expect.anything())
    expect(screen.queryByRole('heading', { name: 'Funcionários' })).not.toBeInTheDocument()
  })

  it('descarta o cache privado no logout e busca os dados da próxima conta', async () => {
    const otherAccount = response(200, {
      authenticated: true,
      user: { id: '2', name: 'Outra conta', login_name: 'outra', active: true },
    })
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(sessionResponse())
      .mockResolvedValueOnce(response(200, [{ id: employeeId, name: 'Maria', active: true }]))
      .mockResolvedValueOnce(response(204))
      .mockResolvedValueOnce(response(200, { authenticated: true }))
      .mockResolvedValueOnce(otherAccount)
      .mockResolvedValueOnce(response(200, []))
    const { queryClient } = renderPanel()

    expect(await screen.findByText('Maria')).toBeVisible()
    const privateQuery = { queryKey: ['private'] }
    expect(queryClient.getQueriesData(privateQuery)).toHaveLength(1)

    fireEvent.click(screen.getAllByRole('button', { name: /Sair/ })[0]!)
    await screen.findByRole('button', { name: 'Entrar' })
    expect(queryClient.getQueriesData(privateQuery)).toHaveLength(0)

    fireEvent.change(screen.getByLabelText('Nome de acesso'), { target: { value: 'outra' } })
    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'senha-teste' } })
    fireEvent.click(screen.getByRole('button', { name: 'Entrar' }))

    await screen.findByRole('navigation', { name: 'Navegação das áreas' })
    const mobileNav = screen.getByRole('navigation', { name: 'Navegação das áreas' })
    fireEvent.click(within(mobileNav).getByRole('button', { name: 'Funcionários' }))
    expect(await screen.findByText('Nenhum funcionário encontrado.')).toBeVisible()
    expect(screen.queryByText('Maria')).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenLastCalledWith(`${prefix}/employees`, expect.anything())
  })

})
