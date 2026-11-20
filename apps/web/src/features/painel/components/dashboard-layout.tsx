import type { ReactNode } from 'react'
import {
  CalendarDays,
  ChartNoAxesCombined,
  ClipboardList,
  Droplets,
  LayoutDashboard,
  Leaf,
  Package,
  ReceiptText,
  Store,
  Users,
  Wallet,
} from 'lucide-react'
import { Navbar } from '@/shared/components/navbar'
import { sentenceCase } from '@/features/painel/lib/sentence-case'

export type DemoPage =
  | 'inicio'
  | 'colheitas'
  | 'funcionarios'
  | 'pagamentos'
  | 'vendas'
  | 'despesas'
  | 'insumos'
  | 'pulverizacoes'
  | 'calendario'
  | 'relatorios'

const navigation: { id: DemoPage; label: string; icon: typeof Leaf; group: string }[] = [
  { id: 'inicio', label: 'Visão geral', icon: LayoutDashboard, group: 'produção' },
  { id: 'funcionarios', label: 'Funcionários', icon: Users, group: 'produção' },
  { id: 'colheitas', label: 'Colheitas', icon: ClipboardList, group: 'produção' },
  { id: 'pagamentos', label: 'Pagamentos', icon: Wallet, group: 'produção' },
  { id: 'vendas', label: 'Vendas', icon: Store, group: 'financeiro' },
  { id: 'despesas', label: 'Despesas', icon: ReceiptText, group: 'financeiro' },
  { id: 'insumos', label: 'Insumos', icon: Package, group: 'manejo' },
  { id: 'pulverizacoes', label: 'Pulverizações', icon: Droplets, group: 'manejo' },
  { id: 'calendario', label: 'Calendário', icon: CalendarDays, group: 'manejo' },
  { id: 'relatorios', label: 'Relatórios', icon: ChartNoAxesCombined, group: 'gestão' },
]

type DashboardLayoutProps = {
  page: DemoPage
  onNavigate: (page: DemoPage | 'landing') => void
  children: ReactNode
  userName: string
  onLogout: () => void
  logoutError: string
  leaving: boolean
}

export function DashboardLayout({
  page,
  onNavigate,
  children,
  userName,
  onLogout,
  logoutError,
  leaving,
}: DashboardLayoutProps) {
  const groups = [...new Set(navigation.map((item) => item.group))]
  const active = navigation.find((item) => item.id === page)

  return (
    <div className="demo-shell">
      <aside className="demo-sidebar" aria-label="Navegação principal">
        <button
          className="brand-lockup"
          onClick={() => onNavigate('inicio')}
          aria-label="Ir para visão geral"
        >
          <img src="/images/marca/icone-agrofin.png" alt="" width="46" height="46" />
          <span>
            <strong>agrofin</strong>
            <small>{sentenceCase('gestão do pomar')}</small>
          </span>
        </button>
        <div className="sidebar-farm">
          <span className="farm-avatar">
            <Leaf size={17} aria-hidden="true" />
          </span>
          <span>
            <strong>Fazenda Boa Vista</strong>
            <small>{sentenceCase('produção de limão')}</small>
          </span>
          <span className="online-dot" title="Demonstração" />
        </div>
        <nav className="side-navigation">
          {groups.map((group) => (
            <div className="nav-group" key={group}>
              <p>{sentenceCase(group)}</p>
              {navigation
                .filter((item) => item.group === group)
                .map((item) => {
                  const Icon = item.icon
                  return (
                    <button
                      key={item.id}
                      className={`nav-link ${page === item.id ? 'is-active' : ''}`}
                      onClick={() => onNavigate(item.id)}
                      aria-current={page === item.id ? 'page' : undefined}
                    >
                      <Icon size={18} strokeWidth={1.8} aria-hidden="true" />
                      <span>{sentenceCase(item.label)}</span>
                    </button>
                  )
                })}
            </div>
          ))}
        </nav>
        <div className="sidebar-help">
          <span className="help-icon">
            <Leaf size={17} aria-hidden="true" />
          </span>
          <strong>Feito para a rotina do campo</strong>
          <p>Registros simples para acompanhar cada etapa da produção.</p>
        </div>
        <button className="sidebar-profile" onClick={onLogout} disabled={leaving}>
          <span className="profile-avatar">{userName.charAt(0).toUpperCase()}</span>
          <span>
            <strong>{userName}</strong>
            <small>{leaving ? 'Saindo...' : 'Sair'}</small>
          </span>
        </button>
      </aside>

      <div className="demo-main-column">
        {logoutError && (
          <p className="auth-error" role="alert">
            {logoutError}
          </p>
        )}
        <Navbar
          variant="panel"
          pageLabel={active ? sentenceCase(active.label) : ''}
          userName={userName}
          onLogout={onLogout}
          leaving={leaving}
        />
        <nav className="mobile-navigation" aria-label="Navegação das áreas">
          {navigation.map((item) => {
            const Icon = item.icon
            return (
              <button
                key={item.id}
                className={page === item.id ? 'is-active' : ''}
                onClick={() => onNavigate(item.id)}
                aria-current={page === item.id ? 'page' : undefined}
              >
                <Icon size={17} aria-hidden="true" />
                {sentenceCase(item.label)}
              </button>
            )
          })}
        </nav>
        <main className="demo-content" id="conteudo">
          {children}
        </main>
        <footer className="demo-footer">
          <span>Demonstração agrofin · interface</span>
          <button onClick={() => onNavigate('landing')}>Voltar à apresentação</button>
        </footer>
      </div>
    </div>
  )
}
