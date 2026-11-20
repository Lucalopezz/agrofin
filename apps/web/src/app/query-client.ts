import { QueryClient } from '@tanstack/react-query'

// As operações falham na hora sem rede. Em especial, escritas não ficam
// enfileiradas para rodar depois que o navegador recuperar a conexão.
export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, networkMode: 'always' },
      mutations: { retry: false, networkMode: 'always' },
    },
  })
}

export const privateQueries = ['private'] as const
