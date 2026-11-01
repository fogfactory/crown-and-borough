import { describe, expect, it } from 'vitest'

import {
  buildCodeIndex,
  chainSuggester,
  sheetSuggester,
  SPECIAL_SLOTS,
  suggestCodes,
  WINTER_SLOTS,
  wordAtCursor,
} from '@/lib/order-suggestions'
import type { StateData, Territory, WinterAids } from '@/types'

function territory(id: string, name: string, adjacencies: string[] = [], village = false): Territory {
  return { id, name, terrain: 'plain', village, points: [], adjacencies, impassable: [] }
}

const territories = [
  territory('RVE', 'Rochevent', ['ECP', 'RLG']),
  territory('ECP', 'Escarpelle', ['RVE']),
  territory('RLG', 'Rochelong', ['RVE'], true),
  territory('REV', 'Révelin'),
]

const army = (owner: string) => ({ owner, size: 1, chain: null })
const state = {
  territories: [
    { id: 'RVE', owner: 'P1', army: army('P1'), infrastructures: [{ type: 'castle', level: 1 }] },
    { id: 'ECP', owner: null, army: null, infrastructures: [] },
    { id: 'RLG', owner: 'P1', army: null, infrastructures: [] },
    { id: 'REV', owner: 'P2', army: army('P2'), infrastructures: [{ type: 'village', level: 1 }] },
  ],
  nobles: [
    { code: 'RAI', name: 'Rainaud', owner: 'P1' },
    { code: 'ROB', name: 'Robert', owner: 'P2' },
  ],
  revoltTargets: ['ECP'],
} as unknown as StateData

const index = buildCodeIndex({
  territories,
  state,
  player: 'P1',
  regions: [{ id: 'R1', seed: 'RLG', territories: ['RLG', 'RVE'] }],
  nobleCards: [
    { code: 'ALI', name: 'Aliénor', kind: 'noble' },
    { code: 'BAS', name: 'Bâtard', kind: 'dignity' },
  ],
  specialCards: [
    { code: 'BT', name: 'Beau temps' },
    { code: 'RE', name: 'Révolte' },
  ],
  calamities: [{ code: 'PE', name: 'Peste' }],
  aids: {
    excommunicable: [{ code: 'ROB' }],
    rituals: [{ noble: 'RAI', region: 'R1' }],
  } as unknown as WinterAids,
})
const codes = (word: string, slot?: string[]) =>
  suggestCodes(index, word, slot).map((suggestion) => suggestion.code)
const chain = chainSuggester(index)
const chainCodes = (textBefore: string, word = '') =>
  chain(word, textBefore).map((suggestion) => suggestion.code)
const winter = sheetSuggester(index, WINTER_SLOTS)
const winterCodes = (textBefore: string, word = '') =>
  winter(word, textBefore).map((suggestion) => suggestion.code)
const special = sheetSuggester(index, SPECIAL_SLOTS)
const specialCodes = (textBefore: string, word = '') =>
  special(word, textBefore).map((suggestion) => suggestion.code)

describe('wordAtCursor', () => {
  it('finds the word the cursor ends, even empty, but not inside a word', () => {
    expect(wordAtCursor('RVE A EC', 8)).toEqual({ start: 6, end: 8, text: 'EC' })
    expect(wordAtCursor('RVE A ', 6)).toEqual({ start: 6, end: 6, text: '' })
    expect(wordAtCursor('RVE A ECP', 7)).toBeNull()
  })

  it('ignores comments', () => {
    expect(wordAtCursor('RVE A ECP # vers Ro', 19)).toBeNull()
    expect(wordAtCursor('# note\nRo', 9)).toEqual({ start: 7, end: 9, text: 'Ro' })
  })
})

describe('suggestCodes', () => {
  it('ranks the slot first, then own codes, then by code', () => {
    expect(codes('R')).toEqual(['RAI', 'RE', 'RLG', 'RVE', 'REV', 'ROB'])
    expect(codes('R', ['ROB'])[0]).toBe('ROB')
  })

  it('matches names, accents aside', () => {
    expect(codes('reve')).toEqual(['REV'])
    expect(codes('rain')).toEqual(['RAI'])
  })

  it('lists the slot alone for an empty word, and nothing without one', () => {
    expect(codes('', ['ECP', 'RVE'])).toEqual(['ECP', 'RVE'])
    expect(codes('')).toEqual([])
  })

  it('offers nothing for a whole code', () => {
    expect(codes('ecp')).toEqual([])
    expect(codes('bt', ['BT'])).toEqual([])
  })
})

describe('chain suggestions', () => {
  it('opens a chain on the armies, the player first', () => {
    expect(chainCodes('')).toEqual(['RVE', 'REV'])
    expect(chainCodes('RAI\n')).toEqual(['RVE', 'REV'])
    expect(chainCodes('H ')).toEqual(['RVE', 'REV'])
  })

  it('offers the neighbours after a move and the position or a neighbour in a dispersion', () => {
    expect(chainCodes('RVE A ')).toEqual(['ECP', 'RLG'])
    expect(chainCodes('RVE D ')).toEqual(['RVE', 'ECP', 'RLG'])
    expect(chainCodes('RVE S RLG - ')).toEqual(['RVE'])
  })

  it('follows the army along the chain', () => {
    expect(chainCodes('RVE A ECP\n')).toEqual(['ECP', 'RVE'])
    expect(chainCodes('(RVE S ECP)\n')).toEqual(['RVE', 'ECP', 'RLG'])
  })

  it('expects no code at the verb', () => {
    expect(chainCodes('RVE ')).toEqual([])
  })
})

describe('winter suggestions', () => {
  it('reads the places of each winter order', () => {
    expect(winterCodes('R N ')).toEqual(['ALI'])
    expect(winterCodes('R N ALI ')).toEqual(['RVE', 'RLG'])
    expect(winterCodes('C M ')).toEqual(['RVE', 'RLG'])
    expect(winterCodes('O N ')).toEqual(['ROB'])
    expect(winterCodes('D N ROB ')).toEqual(['BAS'])
    expect(winterCodes('G RVE ')).toEqual(['REV'])
    expect(winterCodes('H N RAI ')).toEqual(['REV'])
  })

  it('reads the title orders in the winter aids', () => {
    expect(winterCodes('X E ')).toEqual(['ROB'])
    expect(winterCodes('S R ')).toEqual(['RAI'])
    expect(winterCodes('S R RAI ')).toEqual(['PE'])
    expect(winterCodes('S R RAI ', 'PE')).toEqual([])
  })

  it('repeats the territories of a new fief and offers neighbours to annex', () => {
    expect(winterCodes('T F RAI RVE RLG ')).toEqual(['RVE', 'RLG'])
    expect(winterCodes('T X RAI RVE ')).toEqual(['ECP', 'RLG'])
  })
})

describe('special card suggestions', () => {
  it('offers the cards of the hand, then the target of each card', () => {
    expect(specialCodes('P ')).toEqual(['BT', 'RE'])
    expect(specialCodes('P BT ')).toEqual(['RLG'])
    expect(specialCodes('P RE ')).toEqual(['ECP'])
    expect(specialCodes('P AS RAI ')).toEqual(['ROB'])
  })
})
