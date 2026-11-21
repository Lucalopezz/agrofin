import { HarvestRates } from '../components/harvest-rates'
import { useState } from 'react'
import { PageHeading } from '@/features/painel/components/page-ui'
import { HarvestDemoForm } from '../components/harvest-demo-form'
import { HarvestDemoHistory, type HarvestRecord } from '../components/harvest-demo-history'

const initialRows: HarvestRecord[] = [
  { person: 'Ana Paula', date: 'Hoje, 30 out', normal: 24, moagem: 4, total: 128 },
  { person: 'João Carlos', date: 'Hoje, 30 out', normal: 18, moagem: 0, total: 90 },
  { person: 'Raimundo Silva', date: 'Ontem, 29 out', normal: 21, moagem: 3, total: 111 },
]

export function ColheitaPage({ onOpenEmployees }: { onOpenEmployees: () => void }) {
  const [rows, setRows] = useState(initialRows)

  return (
    <>
      <PageHeading
        eyebrow="Produção"
        title="Registrar colheita"
        description="Informe funcionário e caixas."
      />
      <div className="form-page-grid harvest-page-grid">
        <HarvestDemoForm
          onOpenEmployees={onOpenEmployees}
          onSave={(record) => setRows((current) => [record, ...current])}
        />
      </div>
      <HarvestRates />
      <HarvestDemoHistory rows={rows} />
    </>
  )
}
