import type { ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { createQueryClient } from '@/app/query-client'

export function renderWithQuery(ui: ReactNode) {
  const queryClient = createQueryClient()
  return {
    ...render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>),
    queryClient,
  }
}
