import { useState, type FormEvent } from 'react'
import { Check, ClipboardList, RotateCcw } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import { Feedback, FieldInput, FieldSelect, FormActions, FormField, PageHeading, SectionCard } from '@/features/painel/components/page-ui'
import { formatDemoDate } from '@/features/painel/lib/demo-date'

type HarvestRecord = { person: string; date: string; normal: number; moagem: number; total: number }
const initialRows: HarvestRecord[] = [
  { person: 'Ana Paula', date: 'Hoje, 30 out', normal: 24, moagem: 4, total: 128 },
  { person: 'João Carlos', date: 'Hoje, 30 out', normal: 18, moagem: 0, total: 90 },
  { person: 'Raimundo Silva', date: 'Ontem, 29 out', normal: 21, moagem: 3, total: 111 },
]
const staff = ['Ana Paula', 'João Carlos', 'Raimundo Silva', 'Dona Célia']

export function ColheitaPage() {
  const [person, setPerson] = useState('Ana Paula')
  const [date, setDate] = useState('2026-10-30')
  const [normal, setNormal] = useState('')
  const [moagem, setMoagem] = useState('')
  const [rows, setRows] = useState(initialRows)
  const [saved, setSaved] = useState(false)
  const countNormal = Number(normal) || 0
  const countMoagem = Number(moagem) || 0
  const total = countNormal * 5 + countMoagem * 2

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (countNormal + countMoagem < 1) return
    setRows(current => [{ person, date: formatDemoDate(date), normal: countNormal, moagem: countMoagem, total }, ...current])
    setNormal('')
    setMoagem('')
    setSaved(true)
  }

  return <>
    <PageHeading eyebrow="Produção" title="Registrar colheita" description="Informe funcionário e caixas." />
    <div className="form-page-grid harvest-page-grid">
      <SectionCard title="Dados da colheita">
        {saved ? <Feedback>Lançamento incluído nesta demonstração. Você pode registrar outra pessoa.</Feedback> : null}
        <form className="demo-form" onSubmit={save}>
          <div className="form-grid"><FormField label="data"><FieldInput type="date" value={date} onChange={event => setDate(event.target.value)} required /></FormField><FormField label="funcionário"><FieldSelect value={person} onChange={event => setPerson(event.target.value)}>{staff.map(name => <option key={name}>{name}</option>)}</FieldSelect></FormField></div>
          <div className="form-divider"><span>Quantidade</span></div>
          <div className="form-grid"><FormField label="caixas normais" hint="R$ 5 por caixa"><div className="input-with-unit"><FieldInput type="number" min="0" step="1" inputMode="numeric" placeholder="0" value={normal} onChange={event => { setNormal(event.target.value); setSaved(false) }} /><span>caixas</span></div></FormField><FormField label="caixas de moagem" hint="R$ 2 por caixa"><div className="input-with-unit"><FieldInput type="number" min="0" step="1" inputMode="numeric" placeholder="0" value={moagem} onChange={event => { setMoagem(event.target.value); setSaved(false) }} /><span>caixas</span></div></FormField></div>
          <div className="harvest-preview"><div><span className="preview-icon"><ClipboardList size={17} /></span><span><small>Prévia do lançamento</small><strong>{person} · {countNormal + countMoagem} caixas</strong></span></div><div className="preview-money"><small>Valor estimado</small><strong>{total.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></div></div>
          <FormActions><Button type="submit" disabled={countNormal + countMoagem < 1}><Check size={16} /> Salvar</Button><Button type="button" variant="outline" onClick={() => { setNormal(''); setMoagem(''); setSaved(false) }}><RotateCcw size={15} /> Limpar</Button></FormActions>
        </form>
      </SectionCard>
    </div>
    <SectionCard title="Últimos registros">
      <div className="table-scroll"><table className="data-table"><thead><tr><th>Funcionário</th><th>Data</th><th>Normal</th><th>Moagem</th><th>Total devido · exemplo</th></tr></thead><tbody>{rows.map((row, index) => <tr key={`${row.person}-${index}`}><td><span className="table-person"><span>{row.person.split(' ').map(part => part[0]).slice(0, 2).join('')}</span>{row.person}</span></td><td>{row.date}</td><td>{row.normal} cx</td><td>{row.moagem} cx</td><td className="money-cell">{row.total.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</td></tr>)}</tbody></table></div>
      <div className="table-mobile-list">{rows.map((row, index) => <article className="mobile-record" key={`${row.person}-${index}`}><div><strong>{row.person}</strong><small>{row.date} · {row.normal + row.moagem} caixas</small></div><strong>{row.total.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></article>)}</div>
    </SectionCard>
  </>
}
