import { renameEmployee } from '../api/rename-employee'
import { useEmployeeWrite } from './use-employee-write'

export function useRenameEmployee() {
  return useEmployeeWrite(renameEmployee)
}
