import { describe, expect, it } from 'vitest'

import { parseChainDraft } from '@/lib/chain-parse'
import { buildIntentions } from '@/lib/intent-overlay'
import type { MapData, StateData } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS', name: 'R', terrain: 'plain', village: false,
      points: [[0, 0], [50, 0], [50, 50], [0, 50]],
      adjacencies: ['BRU', 'CHA'], impassable: [],
    },
    {
      id: 'BRU', name: 'B', terrain: 'plain', village: false,
      points: [[50, 0], [100, 0], [100, 50], [50,  50]],
      adjacencies: ['ROS', 'CHA'], impassable: [],
    },
    {
      id: 'CHA', name: 'C', terrain: 'plain', village: false,
      points: [[0, 50], [50, 50], [50,                         100], [0,                         100]],
      adjacencies: ['ROS', 'BRU'], impassable: [],
    },
  ],
}

const baseState: StateData = {
  turn: 1,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [
    {
      id: 'ROS', owner: 'P1', resources: 0,
      army: { owner: 'P1', size: 3, chain: null },
      infrastructures: [],
    },
    { id: 'BRU', owner: null, resources:  0, army: null, infrastructures: [] },
    { id: 'CHA', owner: null, resources:  0, army: null, infrastructures: [] },
  ],
  nobles: [
    { id: 'N1', code: 'HUG', name: 'H', owner: 'P1', location: 'ROS', status: 'free' },
  ],
}

describe('scratch', () => {
  it('prints', () => {
    const text = 'HUG\nROS H\nROS A BRU'
    const parsed = parseChainDraft(text)
    console.log('parsed:', JSON.stringify(parsed))
    const intent = buildIntentions(map, baseState, 'P1', { HUG: text })
    console.log('intents:', JSON.stringify(intent))
    expect(parsed.length).toBeGreaterThanOrEqual(1)
  })
})
