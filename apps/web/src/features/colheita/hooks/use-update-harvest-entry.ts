import { updateHarvestEntry } from '../api/update-harvest-entry'
import { useHarvestWrite } from './use-harvest-write'

export function useUpdateHarvestEntry() {
  return useHarvestWrite('entries', updateHarvestEntry)
}
