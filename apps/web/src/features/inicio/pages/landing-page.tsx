import { ArrowRight, ClipboardList, Droplets, Leaf, Wallet } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'

type LandingPageProps = { onOpenDemo: () => void }

const areas = [
  { icon: ClipboardList, title: 'Produção', text: 'Colheitas e pagamentos da equipe.' },
  { icon: Wallet, title: 'Financeiro', text: 'Vendas, recebimentos e despesas.' },
  { icon: Droplets, title: 'Manejo', text: 'Insumos, pulverizações e calendário.' },
]

export function LandingPage({ onOpenDemo }: LandingPageProps) {
  return (
    <div className="landing-page">
      <header className="landing-header">
        <a className="landing-brand" href="#/" aria-label="Agrofin, início"><img src="/images/marca/logo-agrofin.png" alt="Agrofin" width="960" height="326" /></a>
        <nav aria-label="Navegação da apresentação"><a href="#recursos">Recursos</a><Button onClick={onOpenDemo}>Ver demonstração <ArrowRight size={16} /></Button></nav>
      </header>

      <main>
        <section className="landing-hero">
          <div className="hero-copy">
            <span className="hero-kicker"><Leaf size={15} aria-hidden="true" /> Gestão do pomar</span>
            <h1>A rotina do pomar,<br /><em>mais simples.</em></h1>
            <p>Colheitas, pagamentos e manejo em um só lugar.</p>
            <div className="hero-actions"><Button size="lg" onClick={onOpenDemo}>Explorar painel <ArrowRight size={17} /></Button></div>
            <small className="landing-demo-note">Demonstração com dados fictícios.</small>
          </div>
          <div className="hero-art" role="img" aria-label="Pomar de limão">
            <div className="hero-photo" />
            <div className="hero-photo-shade" />
            <div className="hero-photo-caption"><span className="caption-icon"><Leaf size={17} /></span><span><strong>Fazenda Boa Vista</strong><small>Produção de limão</small></span></div>
          </div>
        </section>

        <section className="feature-section" id="recursos" aria-labelledby="areas-title">
          <div className="feature-heading"><div><span className="section-kicker">Tudo conectado</span><h2 id="areas-title">Do pomar ao caixa.</h2></div></div>
          <div className="feature-grid">{areas.map(({ icon: Icon, title, text }) => <article className="feature-card" key={title}><span className="feature-icon"><Icon size={21} strokeWidth={1.8} aria-hidden="true" /></span><h3>{title}</h3><p>{text}</p></article>)}</div>
        </section>

        <section className="landing-cta"><div><span className="section-kicker">Agrofin</span><h2>Conheça a demonstração.</h2></div><Button size="lg" onClick={onOpenDemo}>Abrir painel <ArrowRight size={17} /></Button></section>
      </main>
      <footer className="landing-footer"><a className="landing-brand" href="#/" aria-label="Agrofin, início"><img src="/images/marca/logo-agrofin.png" alt="Agrofin" width="960" height="326" /></a><span>Gestão da produção de limão</span></footer>
    </div>
  )
}
