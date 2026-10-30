import { useEffect, useState } from 'react'
import { DashboardLayout, type DemoPage } from '@/features/painel/components/dashboard-layout'
import { LandingPage } from '@/features/inicio/pages/landing-page'
import { DashboardPage } from '@/features/painel/pages/dashboard-page'
import { ColheitaPage } from '@/features/colheita/pages/colheita-page'
import { PagamentosPage } from '@/features/pagamentos/pages/pagamentos-page'
import { VendasPage } from '@/features/vendas/pages/vendas-page'
import { DespesasPage } from '@/features/despesas/pages/despesas-page'
import { InsumosPage } from '@/features/insumos/pages/insumos-page'
import { PulverizacoesPage } from '@/features/pulverizacoes/pages/pulverizacoes-page'
import { CalendarioPage } from '@/features/calendario/pages/calendario-page'
import { RelatoriosPage } from '@/features/relatorios/pages/relatorios-page'

type Route = DemoPage | 'landing'

function readRoute(): Route {
  const slug = window.location.hash.replace(/^#\/demo\/?/, '')
  const pages: DemoPage[] = ['inicio', 'colheitas', 'pagamentos', 'vendas', 'despesas', 'insumos', 'pulverizacoes', 'calendario', 'relatorios']
  return pages.includes(slug as DemoPage) ? slug as DemoPage : 'landing'
}

export function App() {
  const [route, setRoute] = useState<Route>(readRoute)

  useEffect(() => {
    const syncRoute = () => setRoute(readRoute())
    window.addEventListener('hashchange', syncRoute)
    return () => window.removeEventListener('hashchange', syncRoute)
  }, [])

  function navigate(page: Route) {
    const hash = page === 'landing' ? '#/' : `#/demo/${page}`
    if (window.location.hash === hash) setRoute(page)
    else window.location.hash = hash
  }

  if (route === 'landing') return <LandingPage onOpenDemo={() => navigate('inicio')} />

  const pageContent = {
    inicio: <DashboardPage onNavigate={navigate} />,
    colheitas: <ColheitaPage />,
    pagamentos: <PagamentosPage />,
    vendas: <VendasPage />,
    despesas: <DespesasPage />,
    insumos: <InsumosPage />,
    pulverizacoes: <PulverizacoesPage onNavigate={navigate} />,
    calendario: <CalendarioPage />,
    relatorios: <RelatoriosPage />,
  }[route]

  return <DashboardLayout page={route} onNavigate={navigate}>{pageContent}</DashboardLayout>
}
