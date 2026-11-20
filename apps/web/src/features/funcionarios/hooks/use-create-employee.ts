import { createEmployee } from '../api/create-employee'
import { useEmployeeWrite } from './use-employee-write'

export function useCreateEmployee() {
  return useEmployeeWrite(createEmployee)
}
