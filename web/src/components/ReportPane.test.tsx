import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { ReportPane, type ReportSummary } from '@/components/ReportPane'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { MapData, TurnReport } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: false,
      points: [],
      adjacencies: [],
      impassable: [],
    },
  ],
}

const emptyReport: TurnReport = {
  header: { year: 1, season: 'spring', turn: 1 },
  players: [],
  receptions: [],
  supply: [],
  famines: [],
  combats: [],
  orders: [],
  moves: [],
  nobles: [],
}

const summaries: ReportSummary[] = [
  { index: 0, header: { year: 1, season: 'spring', turn: 1 } },
  { index: 1, header: { year: 1, season: 'summer', turn: 2 } },
]

function renderPane(props: Parameters<typeof ReportPane>[0]) {
  return render(
    <LanguageProvider initialLanguage="en">
      <ReportPane {...props} />
    </LanguageProvider>,
  )
}

describe('ReportPane', () => {
  it('shows the empty state when no report is available', () => {
    renderPane({ report: null, map, players: [] })

    expect(screen.getByText('No report available')).toBeInTheDocument()
  })

  it('renders the report panel when a report is available', () => {
    renderPane({ report: emptyReport, map, players: [] })

    expect(screen.getByText('Turn report 1')).toBeInTheDocument()
  })

  it('offers the turn history and loads a selected report', () => {
    const onSelectReport = vi.fn()
    renderPane({
      report: emptyReport,
      map,
      players: [],
      summaries,
      onSelectReport,
    })

    expect(screen.getByText('Turn history')).toBeInTheDocument()
    expect(screen.getByText('Turn 1 · Spring')).toBeInTheDocument()
    expect(screen.getByText('Turn 2 · Summer')).toBeInTheDocument()

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '1' } })
    expect(onSelectReport).toHaveBeenCalledWith(1)
  })

  it('shows the loading message while a report is being fetched', () => {
    renderPane({ report: null, map, players: [], summaries, loading: true })

    expect(screen.getByText('Loading...')).toBeInTheDocument()
    expect(screen.queryByText('No report available')).not.toBeInTheDocument()
  })

  it('surfaces a report loading error', () => {
    renderPane({ report: null, map, players: [], error: 'boom' })

    const alert = screen.getByRole('alert')
    expect(alert).toHaveTextContent('boom')
  })
})
