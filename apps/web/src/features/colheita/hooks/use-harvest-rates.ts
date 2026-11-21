import { listHarvestRates } from '../api/list-harvest-rates'
import { useHarvestQuery } from './use-harvest-query'

export function useHarvestRates() {
  return useHarvestQuery('rates', [], listHarvestRates)
}
