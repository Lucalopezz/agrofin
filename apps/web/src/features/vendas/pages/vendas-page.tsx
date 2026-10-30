import { useState, type FormEvent } from 'react'
import { Check } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import { Feedback, FieldInput, FieldSelect, FormActions, FormField, PageHeading, SectionCard } from '@/features/painel/components/page-ui'

type SaleLine = { type: string; count: string; price: string }

export function VendasPage() {
  const [date, setDate] = useState('2026-10-30')
  const [lines, setLines] = useState<SaleLine[]>([{ type: 'Exportação', count: '', price: '' }, { type: 'Mercado', count: '', price: '' }, { type: 'Moagem', count: '', price: '' }])
  const [discount, setDiscount] = useState('')
  const [reason, setReason] = useState('')
  const [received, setReceived] = useState('')
  const [saved, setSaved] = useState(false)
  const gross = lines.reduce((sum, line) => sum + (Number(line.count) || 0) * (Number(line.price) || 0), 0)
  const net = Math.max(0, gross - (Number(discount) || 0))
  const due = Math.max(0, net - (Number(received) || 0))
  const boxCount = lines.reduce((sum, line) => sum + (Number(line.count) || 0), 0)

  function changeLine(index: number, key: 'count' | 'price', value: string) {
    setLines(current => current.map((line, lineIndex) => lineIndex === index ? { ...line, [key]: value } : line))
    setSaved(false)
  }

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (gross <= 0 || (Number(discount) > 0 && !reason.trim())) return
    setSaved(true)
  }

  return <>
    <PageHeading eyebrow="Financeiro" title="Registrar venda" description="Informe as caixas e confira o acerto." />
    <div className="form-page-grid sales-page-grid">
      <SectionCard title="Venda">
        {saved ? <Feedback>Venda incluída nesta demonstração.</Feedback> : null}
        <form className="demo-form" onSubmit={save}>
          <div className="form-grid"><FormField label="data"><FieldInput type="date" value={date} onChange={event => setDate(event.target.value)} required /></FormField><FormField label="barracão"><FieldSelect defaultValue="Barracão Santa Luzia"><option>Barracão Santa Luzia</option><option>Outro barracão</option></FieldSelect></FormField></div>
          <div className="sale-lines"><div className="form-divider"><span>Caixas e preço por tipo</span></div>{lines.map((line, index) => <div className="sale-line" key={line.type}><span className={`sale-type sale-type-${index}`}>{line.type}</span><label><span className="sr-only">Caixas de {line.type}</span><FieldInput type="number" min="0" step="1" inputMode="numeric" placeholder="0 cx" value={line.count} onChange={event => changeLine(index, 'count', event.target.value)} /></label><span className="sale-at">×</span><label className="sale-price-field"><span className="sr-only">Preço por caixa de {line.type}</span><span className="currency-prefix">R$</span><FieldInput type="number" min="0" step="0.01" inputMode="decimal" placeholder="0,00" value={line.price} onChange={event => changeLine(index, 'price', event.target.value)} /></label></div>)}</div>
          <div className="form-grid"><FormField label="desconto"><div className="input-with-unit currency-input"><span>R$</span><FieldInput type="number" min="0" max={gross} step="0.01" inputMode="decimal" placeholder="0,00" value={discount} onChange={event => { setDiscount(event.target.value); setSaved(false) }} /></div></FormField><FormField label="recebido"><div className="input-with-unit currency-input"><span>R$</span><FieldInput type="number" min="0" max={net} step="0.01" inputMode="decimal" placeholder="0,00" value={received} onChange={event => { setReceived(event.target.value); setSaved(false) }} /></div></FormField></div>
          {Number(discount) > 0 ? <FormField label="motivo do desconto"><FieldInput placeholder="Ex.: ajuste do barracão" value={reason} onChange={event => { setReason(event.target.value); setSaved(false) }} required /></FormField> : null}
          <FormActions><Button type="submit" disabled={gross <= 0 || Number(discount) > gross || Number(received) > net || (Number(discount) > 0 && !reason.trim())}><Check size={16} /> Salvar</Button></FormActions>
        </form>
      </SectionCard>
      <SectionCard title="Resumo">
        <div className="sale-preview-rows"><div><span>Caixas</span><strong>{boxCount}</strong></div><div><span>Valor bruto</span><strong>{gross.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></div><div><span>Desconto</span><strong>− {Number(discount || 0).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></div><div className="sale-net-row"><span>Valor líquido</span><strong>{net.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></div><div><span>Recebido</span><strong>{Number(received || 0).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></div><div className="sale-due-row"><span>A receber</span><strong>{due.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></div></div>
      </SectionCard>
    </div>
  </>
}
