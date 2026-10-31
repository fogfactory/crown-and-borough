import { describe, expect, it } from 'vitest'

import { chainDraftLine, mapPreviewErrors } from '@/lib/draft-errors'
import type { OrdersPreview, OrdersPreviewError, StateData } from '@/types'

function stateOf(season: StateData['season']): StateData {
  return {
    season,
    nobles: [
      { code: 'ABC', owner: 'P1', status: 'free' },
      { code: 'DEF', owner: 'P2', status: 'free' },
    ],
  } as unknown as StateData
}

function previewOf(
  errors: OrdersPreviewError[],
  winter: OrdersPreview['winter'] = [],
): OrdersPreview {
  return { errors, chains: [], winter }
}

const drafts = { chainDrafts: {}, winterDraft: '', specialDraft: '' }

describe('chainDraftLine', () => {
  it('shifts the lines by the header added to a draft that has none', () => {
    expect(chainDraftLine('ABC', 'H AAA\nNOPE', 3)).toBe(2)
    expect(chainDraftLine('ABC', 'H AAA', 1)).toBeUndefined()
    expect(chainDraftLine('ABC', 'H AAA', undefined)).toBeUndefined()
  })

  it('skips the typed header, even after a comment line', () => {
    const draft = '# plan\nabc\nH AAA\nNOPE'
    // Sent as "ABC\n# plan\nH AAA\nNOPE".
    expect(chainDraftLine('ABC', draft, 1)).toBe(2)
    expect(chainDraftLine('ABC', draft, 2)).toBe(1)
    expect(chainDraftLine('ABC', draft, 3)).toBe(3)
    expect(chainDraftLine('ABC', draft, 4)).toBe(4)
  })
})

describe('mapPreviewErrors', () => {
  it('puts chain errors under their noble with the draft line', () => {
    const result = mapPreviewErrors(
      stateOf('spring'),
      'P1',
      { ...drafts, chainDrafts: { ABC: '# c\nABC\nH AAA\nX' } },
      previewOf([
        { source: 'chain', noble: 'ABC', line: 4, code: 'x', message: 'bad' },
        { source: 'chain', noble: 'ABC', code: 'not_received', message: 'lost' },
        { source: 'chain', noble: 'DEF', line: 2, code: 'x', message: 'other player' },
      ]),
    )
    expect(result.chains).toEqual({
      ABC: [{ line: 4, message: 'bad' }, { line: undefined, message: 'lost' }],
    })
  })

  it('splits the joined winter text between the winter and special drafts', () => {
    const result = mapPreviewErrors(
      stateOf('winter'),
      'P1',
      { ...drafts, winterDraft: 'A\nB', specialDraft: 'C\nD' },
      previewOf(
        [{ source: 'winter', line: 4, code: 'x', message: 'in special' }],
        [
          { line: 2, status: 'invalid', message: 'in winter' },
          { line: 4, status: 'invalid', message: 'in special' },
          { line: 1, status: 'applied' },
        ],
      ),
    )
    expect(result.winter).toEqual([{ line: 2, message: 'in winter' }])
    expect(result.special).toEqual([{ line: 2, message: 'in special' }])
  })

  it('starts the special draft at the first line when the winter draft is empty', () => {
    const result = mapPreviewErrors(
      stateOf('winter'),
      'P1',
      { ...drafts, winterDraft: '  \n', specialDraft: 'C' },
      previewOf([{ source: 'winter', line: 1, code: 'x', message: 'bad' }]),
    )
    expect(result.special).toEqual([{ line: 1, message: 'bad' }])
  })

  it('maps the card orders kept outside winter back to the winter draft', () => {
    const result = mapPreviewErrors(
      stateOf('spring'),
      'P1',
      { ...drafts, winterDraft: 'B AAA 3\nR N XYZ 1\n\nD N AAA BBB\nC N AAA' },
      previewOf([
        { source: 'winter', line: 2, code: 'x', message: 'second card' },
        { source: 'winter', line: 0, code: 'winter_out_of_season', message: 'season' },
      ]),
    )
    expect(result.winter).toEqual([
      { line: 4, message: 'second card' },
      { line: undefined, message: 'season' },
    ])
  })

  it('keeps special errors on their own line and lines without one as messages', () => {
    const result = mapPreviewErrors(
      stateOf('spring'),
      'P1',
      { ...drafts, specialDraft: 'A\nB' },
      previewOf([
        { source: 'special', line: 2, code: 'x', message: 'bad' },
        { source: 'special', noble: 'DEF', code: 'y', message: 'appeasement' },
      ]),
    )
    expect(result.special).toEqual([
      { line: 2, message: 'bad' },
      { line: undefined, message: 'appeasement' },
    ])
    expect(result.chains).toEqual({})
  })
})
