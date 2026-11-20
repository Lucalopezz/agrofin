import { useState } from 'react'
import {
  Feedback,
  FieldSelect,
  FormField,
  PageHeading,
  SectionCard,
} from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import type { Employee } from '../types'
import { EmployeeForm } from '../components/employee-form'
import { EmployeeList } from '../components/employee-list'
import { useEmployeeActions } from '../hooks/use-employee-actions'
import { useEmployees } from '../hooks/use-employees'

export function FuncionariosPage() {
  const [filter, setFilter] = useState('todos')
  const active = filter === 'todos' ? undefined : filter === 'ativos'
  const list = useEmployees(active)
  const [editing, setEditing] = useState<Employee | null>(null)
  const actions = useEmployeeActions()
  const { busy, feedback } = actions

  async function saveName(name: string) {
    if (!editing) return actions.create(name)

    const saved = await actions.rename(editing.id, name)
    if (saved) setEditing(null)
    return saved
  }

  function changeStatus(employee: Employee) {
    return employee.active ? actions.deactivate(employee.id) : actions.reactivate(employee.id)
  }

  return (
    <>
      <PageHeading
        eyebrow="Produção"
        title="Funcionários"
        description="Cadastre e gerencie os funcionários da sua conta."
      />
      {feedback?.status === 'success' && <Feedback>{feedback.message}</Feedback>}
      {feedback?.status === 'error' && (
        <p className="auth-error" role="alert">{feedback.message}</p>
      )}
      {busy && <p role="status">Salvando alteração...</p>}
      <div className="form-page-grid employee-page-grid">
        <EmployeeForm
          key={editing?.id ?? 'new'}
          employee={editing}
          onSave={saveName}
          onCancel={() => setEditing(null)}
          busy={busy}
        />
      </div>
      <SectionCard title="Funcionários cadastrados">
        <FormField label="Situação">
          <FieldSelect
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            disabled={busy}
          >
            <option value="todos">Todos</option>
            <option value="ativos">Ativos</option>
            <option value="inativos">Inativos</option>
          </FieldSelect>
        </FormField>
        {list.loading && <p role="status">Carregando funcionários...</p>}
        {!list.loading && list.error && (
          <div>
            <p role="alert">{list.error}</p>
            <Button type="button" variant="outline" onClick={list.reload}>Tentar novamente</Button>
          </div>
        )}
        {!list.loading && !list.error && (
          list.employees.length ? (
            <EmployeeList
              employees={list.employees}
              busy={busy}
              onRename={setEditing}
              onChangeStatus={(employee) => void changeStatus(employee)}
            />
          ) : (
            <p role="status">Nenhum funcionário encontrado.</p>
          )
        )}
      </SectionCard>
    </>
  )
}
