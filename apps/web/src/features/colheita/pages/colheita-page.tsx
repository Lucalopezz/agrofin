import { useState } from 'react'
import { PageHeading } from '@/features/painel/components/page-ui'
import { HarvestForm } from '../components/harvest-form'
import { HarvestHistory } from '../components/harvest-history'
import { HarvestRates } from '../components/harvest-rates'
import { weekOf } from '../lib/entries'

export function ColheitaPage({ onOpenEmployees }: { onOpenEmployees: () => void }) {
  const [period, setPeriod] = useState(() => weekOf())
  return (
    <>
      <PageHeading
        eyebrow="Produção"
        title="Registrar colheita"
        description="Informe funcionário e caixas."
      />
      <div className="form-page-grid harvest-page-grid">
        <HarvestForm
          onOpenEmployees={onOpenEmployees}
          onSaved={(day) => {
            // Um lançamento retroativo deve aparecer no histórico após salvar.
            if (
              day < period.start_date ||
              day > period.end_date ||
              !period.start_date ||
              !period.end_date
            ) {
              setPeriod(weekOf(day))
            }
          }}
        />
        <HarvestRates />
        <HarvestHistory period={period} onChange={setPeriod} />
      </div>
    </>
  )
}
