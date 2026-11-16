import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Navbar } from './navbar'

function publicActions() {
  return { onLogin: vi.fn(), onRegister: vi.fn(), onOpenPanel: vi.fn(), onRetry: vi.fn() }
}

describe('Barra de navegação', () => {
  it('mostra os acessos públicos quando não há sessão', () => {
    const actions = publicActions()
    render(<Navbar variant="public" status="anonymous" {...actions} />)
    fireEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(actions.onLogin).toHaveBeenCalledOnce()
    expect(screen.getByRole('button', { name: 'Criar conta' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Abrir painel' })).not.toBeInTheDocument()
  })

  it('oferece acesso ao painel para o cliente autenticado', () => {
    const actions = publicActions()
    render(<Navbar variant="public" status="authenticated" userName="Ana" {...actions} />)
    expect(screen.getByText('Olá, Ana')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Abrir painel' }))
    expect(actions.onOpenPanel).toHaveBeenCalledOnce()
    expect(screen.queryByRole('button', { name: 'Entrar' })).not.toBeInTheDocument()
  })

  it('mantém a saída e o aviso de demonstração no painel', () => {
    const onLogout = vi.fn()
    render(
      <Navbar
        variant="panel"
        pageLabel="Colheitas"
        userName="Ana"
        onLogout={onLogout}
        leaving={false}
      />,
    )
    expect(screen.getByText('Modo demonstração')).toBeInTheDocument()
    expect(screen.getByText('Colheitas')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Sair da conta' }))
    expect(onLogout).toHaveBeenCalledOnce()
  })
})
