import { reactivateEmployee } from '../api/reactivate-employee'
import { useEmployeeWrite } from './use-employee-write'

export function useReactivateEmployee() {
  return useEmployeeWrite(reactivateEmployee)
}
