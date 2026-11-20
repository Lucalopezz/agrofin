import { useRef, useState } from 'react'
import {
  Feedback,
  FieldSelect,
  FormField,
  PageHeading,
  SectionCard,
} from '@/features/painel/components/page-ui'
import { sentenceCase } from '@/features/painel/lib/sentence-case'
import { Button } from '@/shared/components/ui/button'
import { ApiError } from '@/shared/lib/api-client'
import type { Employee } from '../api/employees-api'
import { EmployeeForm } from '../components/employee-form'
import { EmployeeList } from '../components/employee-list'
import { useEmployeeMutation } from '../hooks/use-employee-mutation'
import { useEmployees } from '../hooks/use-employees'

export function FuncionariosPage() {
  const [filter, setFilter] = useState('todos')
  const active = filter === 'todos' ? undefined : filter === 'ativos'
  const list = useEmployees(active)
  const [editing, setEditing] = useState<Employee | null>(null)
  const pending = useRef(false)
  const mutation = useEmployeeMutation()
  const busy = mutation.isPending
  const error = mutation.isError && (mutation.error instanceof ApiError
    ? sentenceCase(mutation.error.message)
    : 'Não foi possível confirmar a alteração. Confira a lista antes de tentar novamente.')

  async function save(change: Parameters<typeof mutation.mutateAsync>[0]) {
    // A trava vale antes de isPending chegar ao próximo render.
    if (pending.current) return false
    pending.current = true
    try {
      await mutation.mutateAsync(change)
      return true
    } catch {
      return false
    } finally {
      pending.current = false
    }
  }

  async function saveName(name: string) {
    if (!editing) return save({ action: 'create', name })

    const saved = await save({ action: 'rename', id: editing.id, name })
    if (saved) setEditing(null)
    return saved
  }

  function changeStatus(employee: Employee) {
    return save({
      action: employee.active ? 'deactivate' : 'reactivate',
      id: employee.id,
    })
  }

  return (
    <>
      <PageHeading
        eyebrow="Produção"
        title="Funcionários"
        description="Cadastre e gerencie os funcionários da sua conta."
      />
      {mutation.data && <Feedback>{mutation.data}</Feedback>}
      {error && <p className="auth-error" role="alert">{error}</p>}
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
