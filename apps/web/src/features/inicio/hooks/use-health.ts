import { useQuery } from '@tanstack/react-query'
import { checkHealth } from '../api/check-health'

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: ({ signal }) => checkHealth(signal),
  })
}
