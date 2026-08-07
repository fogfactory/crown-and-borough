import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ReportPanel } from '@/components/ReportPanel'
import type { MapData, StateData, TurnReport } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'T1',
      code: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: true,
      points: [],
      adjacencies: [],
      impassable: [],
    },
    {
      id: 'T2',
      code: 'BRU',
      name: 'Bruyères',
      terrain: 'forest',
      village: false,
      points: [],
      adjacencies: [],
      impassable: [],
    },
    {
      id: 'T3',
      code: 'CHA',
      name: 'Chavaux',
      terrain: 'hill',
      village: false,
      points: [],
      adjacencies: [],
      impassable: [],
    },
  ],
}

const report: TurnReport = {
  header: { year: 1, season: 'spring', turn: 1 },
  players: [
    {
      id: 'P1',
      name: 'One',
      resourcesBefore: 2,
      resourcesAfter: 1,
      controlledBefore: 1,
      controlledAfter: 1,
      armies: [{ id: 'A1', owner: 'P1', territory: 'T1', size: 2 }],
      nobles: [],
      infrastructures: [],
    },
  ],
  receptions: [],
  supply: [
    {
      source: 'T1',
      owner: '',
      production: 1,
      demand: 0,
      rations: {},
      stockConsumed: 0,
      stockAfter: 3,
    },
  ],
  famines: [],
  combats: [],
  orders: [
    {
      army: 'A1',
      chain: 'C1',
      order: 'O1',
      owner: 'P1',
      noble: 'JEA',
      type: 'support',
      source: 'T1',
      targets: ['T2', 'T3'],
      liaison: 'loop',
      outcome: 'success',
      progression: 'advanced',
      indexBefore: 0,
      indexAfter: 1,
    },
    {
      army: 'A1',
      chain: 'C1',
      order: 'O2',
      owner: 'P1',
      noble: 'JEA',
      type: 'attack',
      source: 'T1',
      targets: ['T2'],
      liaison: 'single',
      outcome: 'invalid',
      reason: 'non_adjacent_destination',
      progression: 'broken',
      indexBefore: 1,
      indexAfter: 1,
    },
    {
      army: 'A1',
      chain: 'C1',
      order: 'O3',
      owner: 'P1',
      noble: 'JEA',
      type: 'hold',
      source: 'T1',
      liaison: 'single',
      outcome: 'failure',
      reason: 'allied_destination',
      progression: 'broken',
      indexBefore: 1,
      indexAfter: 1,
    },
  ],
  moves: [],
  nobles: [],
  winter: {
    investments: [
      {
        kind: 'rejected',
        player: 'P1',
        outcome: 'failure',
        cost: 0,
        territory: 'T1',
        reason: 'insufficient_resources',
        order: { type: 'build', territory: 'T1', infrastructureType: 'mill' },
      },
      {
        kind: 'upgrade',
        player: 'P1',
        outcome: 'success',
        cost: 3,
        territory: 'T1',
        type: 'mill',
        level: 2,
      },
      {
        kind: 'capital_elected',
        player: 'P1',
        outcome: 'success',
        cost: 0,
        territory: 'T1',
      },
    ],
    stocks: [],
  },
}

const players: StateData['players'] = [{ id: 'P1', name: 'One', color: '#a84632' }]

describe('ReportPanel', () => {
  it('renders complete order labels, ownership, noble, outcomes, and winter labels', () => {
    render(<ReportPanel report={report} map={map} players={players} />)

    expect(screen.getByText('(ROS S BRU - CHA)')).toBeInTheDocument()
    expect(screen.getAllByText('P1 · Noble JEA')).not.toHaveLength(0)
    expect(screen.getAllByText('Réussi')).not.toHaveLength(0)
    expect(screen.getAllByText('Échec')).not.toHaveLength(0)
    expect(screen.getByText('Invalidé')).toBeInTheDocument()
    expect(screen.getAllByText('C M ROS')).toHaveLength(2)
    expect(screen.getByText(/Niveau 2/)).toBeInTheDocument()
    expect(screen.getByText('coût : 3 R')).toBeInTheDocument()
    expect(screen.getByText('aucun coût')).toBeInTheDocument()
    expect(screen.queryByText('coût : 0 R')).not.toBeInTheDocument()
    expect(screen.getByText(/insufficient_resources/)).toBeInTheDocument()
    expect(screen.getAllByLabelText('Couleur de One')).not.toHaveLength(0)
    expect(screen.getByText('ROS · Neutre')).toBeInTheDocument()
    expect(screen.getByText(/3 R en stock/)).toBeInTheDocument()
  })

  it('does not display storage identifiers in visible report text', () => {
    const { container } = render(
      <ReportPanel report={report} map={map} players={players} />,
    )
    const text = container.textContent ?? ''

    expect(text).not.toMatch(/T\d+/)
    expect(text).not.toMatch(/A\d+/)
    expect(text).not.toMatch(/C\d+/)
    expect(text).not.toMatch(/O\d+/)
  })
})
