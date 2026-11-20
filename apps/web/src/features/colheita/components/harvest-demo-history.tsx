import { SectionCard } from '@/features/painel/components/page-ui'

export type HarvestRecord = {
  person: string
  date: string
  normal: number
  moagem: number
  total: number
}

const currency = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })

export function HarvestDemoHistory({ rows }: { rows: HarvestRecord[] }) {
  return (
    <SectionCard title="Últimos registros">
      <div className="table-scroll">
        <table className="data-table">
          <thead>
            <tr>
              <th>Funcionário</th>
              <th>Data</th>
              <th>Normal</th>
              <th>Moagem</th>
              <th>Total devido · exemplo</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <tr key={`${row.person}-${index}`}>
                <td>
                  <span className="table-person">
                    <span>{row.person.split(' ').map((part) => part[0]).slice(0, 2).join('')}</span>
                    {row.person}
                  </span>
                </td>
                <td>{row.date}</td>
                <td>{row.normal} cx</td>
                <td>{row.moagem} cx</td>
                <td className="money-cell">{currency.format(row.total)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="table-mobile-list">
        {rows.map((row, index) => (
          <article className="mobile-record" key={`${row.person}-${index}`}>
            <div>
              <strong>{row.person}</strong>
              <small>{row.date} · {row.normal + row.moagem} caixas</small>
            </div>
            <strong>{currency.format(row.total)}</strong>
          </article>
        ))}
      </div>
    </SectionCard>
  )
}
