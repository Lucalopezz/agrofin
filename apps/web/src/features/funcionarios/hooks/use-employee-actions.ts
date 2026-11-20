import { useRef, useState } from 'react'
import { sentenceCase } from '@/features/painel/lib/sentence-case'
import { ApiError } from '@/shared/lib/api-client'
import { useCreateEmployee } from './use-create-employee'
import { useDeactivateEmployee } from './use-deactivate-employee'
import { useReactivateEmployee } from './use-reactivate-employee'
import { useRenameEmployee } from './use-rename-employee'

type Feedback =
  | { status: 'success' | 'error'; message: string }
  | null

export function useEmployeeActions() {
  const create = useCreateEmployee()
  const rename = useRenameEmployee()
  const deactivate = useDeactivateEmployee()
  const reactivate = useReactivateEmployee()
  const [feedback, setFeedback] = useState<Feedback>(null)
  const pending = useRef(false)

  async function save(action: () => Promise<unknown>, message: string) {
    // Uma trava comum protege ações distintas antes do próximo render.
    if (pending.current) return false
    pending.current = true
    setFeedback(null)
    try {
      await action()
      setFeedback({ status: 'success', message })
      return true
    } catch (error) {
      setFeedback({
        status: 'error',
        message: error instanceof ApiError
          ? sentenceCase(error.message)
          : 'Não foi possível confirmar a alteração. Confira a lista antes de tentar novamente.',
      })
      return false
    } finally {
      pending.current = false
    }
  }

  return {
    busy: create.isPending || rename.isPending || deactivate.isPending || reactivate.isPending,
    feedback,
    create: (name: string) => save(() => create.mutateAsync(name), 'Funcionário cadastrado.'),
    rename: (id: string, name: string) =>
      save(() => rename.mutateAsync({ id, name }), 'Nome atualizado.'),
    deactivate: (id: string) => save(() => deactivate.mutateAsync(id), 'Funcionário desativado.'),
    reactivate: (id: string) => save(() => reactivate.mutateAsync(id), 'Funcionário reativado.'),
  }
}
