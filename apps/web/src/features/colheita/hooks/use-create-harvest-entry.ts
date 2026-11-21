import { createHarvestEntry } from '../api/create-harvest-entry'
import { useHarvestWrite } from './use-harvest-write'

export function useCreateHarvestEntry() {
  return useHarvestWrite('entries', createHarvestEntry)
}
