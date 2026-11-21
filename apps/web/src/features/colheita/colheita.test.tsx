import { renderWithQuery } from '@/render-with-query'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'

const prefix = '/api/v1'
const employeeId = 'e0184fc1-b810-4596-8013-bfbccffa7252'
const otherEmployeeId = '84e48b38-b920-43f1-b2f8-b7cbe1ddbe5e'

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

function renderHarvest() {
  window.history.replaceState(null, '', '#/demo/colheitas')
  return renderWithQuery(<SessionProvider><App /></SessionProvider>)
}

describe('Funcionários ativos na colheita', () => {
  beforeEach(() => {
    window.history.replaceState(null, '', '#/')
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('usa IDs distintos para funcionários de mesmo nome no seletor de colheita', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(sessionResponse())
      .mockResolvedValueOnce(response(200, [
        { id: employeeId, name: 'Maria Silva', active: true },
        { id: otherEmployeeId, name: 'Maria Silva', active: true },
      ]))
      .mockResolvedValueOnce(response(200, []))
    renderHarvest()

    const select = await screen.findByRole('combobox', { name: 'Funcionário' })
    await waitFor(() => expect(select).toBeEnabled())
    expect(within(select).getAllByRole('option', { name: 'Maria Silva' }).map(
      (option) => (option as HTMLOptionElement).value,
    )).toEqual([employeeId, otherEmployeeId])
    expect(within(select).queryByRole('option', { name: 'Ana Paula' })).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(`${prefix}/employees?active=true`, expect.anything())

    fireEvent.change(screen.getByRole('spinbutton', { name: /Caixas normais/ }), {
      target: { value: '2' },
    })
    const save = screen.getByRole('button', { name: 'Salvar' })
    await waitFor(() => expect(save).toBeDisabled())
    fireEvent.change(select, { target: { value: otherEmployeeId } })
    expect(save).toBeEnabled()
    fireEvent.click(save)
    expect(await screen.findByText(/Lançamento incluído nesta demonstração/)).toBeVisible()
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('oferece acesso ao cadastro quando a colheita não tem funcionários ativos', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(sessionResponse())
      .mockImplementation(async () => response(200, []))
    renderHarvest()

    expect(await screen.findByText('Nenhum funcionário ativo.')).toBeVisible()
    expect(screen.getByRole('combobox', { name: 'Funcionário' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Gerenciar funcionários' }))
    expect(await screen.findByRole('heading', { name: 'Cadastrar funcionário' })).toBeVisible()
  })
})
