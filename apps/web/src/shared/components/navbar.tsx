import { ArrowRight } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'

type PublicNavbarProps = {
  variant: 'public'
  status: 'loading' | 'anonymous' | 'authenticated' | 'error'
  userName?: string
  onLogin: () => void
  onRegister: () => void
  onOpenPanel: () => void
  onRetry: () => void
}

type PanelNavbarProps = {
  variant: 'panel'
  pageLabel: string
  userName: string
  onLogout: () => void
  leaving: boolean
}

type NavbarProps = PublicNavbarProps | PanelNavbarProps

export function Navbar(props: NavbarProps) {
  if (props.variant === 'panel') {
    return (
      <header className="demo-topbar">
        <div className="mobile-brand">
          <img src="/images/marca/icone-agrofin.png" alt="" width="34" height="34" />
          <strong>agrofin</strong>
        </div>
        <div className="topbar-context">
          <span className="context-dot" /> Fazenda Boa Vista
          <span className="context-divider">/</span>
          <span>{props.pageLabel}</span>
        </div>
        <div className="topbar-right">
          <span className="demo-pill">
            <span /> Modo demonstração
          </span>
          <button
            className="topbar-user"
            onClick={props.onLogout}
            disabled={props.leaving}
            aria-label="Sair da conta"
          >
            {props.userName.charAt(0).toUpperCase()}
          </button>
        </div>
      </header>
    )
  }

  return (
    <header className="landing-header">
      <a className="landing-brand" href="#/" aria-label="Agrofin, início">
        <img
          src="/images/marca/logo-agrofin.png"
          alt="Agrofin"
          width="960"
          height="326"
        />
      </a>
      <nav aria-label="Navegação da apresentação">
        <a href="#recursos">Recursos</a>
        {props.status === 'loading' && <span>Verificando sessão...</span>}
        {props.status === 'error' && (
          <button onClick={props.onRetry}>Tentar novamente</button>
        )}
        {props.status === 'anonymous' && (
          <>
            <button onClick={props.onLogin}>Entrar</button>
            <Button onClick={props.onRegister}>Criar conta</Button>
          </>
        )}
        {props.status === 'authenticated' && (
          <>
            <span className="navbar-user">Olá, {props.userName}</span>
            <Button onClick={props.onOpenPanel}>
              Abrir painel <ArrowRight size={16} />
            </Button>
          </>
        )}
      </nav>
    </header>
  )
}
