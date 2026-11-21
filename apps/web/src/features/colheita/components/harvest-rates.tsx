import { SectionCard } from '@/features/painel/components/page-ui'
import { Button } from '@/shared/components/ui/button'
import { useHarvestRates } from '../hooks/use-harvest-rates'
import { displayDate, money } from '../lib/rates'
import { boxLabels } from '../types'
import { HarvestRateForm } from './harvest-rate-form'
import { RequestError } from './request-error'

export function HarvestRates() {
  const rates = useHarvestRates()
  return (
    <SectionCard title="Tarifas de colheita">
      <details>
        <summary>Consultar e cadastrar tarifas</summary>
        <HarvestRateForm />
        <h3>Tarifas cadastradas</h3>
        <Button
          type="button"
          variant="outline"
          disabled={rates.isFetching}
          onClick={() => void rates.refetch()}
        >
          Atualizar tarifas
        </Button>
        {rates.isPending && <p role="status">Carregando tarifas...</p>}
        <RequestError error={rates.error} retry={() => void rates.refetch()} />
        {rates.isSuccess &&
          (rates.data.length ? (
            <ul className="employee-list">
              {rates.data.map((rate) => (
                <li key={rate.id}>
                  <strong>
                    {boxLabels[rate.box_type]} · {money(rate.amount_cents)}
                  </strong>
                  <span>
                    Desde {displayDate(rate.valid_from)} ·{' '}
                    {rate.valid_until
                      ? `Fim exclusivo em ${displayDate(rate.valid_until)}`
                      : 'Sem fim definido'}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p>Nenhuma tarifa cadastrada.</p>
          ))}
      </details>
    </SectionCard>
  )
}
