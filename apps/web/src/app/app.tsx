import { Leaf } from 'lucide-react'
import { HomePage } from '@/features/inicio/pages/home-page'

export function App() {
  return (
    <div className="min-h-dvh">
      <header className="border-b bg-card">
        <div className="mx-auto flex max-w-4xl items-center gap-3 px-5 py-5">
          <span className="rounded-xl bg-primary p-2 text-primary-foreground"><Leaf aria-hidden="true" size={24} /></span>
          <span className="text-xl font-semibold tracking-tight">agrofin</span>
        </div>
      </header>
      <main id="conteudo" className="mx-auto max-w-4xl px-5 py-10 sm:py-16"><HomePage /></main>
    </div>
  )
}
