import { Boxes, ClipboardList, Droplets, Wallet } from 'lucide-react'
import type { DemoPage } from '../components/dashboard-layout'
import { MetricCard, PageHeading, SectionCard } from '../components/page-ui'

type DashboardPageProps = { onNavigate: (page: DemoPage) => void }

const recentActivity = [
  { label: 'Colheita', detail: 'Ana Paula · 24 caixas', time: 'Hoje' },
  { label: 'Venda', detail: 'Barracão Santa Luzia · R$ 2.860', time: 'Ontem' },
]

export function DashboardPage({ onNavigate }: DashboardPageProps) {
  return <>
    <PageHeading eyebrow="Sexta-feira, 30 de outubro" title="Bom dia, Marcos" description="Resumo da sua semana." />

    <div className="metric-grid dashboard-metrics">
      <MetricCard label="Caixas colhidas" value="386" hint="Esta semana" icon={Boxes} tone="green" />
      <MetricCard label="A pagar à equipe" value="R$ 2.180" hint="3 pessoas pendentes" icon={Wallet} tone="orange" />
      <MetricCard label="A receber" value="R$ 2.260" hint="Vendas do período" icon={Wallet} tone="blue" />
    </div>

    <section className="quick-actions-section" aria-labelledby="quick-actions-title">
      <div className="subsection-heading"><h2 id="quick-actions-title">Ações rápidas</h2></div>
      <div className="quick-actions-grid">
        <button className="quick-action-card harvest-action" onClick={() => onNavigate('colheitas')}><span className="quick-icon"><ClipboardList size={19} aria-hidden="true" /></span><strong>Registrar caixas</strong><span className="quick-plus">+</span></button>
        <button className="quick-action-card payment-action" onClick={() => onNavigate('pagamentos')}><span className="quick-icon"><Wallet size={19} aria-hidden="true" /></span><strong>Pagar equipe</strong><span className="quick-plus">+</span></button>
        <button className="quick-action-card spray-action" onClick={() => onNavigate('pulverizacoes')}><span className="quick-icon"><Droplets size={19} aria-hidden="true" /></span><strong>Registrar pulverização</strong><span className="quick-plus">+</span></button>
      </div>
    </section>

    <SectionCard title="Atividade recente">
      <div className="activity-list">{recentActivity.map(item => <div className="activity-row" key={item.label}><span className="activity-dot" /><div className="activity-copy"><strong>{item.label}</strong><small>{item.detail}</small></div><time>{item.time}</time></div>)}</div>
    </SectionCard>
  </>
}
