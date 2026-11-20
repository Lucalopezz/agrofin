import { useEffect, useState } from 'react'
import { useSession } from '@/features/auth/session-context'
import { AuthPage } from '@/features/auth/pages/auth-page'
import { DashboardLayout, type DemoPage } from '@/features/painel/components/dashboard-layout'
import { LandingPage } from '@/features/inicio/pages/landing-page'
import { DashboardPage } from '@/features/painel/pages/dashboard-page'
import { FuncionariosPage } from '@/features/funcionarios/pages/funcionarios-page'
import { ColheitaPage } from '@/features/colheita/pages/colheita-page'
import { PagamentosPage } from '@/features/pagamentos/pages/pagamentos-page'
import { VendasPage } from '@/features/vendas/pages/vendas-page'
import { DespesasPage } from '@/features/despesas/pages/despesas-page'
import { InsumosPage } from '@/features/insumos/pages/insumos-page'
import { PulverizacoesPage } from '@/features/pulverizacoes/pages/pulverizacoes-page'
import { CalendarioPage } from '@/features/calendario/pages/calendario-page'
import { RelatoriosPage } from '@/features/relatorios/pages/relatorios-page'

type Route = DemoPage | 'landing' | 'login' | 'register'

function readRoute(): Route {
  if (window.location.hash === '#/login') return 'login'
  if (window.location.hash === '#/cadastro') return 'register'
  const slug = window.location.hash.replace(/^#\/demo\/?/, '')
  const pages: DemoPage[] = [
    'inicio', 'funcionarios', 'colheitas', 'pagamentos', 'vendas',
    'despesas', 'insumos', 'pulverizacoes', 'calendario', 'relatorios',
  ]
  return pages.includes(slug as DemoPage) ? slug as DemoPage : 'landing'
}

export function App() {
  const [route, setRoute] = useState<Route>(readRoute)
  const session = useSession()
  const [logoutError, setLogoutError] = useState('')
  const [leaving, setLeaving] = useState(false)

  useEffect(() => {
    const syncRoute = () => setRoute(readRoute())
    window.addEventListener('hashchange', syncRoute)
    return () => window.removeEventListener('hashchange', syncRoute)
  }, [])

  function navigate(page: Route) {
    const hash = page === 'landing' ? '#/' : page === 'login' ? '#/login' : page === 'register' ? '#/cadastro' : `#/demo/${page}`
    if (window.location.hash === hash) setRoute(page)
    else window.location.hash = hash
  }

  if (route === 'landing') return <LandingPage onOpenDemo={() => navigate('inicio')} onLogin={() => navigate('login')} onRegister={() => navigate('register')} />
  if (route === 'login' || route === 'register') return <AuthPage key={route} mode={route} onNavigate={navigate} />
  if (session.status === 'loading') return <main className="auth-page"><p>Verificando sessão...</p></main>
  if (session.status === 'error') return <main className="auth-page"><section className="auth-card"><h1>Não foi possível verificar a sessão</h1><p>Confira a conexão e tente novamente.</p><button className="auth-submit" onClick={() => void session.refresh()}>Tentar novamente</button></section></main>
  if (session.status === 'anonymous') return <AuthPage key="login" mode="login" onNavigate={navigate} />

  async function logout() {
    if (leaving) return
    setLeaving(true); setLogoutError('')
    try { await session.logout(); navigate('login') }
    catch { setLogoutError('Não foi possível confirmar a saída. Tente novamente.') }
    finally { setLeaving(false) }
  }

  const pageContent = {
    inicio: <DashboardPage onNavigate={navigate} />,
    funcionarios: <FuncionariosPage key={session.user?.id} />,
    colheitas: <ColheitaPage />,
    pagamentos: <PagamentosPage />,
    vendas: <VendasPage />,
    despesas: <DespesasPage />,
    insumos: <InsumosPage />,
    pulverizacoes: <PulverizacoesPage onNavigate={navigate} />,
    calendario: <CalendarioPage />,
    relatorios: <RelatoriosPage />,
  }[route]

  return <DashboardLayout page={route} onNavigate={navigate} userName={session.user?.name ?? ''} onLogout={() => void logout()} logoutError={logoutError} leaving={leaving}>{pageContent}</DashboardLayout>
}
