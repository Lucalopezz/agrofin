import { useState, type FormEvent } from 'react'
import { Check } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import { Feedback, FieldInput, FieldSelect, FormActions, FormField, PageHeading, SectionCard } from '@/features/painel/components/page-ui'
import { formatDemoDate } from '@/features/painel/lib/demo-date'

type Expense = { date: string; category: string; description: string; amount: number }
const initialExpenses: Expense[] = [
  { date: '28 out', category: 'Energia e irrigação', description: 'Conta de energia · bomba do poço', amount: 480 },
  { date: '24 out', category: 'Fertilizantes', description: 'Adubo foliar', amount: 365 },
  { date: '20 out', category: 'Outras despesas', description: 'Manutenção de equipamento', amount: 220 },
]

export function DespesasPage() {
  const [date, setDate] = useState('2026-10-30')
  const [category, setCategory] = useState('Energia e irrigação')
  const [description, setDescription] = useState('')
  const [amount, setAmount] = useState('')
  const [expenses, setExpenses] = useState(initialExpenses)
  const [saved, setSaved] = useState(false)

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const value = Number(amount)
    if (value <= 0 || !description.trim()) return
    setExpenses(current => [{ date: formatDemoDate(date), category, description: description.trim(), amount: value }, ...current])
    setAmount('')
    setDescription('')
    setSaved(true)
  }

  return <>
    <PageHeading eyebrow="Financeiro" title="Despesas" description="Registre gastos pagos." />
    <div className="form-page-grid expense-page-grid">
      <SectionCard title="Nova despesa">
        {saved ? <Feedback>Despesa registrada nesta demonstração.</Feedback> : null}
        <form className="demo-form" onSubmit={save}><div className="form-grid"><FormField label="data"><FieldInput type="date" value={date} onChange={event => setDate(event.target.value)} required /></FormField><FormField label="categoria"><FieldSelect value={category} onChange={event => setCategory(event.target.value)}><option>Energia e irrigação</option><option>Fertilizantes</option><option>Defensivos</option><option>Manutenção</option><option>Outras despesas</option></FieldSelect></FormField></div><FormField label="descrição"><FieldInput value={description} onChange={event => setDescription(event.target.value)} placeholder="Ex.: conta de energia" required /></FormField><FormField label="valor"><div className="input-with-unit currency-input"><span>R$</span><FieldInput type="number" min="0.01" step="0.01" inputMode="decimal" value={amount} onChange={event => setAmount(event.target.value)} placeholder="0,00" required /></div></FormField><FormActions><Button type="submit" disabled={!description.trim() || Number(amount) <= 0}><Check size={16} /> Salvar</Button></FormActions></form>
      </SectionCard>
    </div>
    <p className="expense-total">{expenses.length} despesas · {expenses.reduce((sum, item) => sum + item.amount, 0).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })} em outubro</p>
    <SectionCard title="Últimas despesas"><div className="table-scroll"><table className="data-table"><thead><tr><th>Data</th><th>Categoria</th><th>Descrição</th><th>Valor</th></tr></thead><tbody>{expenses.map((item, index) => <tr key={`${item.description}-${index}`}><td>{item.date}</td><td><span className="category-chip">{item.category}</span></td><td>{item.description}</td><td className="money-cell">{item.amount.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</td></tr>)}</tbody></table></div><div className="table-mobile-list">{expenses.map((item, index) => <article className="mobile-record" key={`${item.description}-${index}`}><div><strong>{item.description}</strong><small>{item.date} · {item.category}</small></div><strong>{item.amount.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></article>)}</div></SectionCard>
  </>
}
