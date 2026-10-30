import { useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, Droplets, MoonStar } from 'lucide-react'
import { PageHeading, SectionCard } from '@/features/painel/components/page-ui'

type CalendarEvent = { title: string; detail: string }
const eventsByDay: Record<number, CalendarEvent> = {
  9: { title: 'Pulverização', detail: 'Setor norte · óleo mineral' },
  16: { title: 'Pulverização', detail: 'Setor sul · cobre agrícola' },
  22: { title: 'Pulverização', detail: 'Setor norte · óleo mineral' },
  30: { title: 'Planejada', detail: 'Revisar atividade · setor norte' },
}
const moonByDay: Record<number, string> = { 1: 'Lua cheia', 9: 'Quarto minguante', 16: 'Lua nova', 23: 'Quarto crescente', 31: 'Lua cheia' }
const transitions = new Set([1, 9, 16, 23, 31])
const weekdays = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

export function CalendarioPage() {
  const [month, setMonth] = useState(new Date(2026, 9, 1))
  const [selectedDay, setSelectedDay] = useState(30)
  const year = month.getFullYear()
  const monthIndex = month.getMonth()
  const monthName = month.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' })
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate()
  const firstWeekday = new Date(year, monthIndex, 1).getDay()
  const cells = useMemo(() => [...Array(firstWeekday).fill(null), ...Array.from({ length: daysInMonth }, (_, index) => index + 1)], [daysInMonth, firstWeekday])
  // As datas de demonstração representam somente outubro; outros meses ficam sem eventos ou fases inventados.
  const isSampleMonth = year === 2026 && monthIndex === 9
  const selectedEvent = isSampleMonth ? eventsByDay[selectedDay] : undefined
  const selectedMoon = isSampleMonth ? moonByDay[selectedDay] : undefined

  function changeMonth(direction: number) {
    setMonth(current => new Date(current.getFullYear(), current.getMonth() + direction, 1))
    setSelectedDay(1)
  }

  return <>
    <PageHeading eyebrow="Manejo · Calendário" title="Calendário do pomar" description="Atividades e fases da Lua." />
    <div className="calendar-page-grid">
      <SectionCard title={monthName} action={<div className="calendar-controls"><button onClick={() => changeMonth(-1)} aria-label="Mês anterior"><ChevronLeft size={17} /></button><button onClick={() => changeMonth(1)} aria-label="Próximo mês"><ChevronRight size={17} /></button></div>}>
        <div className="calendar-legend"><span><i className="event-legend" /> Atividade</span><span><i className="moon-legend">◐</i> Fase da Lua</span></div>
        <div className="month-grid"><div className="weekdays">{weekdays.map(day => <span key={day}>{day}</span>)}</div><div className="month-days">{cells.map((day, index) => {
          if (day === null) return <span className="calendar-empty" key={`empty-${index}`} />
          const event = isSampleMonth ? eventsByDay[day] : undefined
          const moon = isSampleMonth ? moonByDay[day] : undefined
          const hasTransition = isSampleMonth && transitions.has(day)
          return <button key={day} className={`calendar-day ${day === selectedDay ? 'selected-day' : ''} ${day === 30 && isSampleMonth ? 'today-day' : ''} ${event ? 'has-event' : ''}`} onClick={() => setSelectedDay(day)} aria-pressed={day === selectedDay} aria-label={`${day}${event ? `, ${event.title}` : ''}${moon ? `, ${moon}` : ''}${hasTransition ? ', mudança de fase da Lua' : ''}`}><span className="day-number">{day}</span>{moon ? <span className="day-moon" aria-hidden="true">◐</span> : null}{event ? <span className="day-event"><Droplets size={12} /> <span>{event.title}</span></span> : null}{hasTransition ? <span className="transition-mark" aria-hidden="true">◦</span> : null}</button>
        })}</div></div>
      </SectionCard>
      <aside className="calendar-side">
        <SectionCard title={`${selectedDay} de ${month.toLocaleDateString('pt-BR', { month: 'long' })}`}>
          {selectedEvent ? <div className="calendar-event-detail"><span className="event-detail-icon"><Droplets size={18} /></span><div><strong>{selectedEvent.title}</strong><p>{selectedEvent.detail}</p></div></div> : <div className="calendar-no-event"><strong>Sem atividade</strong></div>}
          {selectedMoon ? <div className="selected-moon"><MoonStar size={18} /><div><small>Exemplo</small><strong>{selectedMoon}</strong>{transitions.has(selectedDay) ? <span>Mudança de fase</span> : null}</div></div> : null}
        </SectionCard>
      </aside>
    </div>
  </>
}
