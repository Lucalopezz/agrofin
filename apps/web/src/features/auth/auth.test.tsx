import { renderWithQuery } from '@/render-with-query'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SessionProvider } from './session'
import { useSession } from './session-context'
import { AuthPage } from './pages/auth-page'

function response(status: number, data?: unknown) {
  return new Response(data === undefined ? null : JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}
function Status() {
  const session = useSession()
  return (
    <>
      <span>{session.status}</span>
      <span>{session.user?.name}</span>
      <button onClick={() => void session.logout()}>Sair</button>
      <button onClick={() => void session.refresh()}>Atualizar</button>
    </>
  )
}

describe('Autenticação', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('recupera a sessão e encerra após confirmação da API', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        response(200, {
          authenticated: true,
          user: { id: '1', name: 'Ana', login_name: 'ana', active: true },
        }),
      )
      .mockResolvedValueOnce(response(204))
    renderWithQuery(
      <SessionProvider>
        <Status />
      </SessionProvider>,
    )
    expect(await screen.findByText('Ana')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Sair' }))
    expect(await screen.findByText('anonymous')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/v1/auth/logout',
      expect.objectContaining({
        method: 'POST',
        credentials: 'same-origin',
        headers: expect.any(Headers),
      }),
    )
    expect(
      (fetchMock.mock.calls[1][1]!.headers as Headers).get('X-Agrofin-Request'),
    ).toBe('1')
  })

  it('distingue sessão ausente de falha de conexão', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockRejectedValueOnce(new TypeError('offline'))
      .mockResolvedValueOnce(response(401, { error: 'autenticação necessária' }))
    renderWithQuery(
      <SessionProvider>
        <Status />
      </SessionProvider>,
    )
    expect(await screen.findByText('error')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Atualizar' }))
    expect(await screen.findByText('anonymous')).toBeInTheDocument()
  })

  it('informa credenciais inválidas sem navegar', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(response(401))
      .mockResolvedValueOnce(response(401, { error: 'credenciais inválidas' }))
    const navigate = vi.fn()
    renderWithQuery(
      <SessionProvider>
        <AuthPage mode="login" onNavigate={navigate} />
      </SessionProvider>,
    )
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(1))
    fireEvent.change(screen.getByLabelText('Nome de acesso'), {
      target: { value: 'cliente' },
    })
    fireEvent.change(screen.getByLabelText('Senha'), { target: { value: 'incorreta' } })
    fireEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Nome de acesso ou senha inválidos.',
    )
    expect(navigate).not.toHaveBeenCalled()
  })

  it('não envia cadastro quando a confirmação difere', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(response(401))
    renderWithQuery(
      <SessionProvider>
        <AuthPage mode="register" onNavigate={vi.fn()} />
      </SessionProvider>,
    )
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(1))
    fireEvent.change(screen.getByLabelText('Nome'), { target: { value: 'Ana' } })
    fireEvent.change(screen.getByLabelText('Nome de acesso'), {
      target: { value: 'ana' },
    })
    fireEvent.change(screen.getByLabelText('Senha'), {
      target: { value: 'uma senha de teste' },
    })
    fireEvent.change(screen.getByLabelText('Confirmar senha'), {
      target: { value: 'outra senha' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Criar conta' }))
    expect(screen.getByRole('alert')).toHaveTextContent('As senhas não coincidem.')
    expect(globalThis.fetch).toHaveBeenCalledTimes(1)
  })
  it('entra e carrega o perfil antes de abrir o painel', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(response(401))
      .mockResolvedValueOnce(response(200, { authenticated: true }))
      .mockResolvedValueOnce(
        response(200, {
          authenticated: true,
          user: { id: '1', name: 'Ana', login_name: 'ana', active: true },
        }),
      )
    const navigate = vi.fn()
    renderWithQuery(
      <SessionProvider>
        <AuthPage mode="login" onNavigate={navigate} />
        <Status />
      </SessionProvider>,
    )
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))
    fireEvent.change(screen.getByLabelText('Nome de acesso'), {
      target: { value: 'ana' },
    })
    fireEvent.change(screen.getByLabelText('Senha'), {
      target: { value: 'senha de teste segura' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByText('Ana')).toBeInTheDocument()
    expect(navigate).toHaveBeenCalledWith('inicio')
  })

  it('informa conflito no cadastro', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(response(401))
      .mockResolvedValueOnce(response(409, { error: 'nome de acesso já está em uso' }))
    renderWithQuery(
      <SessionProvider>
        <AuthPage mode="register" onNavigate={vi.fn()} />
      </SessionProvider>,
    )
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(1))
    fireEvent.change(screen.getByLabelText('Nome'), { target: { value: 'Ana' } })
    fireEvent.change(screen.getByLabelText('Nome de acesso'), {
      target: { value: 'ana' },
    })
    fireEvent.change(screen.getByLabelText('Senha'), {
      target: { value: 'uma senha de teste' },
    })
    fireEvent.change(screen.getByLabelText('Confirmar senha'), {
      target: { value: 'uma senha de teste' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Criar conta' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Este nome de acesso já está em uso.',
    )
  })
})
