import { describe, expect, it } from 'vitest'

import { computeFiefOutlines, fiefOutlinePath } from '@/lib/fief-geometry'
import type { Fief, Territory } from '@/types'

function squarePoints(x: number, y: number, size = 50): Array<[number, number]> {
  return [
    [x, y],
    [x + size, y],
    [x + size, y + size],
    [x, y + size],
  ]
}

function territory(id: string, points: Array<[number, number]>): Territory {
  return {
    id,
    name: id,
    terrain: 'plain',
    village: false,
    points,
    adjacencies: [],
    impassable: [],
  }
}

describe('computeFiefOutlines', () => {
  const territories = [
    territory('AAA', squarePoints(0, 0, 50)),
    territory('BBB', squarePoints(50, 0, 50)),
    territory('CCC', squarePoints(100, 0, 50)),
  ]

  it('treats a fief group as one pseudo-region seeded on its capital', () => {
    const fiefs: Fief[] = [
      { capital: 'AAA', title: 'barony', territories: ['AAA', 'BBB'], owner: 'P1' },
    ]
    const outlines = computeFiefOutlines(territories, fiefs)
    expect(outlines.outlines.get('AAA')?.loops).toHaveLength(1)
    // The AAA/BBB shared border is internal to the fief and excluded.
    expect(outlines.outlines.get('AAA')?.outerSegments).toHaveLength(6)
  })

  it('never merges two distinct fiefs even when their territories touch', () => {
    const fiefs: Fief[] = [
      { capital: 'AAA', title: 'barony', territories: ['AAA'], owner: 'P1' },
      { capital: 'CCC', title: 'barony', territories: ['CCC'], owner: 'P2' },
    ]
    const outlines = computeFiefOutlines(territories, fiefs)
    expect(outlines.outlines.get('AAA')?.loops).toHaveLength(1)
    expect(outlines.outlines.get('CCC')?.loops).toHaveLength(1)
  })

  it('renders a fief outline as a combined SVG path with one subpath per loop', () => {
    const fiefs: Fief[] = [
      { capital: 'AAA', title: 'barony', territories: ['AAA', 'BBB'], owner: 'P1' },
    ]
    const outlines = computeFiefOutlines(territories, fiefs)
    const path = fiefOutlinePath(outlines, 'AAA')
    expect(path.startsWith('M ')).toBe(true)
    expect(path).toContain('Z')
  })

  it('returns an empty path for a fief absent from the outlines', () => {
    const outlines = computeFiefOutlines(territories, [])
    expect(fiefOutlinePath(outlines, 'ZZZ')).toBe('')
  })
})
