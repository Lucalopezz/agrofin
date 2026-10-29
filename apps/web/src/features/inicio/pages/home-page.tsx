import { Leaf } from 'lucide-react'
import { ApiStatus } from '../components/api-status'

export function HomePage() {
  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <p className="text-sm font-medium text-primary">gestão da produção de limão</p>
        <h1 className="max-w-xl text-3xl font-semibold tracking-tight sm:text-4xl">Mais simplicidade na rotina do campo.</h1>
        <p className="max-w-xl leading-relaxed text-muted-foreground">Um lugar para acompanhar a colheita, os pagamentos e os cuidados com a produção.</p>
      </section>
      <section className="rounded-2xl border bg-card p-6 sm:p-8" aria-labelledby="inicio-titulo">
        <Leaf className="mb-4 text-primary" aria-hidden="true" size={28} />
        <h2 id="inicio-titulo" className="text-xl font-semibold">Estamos preparando o terreno</h2>
        <p className="mt-2 max-w-xl leading-relaxed text-muted-foreground">Esta é a primeira versão da interface. Os lançamentos e os resumos da produção estarão disponíveis nas próximas etapas.</p>
      </section>
      {import.meta.env.DEV ? <ApiStatus /> : null}
    </div>
  )
}
