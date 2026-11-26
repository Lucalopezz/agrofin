import { Boxes, ClipboardList, Droplets, Wallet } from 'lucide-react'
import { money } from '@/features/colheita/lib/rates'
import type { Period } from '@/features/colheita/types'
import { usePeriodSummary } from '@/features/relatorios/hooks/use-period-summary'
import { periodLabel, reportWeek } from '@/features/relatorios/lib/period'
import { Button } from '@/shared/components/ui/button'
import type { DemoPage } from '../components/dashboard-layout'
import { MetricCard, PageHeading } from '../components/page-ui'

type DashboardPageProps = {
  userName: string
  onNavigate: (page: DemoPage) => void
  onOpenReport: (period: Period) => void
}

export function DashboardPage({ userName, onNavigate, onOpenReport }: DashboardPageProps) {
  const period = reportWeek()
  const query = usePeriodSummary(period)
  const summary = query.isSuccess ? query.data : undefined

  return (
    <>
      <PageHeading
        eyebrow="Início"
        title={userName ? `Olá, ${userName}` : 'Resumo da semana'}
        description={`Semana de ${periodLabel(period)}`}
        action={<Button variant="outline" onClick={() => onOpenReport(period)}>Ver relatório</Button>}
      />

      {query.isPending && <p role="status">Carregando resumo da semana…</p>}
      {query.isError && (
        <div role="alert">
          <p>Não foi possível carregar o resumo da semana.</p>
          <Button variant="outline" onClick={() => void query.refetch()}>Tentar novamente</Button>
        </div>
      )}
      {summary && (
        <div className="metric-grid dashboard-metrics">
          <MetricCard label="Caixas colhidas" value={summary.harvest.total_boxes.toLocaleString('pt-BR')}
            hint="Normais e moagem" icon={Boxes} tone="green" />
          <MetricCard label="Devido à equipe" value={money(summary.payroll.due_amount_cents)}
            hint="Caixas e diárias da semana" icon={Wallet} tone="orange" />
          <MetricCard label="Saldo de caixa" value={money(summary.cash.balance_amount_cents)}
            hint="Recebimentos menos saídas pagas" icon={Wallet} tone="blue" />
        </div>
      )}

      <section className="quick-actions-section" aria-labelledby="quick-actions-title">
        <div className="subsection-heading"><h2 id="quick-actions-title">Ações rápidas</h2></div>
        <div className="quick-actions-grid">
          <button className="quick-action-card harvest-action" onClick={() => onNavigate('colheitas')}>
            <span className="quick-icon"><ClipboardList size={19} aria-hidden="true" /></span>
            <strong>Registrar caixas</strong><span className="quick-plus">+</span>
          </button>
          <button className="quick-action-card payment-action" onClick={() => onNavigate('pagamentos')}>
            <span className="quick-icon"><Wallet size={19} aria-hidden="true" /></span>
            <strong>Pagar equipe</strong><span className="quick-plus">+</span>
          </button>
          <button className="quick-action-card spray-action" onClick={() => onNavigate('pulverizacoes')}>
            <span className="quick-icon"><Droplets size={19} aria-hidden="true" /></span>
            <strong>Registrar pulverização</strong><span className="quick-plus">+</span>
          </button>
        </div>
      </section>
    </>
  )
}
