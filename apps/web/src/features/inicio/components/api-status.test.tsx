import { fireEvent, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { renderWithQuery } from '@/render-with-query'
import { ApiStatus } from './api-status'

it('avisa quando o health check falha e permite consultar novamente', async () => {
  const request = vi.spyOn(globalThis, 'fetch')
    .mockResolvedValueOnce(new Response('{}', { status: 503 }))
    .mockResolvedValueOnce(new Response('{"status":"ok"}'))
  try {
    renderWithQuery(<ApiStatus />)
    expect(await screen.findByText(/Não foi possível conectar à API/)).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Verificar conexão' }))
    expect(await screen.findByText('Conexão com a API funcionando.')).toBeVisible()
    expect(request).toHaveBeenCalledTimes(2)
    expect(request).toHaveBeenLastCalledWith('/api/v1/health', expect.anything())
  } finally {
    request.mockRestore()
  }
})
