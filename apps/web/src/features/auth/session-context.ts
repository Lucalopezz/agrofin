import { createContext, useContext } from 'react'
import type { Credentials, User } from './api/auth-api'

export type Session = {
  status: 'loading' | 'anonymous' | 'authenticated' | 'error'
  user: User | null
}
type SessionContextValue = Session & {
  refresh: () => Promise<void>
  login: (input: Credentials) => Promise<void>
  logout: () => Promise<void>
}
export const SessionContext = createContext<SessionContextValue | null>(null)

export function useSession() {
  const session = useContext(SessionContext)
  if (!session) throw new Error('SessionProvider ausente')
  return session
}
