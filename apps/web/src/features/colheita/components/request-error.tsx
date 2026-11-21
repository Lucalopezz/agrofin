import { ApiError } from '@/shared/lib/api-client'
import { Button } from '@/shared/components/ui/button'

export function RequestError({ error, retry }: { error: Error | null; retry?: () => void }) {
  if (!error) return null
  const fallback = retry
    ? 'Não foi possível carregar os dados. Confira a conexão.'
    : 'Não foi possível confirmar a alteração. Consulte os registros antes de tentar novamente.'
  return (
    <div>
      <p role="alert">{error instanceof ApiError ? error.message : fallback}</p>
      {retry && <Button type="button" variant="outline" onClick={retry}>Tentar novamente</Button>}
    </div>
  )
}
