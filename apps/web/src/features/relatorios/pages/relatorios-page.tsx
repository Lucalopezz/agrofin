import { ChartNoAxesCombined, Download, TrendingUp, Wallet } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import { MetricCard, PageHeading, SectionCard } from '@/features/painel/components/page-ui'

export function RelatoriosPage() {
  function downloadExampleSummary() {
    // A exportação usa as mesmas amostras estáticas desta tela e não lê lançamentos da API.
    const rows = [
      ['Indicador', 'Valor'],
      ['Caixas colhidas', '1284'],
      ['Valor líquido vendido', 'R$ 12.840,00'],
      ['Valor recebido', 'R$ 9.200,00'],
      ['Devido à equipe', 'R$ 6.740,00'],
      ['Pago à equipe', 'R$ 4.320,00'],
      ['Despesas pagas', 'R$ 3.180,00'],
      ['Saldo de caixa do período', 'R$ 1.700,00'],
    ]
    const csv = `\uFEFF${rows.map(row => row.join(';')).join('\n')}`
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
    const link = document.createElement('a')
    link.href = url
    link.download = 'agrofin-resumo-demonstrativo.csv'
    link.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  }

  return <>
    <PageHeading eyebrow="Gestão · Relatórios" title="Visão da produção" description="Resumo do período." action={<Button variant="outline" onClick={downloadExampleSummary}><Download size={16} /> Exportar CSV</Button>} />
    <div className="report-filters"><span>Outubro de 2026</span></div>
    <div className="metric-grid report-metrics">
      <MetricCard label="Caixas colhidas" value="1.284" hint="946 normais · 338 moagem" icon={ChartNoAxesCombined} tone="green" />
      <MetricCard label="Valor vendido" value="R$ 12.840" hint="320 caixas" icon={TrendingUp} tone="blue" />
      <MetricCard label="A pagar à equipe" value="R$ 2.420" hint="R$ 4.320 pagos" icon={Wallet} tone="orange" />
    </div>
    <SectionCard title="Financeiro">
      <div className="financial-summary-list">
        <div className="financial-summary-item"><span className="finance-dot finance-received" /><span>Recebido</span><strong>R$ 9.200</strong></div>
        <div className="financial-summary-item"><span className="finance-dot finance-paid" /><span>Pago à equipe</span><strong>R$ 4.320</strong></div>
        <div className="financial-summary-item"><span className="finance-dot finance-due" /><span>Despesas pagas</span><strong>R$ 3.180</strong></div>
        <div className="cash-balance-item"><span><Wallet size={16} /> Saldo de caixa</span><strong>R$ 1.700</strong><small>Recebido menos saídas pagas. Não é lucro.</small></div>
      </div>
    </SectionCard>
  </>
}
