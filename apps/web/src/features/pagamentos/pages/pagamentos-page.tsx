import { useMemo, useState, type FormEvent } from 'react'
import { ArrowRight, CalendarDays, Check } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import { Feedback, FieldInput, FormActions, FormField, PageHeading, SectionCard } from '@/features/painel/components/page-ui'
import { formatDemoDate } from '@/features/painel/lib/demo-date'

type EmployeeBalance = { name: string; normal: number; moagem: number; daily: number; due: number; paid: number }
const initialBalances: EmployeeBalance[] = [
  { name: 'Ana Paula', normal: 76, moagem: 12, daily: 0, due: 404, paid: 200 },
  { name: 'João Carlos', normal: 64, moagem: 8, daily: 80, due: 416, paid: 0 },
  { name: 'Raimundo Silva', normal: 58, moagem: 18, daily: 0, due: 326, paid: 250 },
  { name: 'Dona Célia', normal: 48, moagem: 0, daily: 0, due: 240, paid: 240 },
]

export function PagamentosPage() {
  const [balances, setBalances] = useState(initialBalances)
  const [selected, setSelected] = useState<string | null>(null)
  const [amount, setAmount] = useState('')
  const [paymentDate, setPaymentDate] = useState('2026-10-30')
  const [savedName, setSavedName] = useState('')
  const [savedDate, setSavedDate] = useState('')
  const pending = balances.reduce((sum, item) => sum + item.due - item.paid, 0)
  const openEmployee = balances.find(item => item.name === selected)
  const remaining = openEmployee ? openEmployee.due - openEmployee.paid : 0
  const preview = useMemo(() => Math.min(Number(amount) || 0, remaining), [amount, remaining])

  function savePayment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!openEmployee || preview <= 0) return
    setBalances(items => items.map(item => item.name === openEmployee.name ? { ...item, paid: item.paid + preview } : item))
    setSavedName(openEmployee.name)
    setSavedDate(formatDemoDate(paymentDate))
    setSelected(null)
    setAmount('')
  }

  return <>
    <PageHeading eyebrow="Produção" title="Pagamentos" description="Confira o saldo da equipe." action={<div className="date-filter"><CalendarDays size={16} /> 26 out — 1 nov</div>} />
    <div className="payroll-totals"><span>Devido <strong>R$ {balances.reduce((sum, item) => sum + item.due, 0).toLocaleString('pt-BR')}</strong></span><span>Pago <strong>R$ {balances.reduce((sum, item) => sum + item.paid, 0).toLocaleString('pt-BR')}</strong></span><span>Pendente <strong>R$ {pending.toLocaleString('pt-BR')}</strong></span></div>

    {savedName ? <Feedback>Pagamento de demonstração registrado para {savedName}, em {savedDate}. O saldo foi atualizado nesta tela.</Feedback> : null}
    {selected && openEmployee ? <SectionCard title={`registrar pagamento · ${openEmployee.name}`} subtitle={`Saldo pendente: ${remaining.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}`} className="payment-editor-card">
      <form className="demo-form" onSubmit={savePayment}><div className="form-grid"><FormField label="valor"><div className="input-with-unit currency-input"><span>R$</span><FieldInput type="number" min="0.01" step="0.01" max={remaining} inputMode="decimal" placeholder={remaining.toFixed(2)} value={amount} onChange={event => setAmount(event.target.value)} required autoFocus /></div></FormField><FormField label="data"><FieldInput type="date" value={paymentDate} onChange={event => setPaymentDate(event.target.value)} required /></FormField></div><div className="payment-preview"><span>Saldo após o pagamento</span><strong>{(remaining - preview).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></div><FormActions><Button type="submit" disabled={preview <= 0 || Number(amount) > remaining}><Check size={16} /> Confirmar</Button><Button type="button" variant="outline" onClick={() => setSelected(null)}>Cancelar</Button></FormActions></form>
    </SectionCard> : null}

    <SectionCard title="Equipe">
      <div className="table-scroll"><table className="data-table payroll-table"><thead><tr><th>Funcionário</th><th>Caixas normais</th><th>Caixas moagem</th><th>Diárias</th><th>Total devido</th><th>Pago</th><th>Pendente</th><th /></tr></thead><tbody>{balances.map(item => { const balance = item.due - item.paid; return <tr key={item.name}><td><span className="table-person"><span>{item.name.split(' ').map(part => part[0]).slice(0, 2).join('')}</span>{item.name}</span></td><td>{item.normal} cx</td><td>{item.moagem} cx</td><td>{item.daily ? `R$ ${item.daily}` : '—'}</td><td>{item.due.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</td><td>{item.paid.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</td><td className={balance ? 'pending-money' : 'settled-money'}>{balance.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</td><td><button className="row-action" onClick={() => { setSelected(item.name); setAmount(''); setSavedName('') }} disabled={balance <= 0}>{balance ? 'Pagar' : 'Quitado'} <ArrowRight size={13} /></button></td></tr> })}</tbody></table></div>
      <div className="table-mobile-list">{balances.map(item => { const balance = item.due - item.paid; return <article className="mobile-pay-card" key={item.name}><div className="mobile-pay-heading"><span className="table-person"><span>{item.name.split(' ').map(part => part[0]).slice(0, 2).join('')}</span>{item.name}</span><strong className={balance ? 'pending-money' : 'settled-money'}>{balance ? `${balance.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })} pendente` : 'Quitado'}</strong></div><small>{item.normal} cx normais · {item.moagem} cx moagem{item.daily ? ` · diária R$ ${item.daily}` : ''}</small><div className="mobile-pay-totals"><span>Devido <strong>{item.due.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></span><span>Pago <strong>{item.paid.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</strong></span></div><button className="row-action" onClick={() => { setSelected(item.name); setAmount(''); setSavedName('') }} disabled={balance <= 0}>{balance ? 'Registrar pagamento' : 'Sem saldo pendente'} <ArrowRight size={13} /></button></article> })}</div>
    </SectionCard>
  </>
}
