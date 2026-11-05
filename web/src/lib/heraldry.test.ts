import { describe, expect, it } from 'vitest'

import { resolveArmsByOwner } from '@/lib/heraldry'
import type { StateData } from '@/types'

const state = {
  players: [
    {
      id: 'P1',
      name: 'One',
      color: '#b3322b',
      arms: { field: 'gules', metal: 'or', charge: 'bend' },
    },
    { id: 'P2', name: 'Two', color: '#2b5c9e' },
  ],
} as unknown as StateData

describe('resolveArmsByOwner', () => {
  it('lays the charge in its metal over the player color', () => {
    const arms = resolveArmsByOwner(state, new Map([['P1', '#b3322b']]))
    expect(arms.get('P1')).toEqual({
      field: '#b3322b',
      charge: { kind: 'bend', metal: '#d8ab35' },
    })
  })

  it('draws a plain field for a player without arms', () => {
    const arms = resolveArmsByOwner(state, new Map([['P2', '#2b5c9e']]))
    expect(arms.get('P2')).toEqual({ field: '#2b5c9e', charge: null })
  })

  it('always provides the ashen neutral arms, for rebels and vacant fiefs', () => {
    const arms = resolveArmsByOwner(state, new Map())
    expect(arms.get('NEUTRAL')).toEqual({ field: '#7d766a', charge: null })
  })
})
