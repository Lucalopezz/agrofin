import { useEffect, useState } from 'react'
import { Button } from '@/shared/components/ui/button'

type Status = 'checking' | 'connected' | 'unavailable'

export function ApiStatus() {
  const [status, setStatus] = useState<Status>('checking')
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    // O prazo evita deixar a verificação presa se o proxy ou a API não responder.
    const timeout = setTimeout(() => controller.abort(), 5000)
    let active = true
    async function check() {
      try {
        const response = await fetch(`${import.meta.env.VITE_API_PREFIX}/health`, { signal: controller.signal })
        const body: unknown = await response.json()
        if (!response.ok || typeof body !== 'object' || body === null || !('status' in body) || body.status !== 'ok') {
          throw new Error('Resposta de saúde inválida')
        }
        if (active) setStatus('connected')
      } catch {
        if (active) setStatus('unavailable')
      } finally {
        clearTimeout(timeout)
      }
    }
    void check()
    // Impede atualizações após desmontar, inclusive na verificação dupla do StrictMode.
    return () => { active = false; clearTimeout(timeout); controller.abort() }
  }, [attempt])

  return (
    <section className="rounded-2xl border bg-secondary p-5" aria-labelledby="conexao-titulo">
      <h2 id="conexao-titulo" className="font-medium">Ambiente de desenvolvimento</h2>
      <p role="status" className="mt-2 text-sm text-muted-foreground">
        {status === 'checking' ? 'Verificando conexão com a API…' : status === 'connected' ? 'Conexão com a API funcionando.' : 'Não foi possível conectar à API. Confira se o serviço está iniciado.'}
      </p>
      <Button className="mt-4" variant="outline" disabled={status === 'checking'} onClick={() => { setStatus('checking'); setAttempt(value => value + 1) }}>Verificar conexão</Button>
    </section>
  )
}
