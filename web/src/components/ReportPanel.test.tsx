import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ReportPanel } from '@/components/ReportPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { MapData, StateData, TurnReport } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: true,
      points: [],
      adjacencies: [],
      impassable: [],
    },
    {
      id: 'BRU',
      name: 'Bruyères',
      terrain: 'forest',
      village: false,
      points: [],
      adjacencies: [],
      impassable: [],
    },
    {
      id: 'CHA',
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
      armies: [{ id: 'A1', owner: 'P1', territory: 'ROS', size: 2 }],
      nobles: [],
      infrastructures: [],
    },
  ],
  receptions: [],
  supply: [
    {
      source: 'ROS',
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
      source: 'ROS',
      targets: ['BRU', 'CHA'],
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
      source: 'ROS',
      targets: ['BRU'],
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
      source: 'ROS',
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
        territory: 'ROS',
        reason: 'insufficient_resources',
        order: { type: 'build', territory: 'ROS', infrastructureType: 'mill' },
      },
      {
        kind: 'upgrade',
        player: 'P1',
        outcome: 'success',
        cost: 3,
        territory: 'ROS',
        type: 'mill',
        level: 2,
      },
      {
        kind: 'capital_elected',
        player: 'P1',
        outcome: 'success',
        cost: 0,
        territory: 'ROS',
      },
    ],
    stocks: [],
  },
}

const players: StateData['players'] = [{ id: 'P1', name: 'One', color: '#a84632' }]

describe('ReportPanel', () => {
  it('renders complete order labels, ownership, noble, outcomes, and winter labels', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <ReportPanel report={report} map={map} players={players} />
      </LanguageProvider>,
    )

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
    expect(screen.getByText(/Ressources insuffisantes/)).toBeInTheDocument()
    expect(screen.getAllByLabelText('Couleur de One')).not.toHaveLength(0)
    expect(screen.getByText('ROS · Neutre')).toBeInTheDocument()
    expect(screen.getByText(/3 R en stock/)).toBeInTheDocument()
  })

  it('does not display storage identifiers in visible report text', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="fr">
        <ReportPanel report={report} map={map} players={players} />
      </LanguageProvider>,
    )
    const text = container.textContent ?? ''

    expect(text).not.toMatch(/T\d+/)
    expect(text).not.toMatch(/A\d+/)
    expect(text).not.toMatch(/C\d+/)
    expect(text).not.toMatch(/O\d+/)
  })

  it('displays the noble command bonus in combat forces', () => {
    const combatReport: TurnReport = {
      ...report,
      combats: [
        {
          territory: 'BRU',
          baseDefense: 1,
          defense: 1,
          castleBonus: 0,
          contenders: [
            { army: 'A1', owner: 'P1', force: 2, nobleBonus: 1, defender: false },
          ],
          cutSupporters: [],
          reason: 'attack_wins',
          standoff: false,
        },
      ],
    }

    render(
      <LanguageProvider initialLanguage="fr">
        <ReportPanel report={combatReport} map={map} players={players} />
      </LanguageProvider>,
    )

    expect(screen.getByText('2 (+1 noble)')).toBeInTheDocument()
  })

  it('renders the same report in English with translated reason labels', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <ReportPanel report={report} map={map} players={players} />
      </LanguageProvider>,
    )

    expect(screen.getByText('Resolution complete')).toBeInTheDocument()
    expect(screen.getAllByText('Success')).not.toHaveLength(0)
    expect(screen.getByText(/Insufficient resources/)).toBeInTheDocument()
    expect(screen.getByText('The destination is not adjacent.')).toBeInTheDocument()
  })
})
