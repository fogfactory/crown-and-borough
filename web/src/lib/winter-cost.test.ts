import { describe, expect, it } from 'vitest'

import { estimateWinterCost, isWinterCosts } from '@/lib/winter-cost'
import type { StateData, WinterCosts } from '@/types'

const costs: WinterCosts = {
  castle: 10,
  millLevels: [3, 5, 7],
  troop: 1,
  noble: 2,
  supplyDepot: 3,
  liberation: 0,
}

const state: StateData = {
  turn: 4,
  season: 'winter',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 25,
      army: null,
      infrastructures: [{ type: 'castle', level: 1 }],
    },
    {
      id: 'XXX',
      owner: 'P1',
      resources: 0,
      army: null,
      infrastructures: [],
    },
    {
      id: 'YYY',
      owner: 'P1',
      resources: 0,
      army: null,
      infrastructures: [],
    },
    {
      id: 'ZZZ',
      owner: 'P1',
      resources: 0,
      army: null,
      infrastructures: [],
    },
  ],
  nobles: [],
}

describe('estimateWinterCost', () => {
  it('sums configured costs and applies successive mill upgrade costs', () => {
    const estimate = estimateWinterCost(
      state,
      'P1',
      costs,
      'R T XXX\nR N XXX\nC C YYY\nC M ZZZ\nC M ZZZ',
    )

    expect(estimate).toEqual({ spent: 21, available: 25 })
  })

  it('does not include resources outside controlled settlements', () => {
    const estimate = estimateWinterCost(
      {
        ...state,
        territories: [
          ...state.territories,
          {
            id: 'OTH',
            owner: null,
            resources: 100,
            army: null,
            infrastructures: [{ type: 'village', level: 1 }],
          },
        ],
      },
      'P1',
      costs,
      'C C YYY',
    )

    expect(estimate.available).toBe(25)
  })

  it('validates the costs payload before using it', () => {
    expect(isWinterCosts(costs)).toBe(true)
    expect(isWinterCosts({ ...costs, troop: -1 })).toBe(false)
    expect(isWinterCosts({ ...costs, millLevels: ['3'] })).toBe(false)
  })
})
