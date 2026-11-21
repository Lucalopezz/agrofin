import { createHarvestRate } from '../api/create-harvest-rate'
import { useHarvestWrite } from './use-harvest-write'

export function useCreateHarvestRate() {
  return useHarvestWrite('rates', createHarvestRate)
}
