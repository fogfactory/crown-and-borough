import { describe, expect, it } from 'vitest'

import { CARD_ICONS } from '@/lib/game-icons'
import {
  calamityIconItems,
  cardIconItems,
  countCancelingCards,
  fullyCanceledCalamities,
  singleCanceledRegionSeeds,
} from '@/lib/region-effect-icons'
import type { ActiveRegionEffect, Region, Territory } from '@/types'

function square(id: string, x: number): Territory {
  return {
    id,
    name: id,
    terrain: 'plain',
    village: false,
    points: [
      [x, 0],
      [x + 100, 0],
      [x + 100, 100],
      [x, 100],
    ],
    adjacencies: [],
    impassable: [],
  }
}

const territories = [square('ROS', 0), square('BRU', 100), square('VAL', 200)]
const regions: Region[] = [
  { id: 'r1', seed: 'ROS', territories: ['ROS', 'BRU'] },
  { id: 'r2', seed: 'VAL', territories: ['VAL'] },
]
const effects: ActiveRegionEffect[] = [
  { kind: 'bad_weather', regionSeed: 'ROS', season: 'summer', year: 1 },
  { kind: 'famine', regionSeed: 'VAL', season: 'summer', year: 1 },
]
const players = [{ id: 'P1', name: 'One', color: '#123456' }]

describe('canceling cards', () => {
  it('counts drafted canceling cards per canceled calamity and region', () => {
    const counts = countCancelingCards([
      { player: 'P1', text: 'P BT ROS\nP RA VAL' },
      { player: 'P2', text: 'P FW ROS\nP RE BRU' },
    ])
    expect(counts).toEqual(
      new Map([
        ['bad_weather-ROS', 2],
        ['famine-VAL', 1],
      ]),
    )
  })

  it('clears calamities met by two cards and badges those met by one', () => {
    const counts = new Map([
      ['bad_weather-ROS', 2],
      ['famine-VAL', 1],
      ['famine-ROS', 2],
    ])
    expect(fullyCanceledCalamities(counts, effects)).toEqual(new Set(['bad_weather-ROS']))
    expect(singleCanceledRegionSeeds(counts, effects)).toEqual(new Set(['VAL']))
    expect(fullyCanceledCalamities(counts, undefined)).toEqual(new Set())
  })
})

describe('calamityIconItems', () => {
  it('scatters over every territory of the region, skipping cleared calamities', () => {
    const icons = calamityIconItems(
      effects,
      regions,
      territories,
      new Set(['famine-VAL']),
      new Set(['ROS']),
    )
    expect(icons.length).toBeGreaterThan(0)
    expect(icons.every((icon) => icon.key.startsWith('bad_weather-ROS-'))).toBe(true)
    expect(new Set(icons.map((icon) => icon.key.split('-')[2]))).toEqual(
      new Set(['ROS', 'BRU']),
    )
    expect(icons.every((icon) => icon.canceled)).toBe(true)
  })
})

describe('cardIconItems', () => {
  it('colors revolts with the drafting player and scatters bonuses once per region', () => {
    const icons = cardIconItems(
      [{ player: 'P1', text: 'P RE BRU\nP BT VAL\nP BT VAL' }],
      regions,
      territories,
      [],
      players,
      new Set(),
    )
    const revolts = icons.filter((icon) => icon.key.startsWith('revolt-BRU-'))
    expect(revolts).toHaveLength(CARD_ICONS.revolt.count)
    expect(revolts.every((icon) => icon.fill === '#123456')).toBe(true)
    const suns = icons.filter((icon) => icon.key.startsWith('fair_weather-VAL-'))
    expect(suns).toHaveLength(CARD_ICONS.fair_weather.count)
  })

  it('shows the bonus only once the calamity it cancels is fully cleared', () => {
    const draft = [{ player: 'P1', text: 'P BT ROS' }]
    expect(
      cardIconItems(draft, regions, territories, effects, players, new Set()),
    ).toEqual([])
    const cleared = cardIconItems(
      draft,
      regions,
      territories,
      effects,
      players,
      new Set(['bad_weather-ROS']),
    )
    expect(cleared).toHaveLength(2 * CARD_ICONS.fair_weather.count)
  })
})
