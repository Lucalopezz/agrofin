import { deactivateEmployee } from '../api/deactivate-employee'
import { useEmployeeWrite } from './use-employee-write'

export function useDeactivateEmployee() {
  return useEmployeeWrite(deactivateEmployee)
}
