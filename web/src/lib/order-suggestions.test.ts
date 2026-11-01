import { describe, expect, it } from 'vitest'

import { buildCodeIndex, suggestCodes, wordAtCursor } from '@/lib/order-suggestions'
import type { StateData, Territory } from '@/types'

function territory(id: string, name: string, adjacencies: string[] = []): Territory {
  return { id, name, terrain: 'plain', village: false, points: [], adjacencies, impassable: [] }
}

const territories = [
  territory('RVE', 'Rochevent', ['ECP']),
  territory('ECP', 'Escarpelle', ['RVE']),
  territory('RLG', 'Rochelong'),
  territory('REV', 'Révelin'),
]

const state = {
  territories: [
    { id: 'RLG', owner: 'P1', army: null },
    { id: 'REV', owner: 'P2', army: { owner: 'P2', size: 1, chain: null } },
  ],
  nobles: [
    { code: 'RAI', name: 'Rainaud', owner: 'P1' },
    { code: 'ROB', name: 'Robert', owner: 'P2' },
  ],
} as unknown as StateData

const index = buildCodeIndex(territories, state, 'P1')
const codes = (word: string, lineBefore = '') =>
  suggestCodes(index, word, lineBefore).map((suggestion) => suggestion.code)

describe('wordAtCursor', () => {
  it('finds the word the cursor ends, from two letters on', () => {
    expect(wordAtCursor('RVE A EC', 8)).toEqual({ start: 6, end: 8, text: 'EC' })
    expect(wordAtCursor('RVE A', 5)).toBeNull()
    expect(wordAtCursor('RVE A ECP', 7)).toBeNull()
  })

  it('ignores comments', () => {
    expect(wordAtCursor('RVE A ECP # vers Ro', 19)).toBeNull()
    expect(wordAtCursor('# note\nRo', 9)).toEqual({ start: 7, end: 9, text: 'Ro' })
  })
})

describe('suggestCodes', () => {
  it('ranks own codes before the others, then by code', () => {
    expect(codes('R')).toEqual(['RAI', 'RLG', 'REV', 'ROB', 'RVE'])
  })

  it('puts the neighbours of the line territory first', () => {
    expect(codes('E', 'RVE A ')).toEqual(['ECP'])
    expect(codes('R', 'ECP A ')[0]).toBe('RVE')
  })

  it('matches commune and noble names, accents aside', () => {
    expect(codes('reve')).toEqual(['REV'])
    expect(codes('rain')).toEqual(['RAI'])
  })

  it('offers nothing for a whole code', () => {
    expect(codes('ecp')).toEqual([])
  })
})
