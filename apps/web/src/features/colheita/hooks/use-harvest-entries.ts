import { listHarvestEntries } from '../api/list-harvest-entries'
import type { Period } from '../types'
import { useHarvestQuery } from './use-harvest-query'

export function useHarvestEntries(period: Period) {
  return useHarvestQuery('entries', [period.start_date, period.end_date],
    (signal) => listHarvestEntries(period, signal),
    Boolean(period.start_date && period.end_date && period.start_date <= period.end_date),
  )
}
