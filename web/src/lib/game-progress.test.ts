import { describe, expect, it } from 'vitest'

import {
  internalYear,
  ownerName,
  remainingTurns,
  remainingYears,
} from '@/lib/game-progress'
import type { StateData } from '@/types'

function stateWith(overrides: Partial<StateData> = {}): StateData {
  return {
    turn: 1,
    season: 'spring',
    players: [
      { id: 'P1', name: 'One', color: '#a84632' },
      { id: 'P2', name: 'Two', color: '#2d5f9e' },
    ],
    territories: [],
    nobles: [],
    ...overrides,
  }
}

describe('internalYear', () => {
  it('derives the year from the turn when absent', () => {
    expect(internalYear(stateWith({ turn: 1 }))).toBe(1)
    expect(internalYear(stateWith({ turn: 5 }))).toBe(2)
  })

  it('clamps the year to the configured length for finished games', () => {
    expect(internalYear(stateWith({ turn: 41, finished: true, yearCount: 10 }))).toBe(10)
  })
})

describe('remainingYears and remainingTurns', () => {
  it('counts down the remaining years and turns', () => {
    const state = stateWith({ turn: 5, yearCount: 10 })
    expect(remainingYears(state)).toBe(9)
    expect(remainingTurns(state)).toBe(36)
  })

  it('uses the fallback year count when the state omits it', () => {
    const state = stateWith({ turn: 3 })
    expect(remainingYears(state)).toBe(10)
    expect(remainingYears(state, 6)).toBe(6)
  })

  it('returns zero remaining years once the game is finished', () => {
    const state = stateWith({ finished: true, yearCount: 10 })
    expect(remainingYears(state)).toBe(0)
  })
})

describe('ownerName', () => {
  it('prefers the preferred player list names', () => {
    const state = stateWith()
    expect(ownerName('P1', state, [{ id: 'P1', name: 'Alice' }])).toBe('Alice')
  })

  it('falls back to the state player name', () => {
    expect(ownerName('P2', stateWith())).toBe('Two')
  })

  it('falls back to the provided fallback then the owner id', () => {
    const state = stateWith({ players: [] })
    expect(ownerName('P9', state, [], 'Unknown')).toBe('Unknown')
    expect(ownerName('P9', state)).toBe('P9')
    expect(ownerName(null, state, [], 'Nobody')).toBe('Nobody')
  })
})
