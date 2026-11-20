import { useQuery } from '@tanstack/react-query'
import { Button } from '@/shared/components/ui/button'
import { apiRequest } from '@/shared/lib/api-client'

async function checkHealth(signal: AbortSignal) {
  // O prazo permite tentar novamente mesmo quando o proxy não responde.
  const body = await apiRequest<unknown>('/health', {
    signal: AbortSignal.any([signal, AbortSignal.timeout(5000)]),
  })
  if (typeof body !== 'object' || body === null || !('status' in body) || body.status !== 'ok') {
    throw new Error('Resposta de saúde inválida')
  }
  return body.status
}

export function ApiStatus() {
  const query = useQuery({
    queryKey: ['health'],
    queryFn: ({ signal }) => checkHealth(signal),
  })

  return (
    <section className="rounded-2xl border bg-secondary p-5" aria-labelledby="conexao-titulo">
      <h2 id="conexao-titulo" className="font-medium">Ambiente de desenvolvimento</h2>
      <p role="status" className="mt-2 text-sm text-muted-foreground">
        {query.isFetching
          ? 'Verificando conexão com a API…'
          : query.isSuccess
            ? 'Conexão com a API funcionando.'
            : 'Não foi possível conectar à API. Confira se o serviço está iniciado.'}
      </p>
      <Button
        className="mt-4"
        variant="outline"
        disabled={query.isFetching}
        onClick={() => void query.refetch()}
      >
        Verificar conexão
      </Button>
    </section>
  )
}
