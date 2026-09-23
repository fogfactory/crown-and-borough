import { describe, expect, it } from 'vitest'

import { buildWinterIntentions, simulateWinterDraft } from '@/lib/winter-overlay'
import type { MapData, StateData, WinterCosts } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: true,
      points: [
        [0, 0],
        [50, 0],
        [50, 50],
        [0, 50],
      ],
      adjacencies: ['BRU'],
      impassable: [],
    },
    {
      id: 'BRU',
      name: 'Brisecote',
      terrain: 'forest',
      village: false,
      points: [
        [50, 0],
        [100, 0],
        [100, 50],
        [50, 50],
      ],
      adjacencies: ['ROS', 'CHA'],
      impassable: [],
    },
    {
      id: 'CHA',
      name: 'Champborne',
      terrain: 'hill',
      village: false,
      points: [
        [50, 50],
        [100, 50],
        [100, 100],
        [50, 100],
      ],
      adjacencies: ['BRU'],
      impassable: [],
    },
  ],
}

const baseState: StateData = {
  turn: 1,
  season: 'winter',
  players: [
    { id: 'P1', name: 'One', color: '#a84632', capitalTerritory: 'ROS' },
    { id: 'P2', name: 'Two', color: '#2d5f9e' },
  ],
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 20,
      army: null,
      infrastructures: [{ type: 'village', level: 1 }],
    },
    { id: 'BRU', owner: 'P1', resources: 0, army: null, infrastructures: [] },
    { id: 'CHA', owner: 'P1', resources: 0, army: null, infrastructures: [] },
  ],
  nobles: [
    {
      id: 'N1',
      code: 'HUG',
      name: 'Hugues',
      owner: 'P1',
      location: 'BRU',
      status: 'free',
    },
  ],
}

describe('winter overlay validation', () => {
  const costs: WinterCosts = {
    castle: 5,
    millLevels: [3, 5, 7],
    troop: 1,
    noble: 4,
    supplyDepot: 2,
    liberation: 3,
  }

  const poorState: StateData = {
    ...baseState,
    territories: baseState.territories.map((territory) => ({
      ...territory,
      resources: 1,
    })),
  }

  it('simulates a castle, troop, and noble in order', () => {
    const intentions = buildWinterIntentions(
      map,
      baseState,
      'P1',
      'C C ROS\nR T ROS\nR N ROS',
    )

    expect(intentions.map(({ kind, valid }) => ({ kind, valid }))).toEqual([
      { kind: 'build', valid: true },
      { kind: 'recruit_troop', valid: true },
      { kind: 'recruit_noble', valid: true },
    ])
  })

  it('allows a troop-created army to recruit a noble afterwards', () => {
    const result = simulateWinterDraft(baseState, 'P1', 'R T ROS\nR N ROS', map)

    expect(result.outcomes.map(({ valid, reason }) => ({ valid, reason }))).toEqual([
      { valid: true, reason: undefined },
      { valid: true, reason: undefined },
    ])
  })

  it('marks noble recruitment without a player army as invalid', () => {
    const [intention] = buildWinterIntentions(map, baseState, 'P1', 'R N ROS')

    expect(intention).toMatchObject({
      kind: 'error',
      valid: false,
      reason: 'noble_requires_owned_army',
      territory: 'ROS',
    })
  })

  it('marks troop recruitment without an adjacent free noble as invalid', () => {
    const state = {
      ...baseState,
      nobles: baseState.nobles.map((noble) => ({ ...noble, location: 'ROS' })),
    }
    const [intention] = buildWinterIntentions(map, state, 'P1', 'R T CHA')

    expect(intention).toMatchObject({
      kind: 'error',
      valid: false,
      reason: 'troop_requires_adjacent_noble',
      territory: 'CHA',
    })
  })

  it('marks an orphaned mill as invalid', () => {
    const [intention] = buildWinterIntentions(map, baseState, 'P1', 'C M CHA')

    expect(intention).toMatchObject({
      kind: 'error',
      valid: false,
      reason: 'mill_requires_productive_neighbor',
      territory: 'CHA',
    })
  })

  it('warns without rejecting an order that exceeds available resources', () => {
    const state: StateData = {
      ...poorState,
      territories: poorState.territories.map((territory, index) =>
        index === 0
          ? { ...territory, army: { owner: 'P1', size: 1, chain: null } }
          : territory,
      ),
    }
    const [intention] = buildWinterIntentions(map, state, 'P1', 'R N ROS', { costs })

    expect(intention).toMatchObject({
      kind: 'recruit_noble',
      valid: true,
      warning: true,
      reason: 'insufficient_resources',
      territory: 'ROS',
    })
  })

  it('keeps warned structural effects for later orders', () => {
    const result = simulateWinterDraft(
      poorState,
      'P1',
      'C C BRU\nR T BRU\nR N BRU',
      map,
      costs,
    )

    expect(
      result.outcomes.map(({ valid, warning, reason }) => ({ valid, warning, reason })),
    ).toEqual([
      { valid: true, warning: true, reason: 'insufficient_resources' },
      { valid: true, warning: false, reason: undefined },
      { valid: true, warning: true, reason: 'insufficient_resources' },
    ])
  })

  it('keeps syntax errors in the overlay', () => {
    const intentions = buildWinterIntentions(
      map,
      baseState,
      'P1',
      'C M ROS EXTRA\nR T ZZZ',
    )

    expect(intentions).toHaveLength(2)
    expect(intentions[0]).toMatchObject({
      kind: 'error',
      line: 1,
      territory: 'ROS',
      reason: 'error.winter.target_only_one',
    })
    expect(intentions[1]).toMatchObject({
      kind: 'error',
      line: 2,
      reason: 'error.winter.territory_unknown',
    })
  })
})
