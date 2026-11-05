import { describe, expect, it } from 'vitest'

import { buildOrdersBody, isEmptyOrdersBody, winterSheet } from '@/lib/orders-body'
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
      winterDraft: 'R T ROS\nC N HUG BOB\nR N ELE ROS',
      specialDraft: 'P BH ROS',
    })

    expect(body).toEqual({
      chains: [{ noble: 'HUG', text: 'HUG\nROS A BRU' }],
      winter: [{ lines: 'C N HUG BOB\nR N ELE ROS' }],
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

describe('winterSheet', () => {
  const at = (season: StateData['season']) => ({ season, nobles: [] }) as unknown as StateData

  it('joins the winter and special drafts in winter, line by line', () => {
    expect(winterSheet(at('winter'), { winterDraft: 'A\nB\n', specialDraft: 'C' })).toEqual({
      lines: 'A\nB\n\nC',
      origins: [
        { draft: 'winter', line: 1 },
        { draft: 'winter', line: 2 },
        { draft: 'winter', line: 3 },
        { draft: 'special', line: 1 },
      ],
    })
  })

  it('keeps only the card orders outside winter', () => {
    expect(
      winterSheet(at('spring'), { winterDraft: 'R T AAA\nR N XYZ AAA\n\nd n AAA BAS', specialDraft: 'C' }),
    ).toEqual({
      lines: 'R N XYZ AAA\nd n AAA BAS',
      origins: [
        { draft: 'winter', line: 2 },
        { draft: 'winter', line: 4 },
      ],
    })
  })
})
