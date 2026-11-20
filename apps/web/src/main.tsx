import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { App } from '@/app/app'
import { createQueryClient } from '@/app/query-client'
import { SessionProvider } from '@/features/auth/session'
import '@/app/styles.css'

const queryClient = createQueryClient()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <App />
      </SessionProvider>
    </QueryClientProvider>
  </StrictMode>,
)
