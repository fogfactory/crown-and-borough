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
  production: [
    {
      territory: 'ROS',
      owner: 'P1',
      terrainRations: 1,
      infraRations: 2,
      baseProduction: 1,
      millProduction: 2,
      bonusProduction: 1,
      produced: 7,
      sentToRations: { BRU: 3 },
      stockBefore: 5,
      stockConsumed: 2,
      stockAfter: 3,
    },
    {
      territory: 'BRU',
      region: 'ROS',
      terrainRations: 1,
      suppressedRations: 2,
      produced: 1,
    },
  ],
  consumption: [
    {
      army: 'A1',
      owner: 'P1',
      territory: 'ROS',
      size: 2,
      demand: 2,
      receivedLocal: 2,
      receivedTransfer: 0,
      totalReceived: 2,
      missing: 0,
    },
    {
      army: 'A2',
      owner: 'P1',
      territory: 'BRU',
      source: 'ROS',
      size: 3,
      demand: 4,
      receivedLocal: 1,
      receivedTransfer: 0,
      totalReceived: 1,
      missing: 3,
      famine: true,
      troopsLost: 1,
      savedByPillage: true,
      pillageInfrastructure: 'mill',
      resourceCredit: 0,
    },
  ],
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
    {
      army: 'A1',
      chain: 'C1',
      order: 'O4',
      owner: 'P1',
      noble: 'JEA',
      type: 'attack',
      source: 'ROS',
      targets: ['CHA'],
      liaison: 'single',
      outcome: 'invalid',
      reason: 'bad_weather',
      progression: 'broken',
      indexBefore: 1,
      indexAfter: 1,
    },
  ],
  moves: [],
  nobles: [],
  cards: [
    {
      eventType: 'deck_order_played',
      kind: 'fair_weather',
      player: 'P1',
      region: 'ROS',
      season: 'spring',
      outcome: 'success',
    },
    {
      eventType: 'calamity_scheduled',
      kind: 'plague',
      region: 'ROS',
      season: 'summer',
      outcome: 'success',
    },
  ],
  seasonEffects: [
    { kind: 'calamity_applied', cardKind: 'plague', region: 'ROS', season: 'spring' },
    {
      kind: 'calamity_applied',
      cardKind: 'plague',
      region: 'ROS',
      season: 'spring',
      owner: 'P1',
      army: 'A1',
      territory: 'ROS',
      sizeBefore: 5,
      sizeAfter: 2,
    },
    { kind: 'plague_noble_death', cardKind: 'plague', region: 'ROS', season: 'spring', noble: 'ROB', territory: 'ROS' },
    { kind: 'plague_noble_survived', cardKind: 'plague', region: 'ROS', season: 'spring', noble: 'JEA', territory: 'ROS' },
    {
      kind: 'bad_weather_blocked',
      cardKind: 'bad_weather',
      region: 'ROS',
      season: 'spring',
      owner: 'P1',
      territory: 'ROS',
      target: 'BRU',
    },
    { kind: 'calamity_applied', cardKind: 'famine', region: 'ROS', season: 'spring' },
    { kind: 'famine_loss', cardKind: 'famine', region: 'ROS', season: 'spring', productionLost: 2, rationsLost: 2 },
    { kind: 'famine_loss', cardKind: 'famine', region: 'ROS', season: 'spring', territory: 'BRU', productionLost: 2 },
    { kind: 'bonus_effect', cardKind: 'fair_weather', region: 'ROS', season: 'spring' },
  ],
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
  rumors: [{ kind: 'fair_weather', key: 'rumor.fair_weather.level2', level: 2 }],
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
    expect(screen.getAllByText('Invalidé')).not.toHaveLength(0)
    expect(screen.getByText(/Bloqué par le mauvais temps/)).toBeInTheDocument()
    expect(screen.getAllByText('C M ROS')).toHaveLength(2)
    expect(screen.getByText(/Niveau 2/)).toBeInTheDocument()
    expect(screen.getByText('coût : 3 R')).toBeInTheDocument()
    expect(screen.getByText('aucun coût')).toBeInTheDocument()
    expect(screen.queryByText('coût : 0 R')).not.toBeInTheDocument()
    expect(screen.getByText(/Ressources insuffisantes/)).toBeInTheDocument()
    expect(screen.getAllByLabelText('Couleur de One')).not.toHaveLength(0)
    expect(screen.getByText(/7 produits/)).toBeInTheDocument()
    expect(screen.getByText(/production de base 1/)).toBeInTheDocument()
    expect(screen.getByText(/moulins 2/)).toBeInTheDocument()
    expect(screen.getByText(/bonus régional 1/)).toBeInTheDocument()
    expect(screen.getByText(/stock : 5 → 3 \(consommé 2\)/)).toBeInTheDocument()
    expect(screen.getByText(/envoyé 3 vers BRU/)).toBeInTheDocument()
    expect(screen.getByText(/2 supprimés par la mauvaise récolte/)).toBeInTheDocument()
    expect(screen.getByText(/demande 2/)).toBeInTheDocument()
    expect(screen.getByText(/local 2 · sources 0 · reçu 2 · manque 0/)).toBeInTheDocument()
    expect(screen.getByText(/source ROS · local 1 · sources 0 · reçu 1 · manque 3/)).toBeInTheDocument()
    expect(screen.getByText(/· sauvée par pillage/)).toBeInTheDocument()
    expect(screen.getByText(/· perd 1 troupe/)).toBeInTheDocument()
    expect(
      screen.getByText(/Un bel ensoleillement gagne le royaume/),
    ).toBeInTheDocument()
    expect(screen.getByText(/Beau temps \(BT\) jouée sur ROS/)).toBeInTheDocument()
    expect(screen.getByText(/Peste \(PE\) à venir en Été dans ROS/)).toBeInTheDocument()
    expect(screen.getByText(/Mauvaise récolte \(MR\) active dans ROS/)).toBeInTheDocument()
    expect(screen.getByText(/Beau temps \(BT\) actif dans ROS/)).toBeInTheDocument()
    expect(screen.getByText('Peste (PE) active dans ROS')).toBeInTheDocument()
    expect(screen.getByText(/Armée de P1 à ROS : 5 → 2 troupes/)).toBeInTheDocument()
    expect(screen.getByText('Le noble ROB meurt de la peste à ROS')).toBeInTheDocument()
    expect(screen.getByText('Le noble JEA à ROS survit à la peste')).toBeInTheDocument()
    expect(screen.getByText(/Armée de P1 à ROS : mouvement vers BRU bloqué/)).toBeInTheDocument()
    expect(screen.getByText(/2 R de production supprimées, 2 rations/)).toBeInTheDocument()
    expect(screen.getByText(/Moulin à BRU désactivé : 2 R non produites/)).toBeInTheDocument()
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

  it('localizes the mill maximum-level rejection reason', () => {
    const capReport: TurnReport = {
      ...report,
      winter: {
        investments: [
          {
            kind: 'rejected',
            player: 'P1',
            outcome: 'failure',
            cost: 0,
            territory: 'ROS',
            reason: 'mill_max_level_reached',
            order: { type: 'build', territory: 'ROS', infrastructureType: 'mill' },
          },
        ],
        stocks: [],
      },
    }

    render(
      <LanguageProvider initialLanguage="fr">
        <ReportPanel report={capReport} map={map} players={players} />
      </LanguageProvider>,
    )

    expect(
      screen.getByText(/Le moulin a atteint son niveau maximal\./),
    ).toBeInTheDocument()
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

  it('renders a general combat without exposing forces', () => {
    const generalReport: TurnReport = {
      ...report,
      combats: [
        {
          visibility: 'general',
          territory: 'BRU',
          outcome: 'standoff',
          summary: 'The combat ended without a winner.',
        },
      ],
    }

    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <ReportPanel report={generalReport} map={map} players={players} />
      </LanguageProvider>,
    )

    expect(screen.getByText('The combat ended without a winner.')).toBeInTheDocument()
    expect(container.textContent).not.toContain('2 (+1 noble)')
  })

  it('tolerates null report collections from older server responses', () => {
    const sparseReport = {
      ...report,
      receptions: null,
      production: null,
      consumption: null,
      combats: null,
      orders: null,
    } as unknown as TurnReport

    render(
      <LanguageProvider initialLanguage="en">
        <ReportPanel report={sparseReport} map={map} players={players} />
      </LanguageProvider>,
    )

    expect(screen.getByText('Resolution complete')).toBeInTheDocument()
    expect(screen.getByText('No chain reception events.')).toBeInTheDocument()
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
