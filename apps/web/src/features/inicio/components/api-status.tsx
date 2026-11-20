import { Button } from '@/shared/components/ui/button'
import { useHealth } from '../hooks/use-health'

export function ApiStatus() {
  const query = useHealth()

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
