import { apiRequest } from '@/shared/lib/api-client'

export type User = {
  id: string
  name: string
  login_name: string
  active: boolean
}

export type Credentials = {
  login_name: string
  password: string
}

export type Registration = Credentials & {
  name: string
}

export const authApi = {
  register: (input: Registration) =>
    apiRequest<{ registered: boolean }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  login: (input: Credentials) =>
    apiRequest<{ authenticated: boolean }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  me: () => apiRequest<{ authenticated: boolean; user: User }>('/auth/me'),
  logout: () =>
    apiRequest<void>('/auth/logout', {
      method: 'POST',
    }),
}
