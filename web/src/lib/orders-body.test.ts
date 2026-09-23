import { describe, expect, it } from 'vitest'

import { buildOrdersBody, isEmptyOrdersBody } from '@/lib/orders-body'
import type { StateData } from '@/types'

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [],
  nobles: [
    {
      id: 'N1',
      code: 'HUG',
      name: 'Hugues',
      owner: 'P1',
      location: 'ROS',
      status: 'free',
    },
    {
      id: 'N2',
      code: 'JEA',
      name: 'Jean',
      owner: 'P1',
      location: 'ROS',
      status: 'dungeon',
    },
    { id: 'N3', code: 'BOB', name: 'Bob', owner: 'P2', location: 'BRU', status: 'free' },
  ],
}

describe('buildOrdersBody', () => {
  it('sends one headed chain per noble able to emit and the card orders', () => {
    const body = buildOrdersBody(state, 'P1', {
      chainDrafts: { HUG: 'ROS A BRU', JEA: 'ROS H', BOB: 'BRU H' },
      winterDraft: 'R T ROS',
      specialDraft: 'P BH ROS',
    })

    expect(body).toEqual({
      chains: [{ noble: 'HUG', text: 'HUG\nROS A BRU' }],
      winter: [],
      special: [{ text: 'P BH ROS' }],
    })
  })

  it('puts card discards on the winter sheet in winter', () => {
    const body = buildOrdersBody({ ...state, season: 'winter' }, 'P1', {
      chainDrafts: { HUG: 'ROS A BRU' },
      winterDraft: 'R T ROS',
      specialDraft: 'D C GR',
    })

    expect(body).toEqual({
      chains: [],
      winter: [{ lines: 'R T ROS\nD C GR' }],
      special: [],
    })
  })

  it('detects an empty draft', () => {
    const body = buildOrdersBody(state, 'P1', {
      chainDrafts: { HUG: '   ' },
      winterDraft: '',
      specialDraft: '',
    })

    expect(isEmptyOrdersBody(body)).toBe(true)
  })
})
