import { Button } from '@/shared/components/ui/button'
import { FormActions } from '@/features/painel/components/page-ui'
import type { Employee } from '../types'

type EmployeeListProps = {
  employees: Employee[]
  busy: boolean
  onRename: (employee: Employee) => void
  onChangeStatus: (employee: Employee) => void
}

export function EmployeeList({ employees, busy, onRename, onChangeStatus }: EmployeeListProps) {
  return (
    <ul className="employee-list" aria-label="Funcionários cadastrados">
      {employees.map((employee) => (
        <li key={employee.id}>
          <div className="employee-list-name">
            <strong>{employee.name}</strong>
            <small>{employee.active ? 'Ativo' : 'Inativo'}</small>
          </div>
          <FormActions>
            <Button
              type="button"
              variant="outline"
              aria-label={`Renomear ${employee.name}`}
              disabled={busy}
              onClick={() => onRename(employee)}
            >
              Renomear
            </Button>
            <Button
              type="button"
              variant="outline"
              aria-label={`${employee.active ? 'Desativar' : 'Reativar'} ${employee.name}`}
              disabled={busy}
              onClick={() => onChangeStatus(employee)}
            >
              {employee.active ? 'Desativar' : 'Reativar'}
            </Button>
          </FormActions>
        </li>
      ))}
    </ul>
  )
}
