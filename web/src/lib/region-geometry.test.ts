import { describe, expect, it } from 'vitest'

import type { Point, Region, Territory } from '@/types'
import {
  chainSegments,
  computeRegionOutlines,
  fitLabelFontSize,
  insetPolygon,
  loopIsHole,
  polylineLength,
  polygonContains,
} from '@/lib/region-geometry'

function squarePoints(x: number, y: number, size = 50): Array<[number, number]> {
  return [
    [x, y],
    [x + size, y],
    [x + size, y + size],
    [x, y + size],
  ]
}

describe('insetPolygon', () => {
  it('shrinks a square by the given distance', () => {
    const inset = insetPolygon(squarePoints(0, 0, 10), 2)
    expect(inset).toHaveLength(4)
    for (const [x, y] of inset) {
      expect(x).toBeGreaterThanOrEqual(2 - 1e-9)
      expect(x).toBeLessThanOrEqual(8 + 1e-9)
      expect(y).toBeGreaterThanOrEqual(2 - 1e-9)
      expect(y).toBeLessThanOrEqual(8 + 1e-9)
    }
  })

  it('is independent of the input orientation', () => {
    const ring = squarePoints(0, 0, 10)
    const forward = insetPolygon(ring, 2)
    const reversed = insetPolygon([...ring].reverse(), 2)
    expect(reversed).toEqual(forward)
  })

  it('expands with a negative distance', () => {
    const expanded = insetPolygon(squarePoints(2, 2, 6), -2)
    const xs = expanded.map(([x]) => x)
    const ys = expanded.map(([, y]) => y)
    expect(Math.min(...xs)).toBeCloseTo(0, 9)
    expect(Math.max(...xs)).toBeCloseTo(10, 9)
    expect(Math.min(...ys)).toBeCloseTo(0, 9)
    expect(Math.max(...ys)).toBeCloseTo(10, 9)
  })
})

describe('chainSegments', () => {
  it('chains a square outline into one loop', () => {
    const [a, b, c, d] = squarePoints(0, 0, 10)
    const loops = chainSegments([
      [a, b],
      [b, c],
      [c, d],
      [d, a],
    ])
    expect(loops).toHaveLength(1)
    expect(loops[0]).toHaveLength(4)
  })

  it('chains two disjoint outlines into two loops', () => {
    const [a, b, c, d] = squarePoints(0, 0, 10)
    const [e, f, g, h] = squarePoints(20, 20, 10)
    const loops = chainSegments([
      [a, b],
      [b, c],
      [c, d],
      [d, a],
      [e, f],
      [f, g],
      [g, h],
      [h, e],
    ])
    expect(loops).toHaveLength(2)
  })
})

describe('computeRegionOutlines', () => {
  function territory(id: string, points: Array<[number, number]>, village = false): Territory {
    return {
      id,
      name: id,
      terrain: 'plain',
      village,
      points,
      adjacencies: [],
      impassable: [],
    }
  }

  it('treats inter-region borders as boundary for both regions', () => {
    const territories = [
      territory('AAA', squarePoints(0, 0, 50)),
      territory('BBB', squarePoints(50, 0, 50)),
    ]
    const regions: Region[] = [
      { id: 'R1', seed: 'AAA', territories: ['AAA'] },
      { id: 'R2', seed: 'BBB', territories: ['BBB'] },
    ]
    const { outlines } = computeRegionOutlines(territories, regions)
    expect(outlines.get('R1')?.loops).toHaveLength(1)
    expect(outlines.get('R1')?.outerSegments).toHaveLength(3)
    expect(outlines.get('R2')?.loops).toHaveLength(1)
    expect(outlines.get('R2')?.outerSegments).toHaveLength(3)
  })

  it('excludes internal borders shared by territories of the same region', () => {
    const territories = [
      territory('AAA', squarePoints(0, 0, 50)),
      territory('BBB', squarePoints(50, 0, 50)),
    ]
    const regions: Region[] = [
      { id: 'R1', seed: 'AAA', territories: ['AAA', 'BBB'] },
    ]
    const { outlines } = computeRegionOutlines(territories, regions)
    expect(outlines.get('R1')?.loops).toHaveLength(1)
    expect(outlines.get('R1')?.outerSegments).toHaveLength(6)
    expect(outlines.get('R1')?.loops[0]).toHaveLength(6)
  })

  function grid3x3(): Territory[] {
    const ids = ['AAA', 'BBB', 'CCC', 'DDD', 'EEE', 'FFF', 'GGG', 'HHH', 'III']
    return ids.map((id, index) => {
      const column = index % 3
      const row = Math.floor(index / 3)
      return territory(id, squarePoints(column * 50, row * 50), false)
    })
  }

  it('detects enclave loops as holes', () => {
    const territories = grid3x3()
    const regions: Region[] = [
      {
        id: 'RING',
        seed: 'AAA',
        territories: ['AAA', 'BBB', 'CCC', 'DDD', 'FFF', 'GGG', 'HHH', 'III'],
      },
      { id: 'CORE', seed: 'EEE', territories: ['EEE'] },
    ]
    const { outlines } = computeRegionOutlines(territories, regions)
    const regionOf = (id: string) => regions.find((region) => region.territories.includes(id))?.id
    const territoryAt = (point: [number, number]) => {
      for (const candidate of territories) {
        if (polygonContains(candidate.points, point)) {
          return candidate.id
        }
      }
      return undefined
    }
    const coreLoop = outlines.get('CORE')?.loops[0] ?? []
    expect(coreLoop).toHaveLength(4)
    expect(outlines.get('CORE')?.outerSegments).toHaveLength(0)
    expect(outlines.get('CORE')?.outerLoops).toHaveLength(0)
    expect(loopIsHole(coreLoop, territoryAt, regionOf, 'RING')).toBe(true)
    expect(loopIsHole(coreLoop, territoryAt, regionOf, 'CORE')).toBe(false)
  })

  it('chains the outer border into one contiguous stretch per region', () => {
    const territories = grid3x3()
    const regions: Region[] = [
      {
        id: 'TOP',
        seed: 'AAA',
        territories: ['AAA', 'BBB', 'CCC'],
      },
      {
        id: 'REST',
        seed: 'DDD',
        territories: ['DDD', 'EEE', 'FFF', 'GGG', 'HHH', 'III'],
      },
    ]
    const { outlines } = computeRegionOutlines(territories, regions)
    expect(outlines.get('TOP')?.outerSegments).toHaveLength(5)
    expect(outlines.get('TOP')?.outerLoops).toHaveLength(1)
    expect(outlines.get('TOP')?.outerLoops[0]).toHaveLength(6)
    expect(outlines.get('REST')?.outerSegments).toHaveLength(7)
    expect(outlines.get('REST')?.outerLoops).toHaveLength(1)
    const restLoop = outlines.get('REST')?.outerLoops[0] ?? []
    expect(restLoop).toHaveLength(8)
  })
})

describe('polyline helpers', () => {
  it('measures polyline length', () => {
    expect(polylineLength([[0, 0], [3, 0], [3, 4]] as Point[])).toBeCloseTo(7, 9)
  })
})

describe('fitLabelFontSize', () => {
  it('keeps the base size for short labels', () => {
    expect(fitLabelFontSize('BMH', 1000)).toBe(11)
  })

  it('shrinks long labels to fit the segment', () => {
    const size = fitLabelFontSize('Bishopric of Beaumarchais (BMH)', 120)
    expect(size).toBeGreaterThanOrEqual(6.5)
    expect(size).toBeLessThan(11)
  })
})
