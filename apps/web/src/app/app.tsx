import { HomePage } from '@/features/inicio/pages/home-page'

export function App() {
  return (
    <div className="min-h-dvh">
      <header className="border-b bg-card">
        <div className="mx-auto flex max-w-4xl items-center gap-3 px-5 py-5">
          <img src="/images/marca/logo-agrofin.png" alt="agrofin" width={960} height={326} className="h-12 w-auto" />
        </div>
      </header>
      <main id="conteudo" className="mx-auto max-w-4xl px-5 py-10 sm:py-16"><HomePage /></main>
    </div>
  )
}
