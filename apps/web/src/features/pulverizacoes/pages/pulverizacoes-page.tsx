import { useState, type FormEvent } from 'react'
import { CalendarDays, Check, Plus, Sprout } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import { Feedback, FieldInput, FormActions, FormField, PageHeading, SectionCard } from '@/features/painel/components/page-ui'
import { formatDemoDate } from '@/features/painel/lib/demo-date'
import type { DemoPage } from '@/features/painel/components/dashboard-layout'

type Ingredient = { name: string; unit: string; selected: boolean; amount: string; stock: string }
const starterIngredients: Ingredient[] = [
  { name: 'Óleo mineral', unit: 'L', selected: true, amount: '3', stock: '18 L disponível' },
  { name: 'Fertilizante foliar', unit: 'L', selected: false, amount: '', stock: '12 L disponível' },
  { name: 'Cobre agrícola', unit: 'kg', selected: false, amount: '', stock: 'Estoque a conferir' },
]

export function PulverizacoesPage({ onNavigate }: { onNavigate: (page: DemoPage) => void }) {
  const [date, setDate] = useState('2026-10-30')
  const [sector, setSector] = useState('Setor norte')
  const [ingredients, setIngredients] = useState(starterIngredients)
  const [showQuickAdd, setShowQuickAdd] = useState(false)
  const [newName, setNewName] = useState('')
  const [newUnit, setNewUnit] = useState('L')
  const [saved, setSaved] = useState(false)
  const [events, setEvents] = useState([{ date: '22 out', sector: 'Setor norte', products: 'Óleo mineral' }, { date: '16 out', sector: 'Setor sul', products: 'Cobre agrícola' }])
  const selected = ingredients.filter(item => item.selected)

  function toggleIngredient(index: number) {
    setIngredients(current => current.map((item, currentIndex) => currentIndex === index ? { ...item, selected: !item.selected } : item))
  }

  function addIngredient() {
    const name = newName.trim()
    if (!name) return
    // O cadastro rápido preserva o rascunho da pulverização e sinaliza que o saldo do insumo precisa ser conferido.
    setIngredients(current => [...current, { name, unit: newUnit, selected: true, amount: '', stock: 'Estoque pendente de conferência' }])
    setNewName('')
    setShowQuickAdd(false)
  }

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!selected.length) return
    setEvents(current => [{ date: formatDemoDate(date), sector, products: selected.map(item => item.name).join(', ') }, ...current])
    setSaved(true)
  }

  return <>
    <PageHeading eyebrow="Manejo" title="Pulverização" description="Registre data, setor e insumos." action={<button className="date-filter" onClick={() => onNavigate('calendario')}><CalendarDays size={16} /> Calendário</button>} />
    <SectionCard title="Nova aplicação">
        {saved ? <Feedback>Pulverização adicionada ao histórico fictício. O formulário continua disponível para revisar.</Feedback> : null}
        <form className="demo-form" onSubmit={save}>
          <div className="form-grid"><FormField label="data"><FieldInput type="date" value={date} onChange={event => setDate(event.target.value)} required /></FormField><FormField label="setor"><FieldInput value={sector} onChange={event => setSector(event.target.value)} placeholder="Ex.: setor norte" required /></FormField></div>
          <div className="form-divider"><span>Insumos</span></div>
          <div className="ingredient-list">{ingredients.map((item, index) => <div className={`ingredient-row ${item.selected ? 'ingredient-selected' : ''}`} key={`${item.name}-${index}`}><label className="ingredient-check"><input type="checkbox" checked={item.selected} onChange={() => toggleIngredient(index)} /><span className="custom-check"><Check size={12} /></span><span className="ingredient-name"><strong>{item.name}</strong><small>{item.stock}</small></span></label><div className="ingredient-amount"><label className="sr-only" htmlFor={`quantity-${index}`}>Quantidade de {item.name}</label><FieldInput id={`quantity-${index}`} type="number" min="0" step="0.1" inputMode="decimal" placeholder="0" disabled={!item.selected} value={item.amount} onChange={event => setIngredients(current => current.map((currentItem, currentIndex) => currentIndex === index ? { ...currentItem, amount: event.target.value } : currentItem))} /><span>{item.unit}</span></div></div>)}</div>
          <button className="inline-add" type="button" onClick={() => setShowQuickAdd(open => !open)}><Plus size={16} /> Cadastrar insumo sem sair daqui</button>
          {showQuickAdd ? <div className="quick-add-form"><strong>Novo insumo</strong><div className="quick-add-fields"><label className="form-field"><span>Nome</span><FieldInput value={newName} onChange={event => setNewName(event.target.value)} placeholder="Ex.: espalhante adesivo" autoFocus /></label><label className="form-field"><span>Unidade</span><select className="form-control" value={newUnit} onChange={event => setNewUnit(event.target.value)}><option value="L">Litro (L)</option><option value="kg">Quilograma (kg)</option><option value="un">Unidade</option></select></label></div><div className="quick-stock-note"><Sprout size={15} /> Estoque a conferir.</div><Button type="button" disabled={!newName.trim()} onClick={addIngredient}><Plus size={15} /> Adicionar</Button></div> : null}
          <FormActions><Button type="submit" disabled={!selected.length}><Check size={16} /> Salvar</Button></FormActions>
        </form>
    </SectionCard>
    <SectionCard title="Últimas aplicações">
      <div className="table-mobile-list spray-history-list">{events.map((event, index) => <article className="mobile-record" key={`${event.date}-${index}`}><div><strong>{event.sector}</strong><small>{event.date} · {event.products}</small></div></article>)}</div>
    </SectionCard>
  </>
}
