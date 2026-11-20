import { useState, type FormEvent } from 'react'
import { Check, ClipboardList, RotateCcw } from 'lucide-react'
import { useEmployees } from '@/features/funcionarios/hooks/use-employees'
import {
  DemoNotice,
  Feedback,
  FieldInput,
  FieldSelect,
  FormActions,
  FormField,
  SectionCard,
} from '@/features/painel/components/page-ui'
import { formatDemoDate } from '@/features/painel/lib/demo-date'
import { Button } from '@/shared/components/ui/button'
import type { HarvestRecord } from './harvest-demo-history'

type HarvestDemoFormProps = {
  onOpenEmployees: () => void
  onSave: (record: HarvestRecord) => void
}

export function HarvestDemoForm({ onOpenEmployees, onSave }: HarvestDemoFormProps) {
  const list = useEmployees(true)
  // O ID distingue funcionários com o mesmo nome.
  const [employeeId, setEmployeeId] = useState('')
  const employees = list.loading || list.error ? [] : list.employees
  const employee = employees.find(({ id }) => id === employeeId)
  const [date, setDate] = useState('2026-10-30')
  const [normal, setNormal] = useState('')
  const [moagem, setMoagem] = useState('')
  const [saved, setSaved] = useState(false)
  const countNormal = Number(normal) || 0
  const countMoagem = Number(moagem) || 0
  const total = countNormal * 5 + countMoagem * 2

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!employee || countNormal + countMoagem < 1) return
    onSave({
      person: employee.name,
      date: formatDemoDate(date),
      normal: countNormal,
      moagem: countMoagem,
      total,
    })
    clearQuantities()
    setSaved(true)
  }

  function clearQuantities() {
    setNormal('')
    setMoagem('')
    setSaved(false)
  }

  return (
    <SectionCard title="Dados da colheita">
      <DemoNotice>Os lançamentos e valores desta tela são demonstrativos.</DemoNotice>
      {saved && (
        <Feedback>Lançamento incluído nesta demonstração. Você pode registrar outra pessoa.</Feedback>
      )}
      <form className="demo-form" onSubmit={save}>
        <div className="form-grid">
          <FormField label="Data">
            <FieldInput
              type="date"
              value={date}
              onChange={(event) => setDate(event.target.value)}
              required
            />
          </FormField>
          <FormField label="Funcionário">
            <FieldSelect
              value={employee?.id ?? ''}
              onChange={(event) => {
                setEmployeeId(event.target.value)
                setSaved(false)
              }}
              disabled={employees.length === 0}
              required
            >
              <option value="">Selecione um funcionário</option>
              {employees.map(({ id, name }) => (
                <option key={id} value={id}>{name}</option>
              ))}
            </FieldSelect>
          </FormField>
        </div>
        {list.loading && <p role="status">Carregando funcionários...</p>}
        {!list.loading && list.error && (
          <div>
            <p role="alert">{list.error}</p>
            <Button type="button" variant="outline" onClick={list.reload}>Tentar novamente</Button>
          </div>
        )}
        {!list.loading && !list.error && employees.length === 0 && (
          <div>
            <p role="status">Nenhum funcionário ativo.</p>
            <Button type="button" variant="outline" onClick={onOpenEmployees}>
              Gerenciar funcionários
            </Button>
          </div>
        )}
        <div className="form-divider"><span>Quantidade</span></div>
        <div className="form-grid">
          <FormField label="Caixas normais" hint="R$ 5 por caixa">
            <div className="input-with-unit">
              <FieldInput
                type="number"
                min="0"
                step="1"
                inputMode="numeric"
                placeholder="0"
                value={normal}
                onChange={(event) => {
                  setNormal(event.target.value)
                  setSaved(false)
                }}
              />
              <span>caixas</span>
            </div>
          </FormField>
          <FormField label="Caixas de moagem" hint="R$ 2 por caixa">
            <div className="input-with-unit">
              <FieldInput
                type="number"
                min="0"
                step="1"
                inputMode="numeric"
                placeholder="0"
                value={moagem}
                onChange={(event) => {
                  setMoagem(event.target.value)
                  setSaved(false)
                }}
              />
              <span>caixas</span>
            </div>
          </FormField>
        </div>
        <div className="harvest-preview">
          <div>
            <span className="preview-icon"><ClipboardList size={17} /></span>
            <span className="harvest-preview-summary">
              <small>Prévia do lançamento</small>
              <strong>{employee?.name ?? 'Selecione um funcionário'} · {countNormal + countMoagem} caixas</strong>
            </span>
          </div>
          <div className="preview-money">
            <small>Valor estimado</small>
            <strong>{total.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong>
          </div>
        </div>
        <FormActions>
          <Button type="submit" disabled={!employee || countNormal + countMoagem < 1}>
            <Check size={16} /> Salvar
          </Button>
          <Button type="button" variant="outline" onClick={clearQuantities}>
            <RotateCcw size={15} /> Limpar
          </Button>
        </FormActions>
      </form>
    </SectionCard>
  )
}
