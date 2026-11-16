export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const method = (options.method ?? 'GET').toUpperCase()
  if (options.body) headers.set('Content-Type', 'application/json')
  // A API exige este cabeçalho nas escritas para validar requisições do navegador.
  if (!['GET', 'HEAD', 'OPTIONS'].includes(method)) headers.set('X-Agrofin-Request', '1')
  const response = await fetch(`${import.meta.env.VITE_API_PREFIX}${path}`, {
    ...options,
    headers,
    credentials: 'same-origin',
  })
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { error?: string } | null
    throw new ApiError(
      response.status,
      body?.error ?? 'Não foi possível concluir a solicitação.',
    )
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}
