import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Button } from '@/shared/components/ui/button'
import {
  FieldInput,
  FormActions,
  FormField,
  SectionCard,
} from '@/features/painel/components/page-ui'
import type { Employee } from '../types'

type EmployeeFormProps = {
  employee: Employee | null
  busy: boolean
  onSave: (name: string) => Promise<boolean>
  onCancel: () => void
}

export function EmployeeForm({ employee, busy, onSave, onCancel }: EmployeeFormProps) {
  const [name, setName] = useState(employee?.name ?? '')
  const input = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (employee) input.current?.focus()
  }, [employee])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const normalizedName = name.trim()
    if (busy || !normalizedName) return

    if (await onSave(normalizedName)) setName('')
  }

  return (
    <SectionCard title={employee ? 'Renomear funcionário' : 'Cadastrar funcionário'}>
      <form className="demo-form" onSubmit={(event) => void submit(event)}>
        <FormField label="Nome">
          <FieldInput
            ref={input}
            value={name}
            onChange={(event) => setName(event.target.value)}
            autoComplete="off"
            required
            disabled={busy}
          />
        </FormField>
        <FormActions>
          <Button type="submit" disabled={busy || !name.trim()}>
            {employee ? 'Salvar nome' : 'Cadastrar'}
          </Button>
          {employee && (
            <Button type="button" variant="outline" onClick={onCancel} disabled={busy}>
              Cancelar
            </Button>
          )}
        </FormActions>
      </form>
    </SectionCard>
  )
}
