import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'
import { Check, Info, Plus, type LucideIcon } from 'lucide-react'
import { sentenceCase } from '@/features/painel/lib/sentence-case'

export function PageHeading({ eyebrow, title, description, action }: { eyebrow: string; title: string; description: string; action?: ReactNode }) {
  return <div className="page-heading"><div><p className="eyebrow">{sentenceCase(eyebrow)}</p><h1>{sentenceCase(title)}</h1><p className="page-description">{sentenceCase(description)}</p></div>{action ? <div className="heading-action">{action}</div> : null}</div>
}

export function MetricCard({ label, value, hint, icon: Icon, tone = 'green' }: { label: string; value: string; hint: string; icon: LucideIcon; tone?: 'green' | 'orange' | 'blue' | 'purple' }) {
  return <article className="metric-card"><span className={`metric-icon tone-${tone}`}><Icon size={19} strokeWidth={1.8} aria-hidden="true" /></span><p>{sentenceCase(label)}</p><strong>{value}</strong><small>{sentenceCase(hint)}</small></article>
}

export function SectionCard({ title, subtitle, action, children, className = '' }: { title: string; subtitle?: string; action?: ReactNode; children: ReactNode; className?: string }) {
  return <section className={`section-card ${className}`}><div className="section-heading"><div><h2>{sentenceCase(title)}</h2>{subtitle ? <p>{sentenceCase(subtitle)}</p> : null}</div>{action}</div>{children}</section>
}

export function DemoNotice({ children = 'Os dados e lançamentos desta tela são fictícios e usados apenas para demonstrar o fluxo.' }: { children?: ReactNode }) {
  return <div className="demo-notice"><Info size={16} aria-hidden="true" /><span>{typeof children === 'string' ? sentenceCase(children) : children}</span></div>
}

export function Feedback({ children }: { children: ReactNode }) {
  return <div className="success-feedback" role="status"><span><Check size={15} aria-hidden="true" /></span>{typeof children === 'string' ? sentenceCase(children) : children}</div>
}

export function FormField({ label, children, hint }: { label: string; children: ReactNode; hint?: string }) {
  return <label className="form-field"><span>{sentenceCase(label)}</span>{children}{hint ? <small>{sentenceCase(hint)}</small> : null}</label>
}

export function FormActions({ children }: { children: ReactNode }) {
  return <div className="form-actions">{children}</div>
}

export function SmallAction({ children, onClick }: { children: ReactNode; onClick: () => void }) {
  return <button className="small-action" type="button" onClick={onClick}><Plus size={15} aria-hidden="true" />{typeof children === 'string' ? sentenceCase(children) : children}</button>
}

export function FieldInput(props: InputHTMLAttributes<HTMLInputElement>) {
  const placeholder = typeof props.placeholder === 'string' ? sentenceCase(props.placeholder) : props.placeholder
  return <input className="form-control" {...props} placeholder={placeholder} />
}

export function FieldSelect(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className="form-control" {...props} />
}

export function FieldTextarea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const placeholder = typeof props.placeholder === 'string' ? sentenceCase(props.placeholder) : props.placeholder
  return <textarea className="form-control" rows={3} {...props} placeholder={placeholder} />
}
