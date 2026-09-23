import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { useGameIntentions } from '@/lib/use-game-intentions'
import type { MapData, StateData, SubmittedOrdersResponse } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: false,
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
      adjacencies: ['ROS'],
      impassable: [],
    },
  ],
}

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [
    { id: 'P1', name: 'One', color: '#a84632' },
    { id: 'P2', name: 'Two', color: '#2d5f9e' },
  ],
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 0,
      army: { owner: 'P1', size: 2, chain: null },
      infrastructures: [],
    },
    { id: 'BRU', owner: 'P2', resources: 0, army: null, infrastructures: [] },
  ],
  nobles: [
    {
      id: 'N1',
      code: 'HUG',
      name: 'Hugues',
      owner: 'P1',
      location: 'ROS',
      status: 'free',
    },
  ],
}

function buildHook(overrides: Partial<Parameters<typeof useGameIntentions>[0]> = {}) {
  const props = {
    state,
    map,
    playerID: 'P1',
    chainDrafts: { HUG: 'ROS A BRU' },
    winterDraft: '',
    winterCosts: null,
    spectator: false,
    submittedOrders: null,
    ...overrides,
  }
  return renderHook(() => useGameIntentions(props))
}

describe('useGameIntentions', () => {
  it('builds the active player drafts and color', () => {
    const { result } = buildHook()

    expect(result.current.intentions).toHaveLength(1)
    expect(result.current.intentions[0]).toMatchObject({
      symbol: 'A',
      source: 'draft',
    })
    expect(result.current.winterIntentions).toHaveLength(0)
    expect(result.current.intentionsColor).toBe('#a84632')
  })

  it('builds winter investments for the active player in winter', () => {
    const { result } = buildHook({
      state: { ...state, season: 'winter' },
      chainDrafts: {},
      winterDraft: 'C C ROS',
    })

    expect(result.current.intentions).toHaveLength(0)
    expect(result.current.winterIntentions).toHaveLength(1)
    expect(result.current.winterIntentions[0]).toMatchObject({
      kind: 'build',
      territory: 'ROS',
      valid: true,
    })
  })

  it('colors submitted chains and winter orders for a spectator', () => {
    const submittedOrders: SubmittedOrdersResponse = {
      turn: 1,
      season: 'spring',
      submissions: [
        {
          player: 'P2',
          chains: [{ noble: 'BOB', text: 'BOB\nBRU A ROS' }],
          winter: { lines: 'R T BRU' },
        },
      ],
    }
    const winterState: StateData = {
      ...state,
      season: 'winter',
      territories: state.territories.map((territory) =>
        territory.id === 'BRU'
          ? {
              ...territory,
              army: {
                owner: 'P2',
                size: 2,
                chain: {
                  visibility: 'known',
                  currentIndex: 0,
                  orders: [
                    {
                      type: 'hold',
                      position: 'BRU',
                      targets: [],
                      liaison: 'loop',
                    },
                  ],
                },
              },
            }
          : territory,
      ),
      nobles: [
        ...state.nobles,
        {
          id: 'N2',
          code: 'BOB',
          name: 'Robert',
          owner: 'P2',
          location: 'BRU',
          status: 'free',
        },
      ],
    }
    const { result } = buildHook({
      state: winterState,
      chainDrafts: {},
      spectator: true,
      submittedOrders,
    })

    expect(result.current.intentions.length).toBeGreaterThanOrEqual(1)
    expect(
      result.current.intentions.some((intention) => intention.color === '#2d5f9e'),
    ).toBe(true)
    expect(result.current.winterIntentions).toHaveLength(1)
    expect(result.current.winterIntentions[0]).toMatchObject({
      kind: 'recruit_troop',
      color: '#2d5f9e',
      source: 'submitted',
    })
  })

  it('defaults the color when the player is unknown', () => {
    const { result } = buildHook({ playerID: null, chainDrafts: {} })

    expect(result.current.intentions).toHaveLength(0)
    expect(result.current.intentionsColor).toBe('#a84632')
  })
})
