import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from '@/app/app'
import { SessionProvider } from '@/features/auth/session'
import '@/app/styles.css'

createRoot(document.getElementById('root')!).render(<StrictMode><SessionProvider><App /></SessionProvider></StrictMode>)
