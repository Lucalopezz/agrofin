import { useState } from 'react'
import { AlertCircle, Check, Package, Plus } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import { Feedback, FieldInput, FieldSelect, FormActions, FormField, PageHeading, SectionCard } from '@/features/painel/components/page-ui'

type Supply = { name: string; category: string; unit: string; stock: string; state: 'ok' | 'check' }
const initialSupplies: Supply[] = [
  { name: 'Óleo mineral', category: 'Defensivo', unit: 'L', stock: '18 L', state: 'ok' },
  { name: 'Fertilizante foliar', category: 'Fertilizante', unit: 'L', stock: '12 L', state: 'ok' },
  { name: 'Cobre agrícola', category: 'Defensivo', unit: 'kg', stock: 'A conferir', state: 'check' },
  { name: 'Adubo NPK', category: 'Fertilizante', unit: 'kg', stock: 'A conferir', state: 'check' },
]

export function InsumosPage() {
  const [items, setItems] = useState(initialSupplies)
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [category, setCategory] = useState('Fertilizante')
  const [unit, setUnit] = useState('L')
  const [saved, setSaved] = useState(false)

  function addItem(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const cleanName = name.trim()
    if (!cleanName) return
    setItems(current => [{ name: cleanName, category, unit, stock: 'A conferir', state: 'check' }, ...current])
    setName('')
    setSaved(true)
    setShowForm(false)
  }

  return <>
    <PageHeading eyebrow="Manejo" title="Insumos" description="Produtos e saldos do pomar." action={<Button onClick={() => { setShowForm(open => !open); setSaved(false) }}><Plus size={16} /> Novo insumo</Button>} />
    {saved ? <Feedback>Insumo salvo. Saldo a conferir.</Feedback> : null}
    {showForm ? <SectionCard title="Novo insumo"><form className="demo-form" onSubmit={addItem}><div className="form-grid"><FormField label="nome"><FieldInput value={name} onChange={event => setName(event.target.value)} placeholder="Ex.: calcário dolomítico" required autoFocus /></FormField><FormField label="categoria"><FieldSelect value={category} onChange={event => setCategory(event.target.value)}><option>Fertilizante</option><option>Defensivo</option></FieldSelect></FormField><FormField label="unidade"><FieldSelect value={unit} onChange={event => setUnit(event.target.value)}><option value="L">Litro (L)</option><option value="kg">Quilograma (kg)</option><option value="un">Unidade</option></FieldSelect></FormField></div><div className="quick-stock-note"><AlertCircle size={15} /> Saldo a conferir.</div><FormActions><Button type="submit" disabled={!name.trim()}><Check size={16} /> Salvar</Button><Button type="button" variant="outline" onClick={() => setShowForm(false)}>Cancelar</Button></FormActions></form></SectionCard> : null}
    <p className="inventory-count">{items.length} insumos <span>·</span> {items.filter(item => item.state === 'check').length} saldos a conferir</p>
    <SectionCard title="Produtos">
      <div className="inventory-list">{items.map((item, index) => <article className="inventory-row" key={`${item.name}-${index}`}><span className={`inventory-product-icon inventory-tone-${index % 3}`}><Package size={18} /></span><div className="inventory-product"><strong>{item.name}</strong><small>{item.category} · {item.unit}</small></div><div className="inventory-stock"><strong>{item.stock}</strong></div></article>)}</div>
    </SectionCard>
  </>
}
