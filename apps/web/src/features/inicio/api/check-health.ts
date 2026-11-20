import { apiRequest } from '@/shared/lib/api-client'

export async function checkHealth(signal: AbortSignal) {
  // O prazo permite tentar novamente mesmo quando o proxy não responde.
  const body = await apiRequest<unknown>('/health', {
    signal: AbortSignal.any([signal, AbortSignal.timeout(5000)]),
  })
  if (typeof body !== 'object' || body === null || !('status' in body) || body.status !== 'ok') {
    throw new Error('Resposta de saúde inválida')
  }
  return body.status
}
