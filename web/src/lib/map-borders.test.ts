import { describe, expect, it } from 'vitest'

import {
  computeMapBorders,
  impassableBorderIcons,
  impassableBorderKind,
  lowerTerritory,
} from '@/lib/map-borders'
import type { MapData, Terrain, Territory } from '@/types'

function square(
  id: string,
  x: number,
  adjacencies: string[],
  impassable: string[],
  terrain: Terrain = 'plain',
): Territory {
  return {
    id,
    name: id,
    terrain,
    village: false,
    points: [
      [x, 10],
      [x + 100, 10],
      [x + 100, 110],
      [x, 110],
    ],
    adjacencies,
    impassable,
  }
}

const map: MapData = {
  territories: [
    square('ROS', 10, ['BRU'], []),
    square('BRU', 110, ['ROS'], ['VAL']),
    square('VAL', 210, [], ['BRU']),
  ],
}

describe('computeMapBorders', () => {
  it('mirrors the top-left margin to size the map', () => {
    const { mapWidth, mapHeight } = computeMapBorders(map)
    expect(mapWidth).toBe(320)
    expect(mapHeight).toBe(120)
  })

  it('splits single-owner edges from shared ones', () => {
    const { outerBorders, sharedBorders } = computeMapBorders(map)
    expect(outerBorders).toHaveLength(8)
    expect(sharedBorders.map((border) => border.passable)).toEqual([true, false])
  })

  it('falls back to a unit map without territories', () => {
    const { mapWidth, mapHeight } = computeMapBorders({ territories: [] })
    expect([mapWidth, mapHeight]).toEqual([1, 1])
  })
})

describe('impassableBorderIcons', () => {
  it('chains icons along impassable borders only, depth-sorted', () => {
    const { sharedBorders } = computeMapBorders(map)
    const icons = impassableBorderIcons(sharedBorders, map.territories, 1)
    const impassableKey = sharedBorders.find((border) => !border.passable)?.key
    expect(icons.length).toBeGreaterThan(0)
    expect(icons.every((icon) => icon.key.startsWith(`${impassableKey}-`))).toBe(true)
    const ys = icons.map((icon) => icon.placement.y)
    expect(ys).toEqual([...ys].sort((first, second) => first - second))
  })
})

describe('impassableBorderKind', () => {
  it('prefers the marsh, then the peaks, then the escarpment', () => {
    expect(impassableBorderKind('swamp', 'mountain')).toBe('marsh')
    expect(impassableBorderKind('hill', 'mountain')).toBe('peaks')
    expect(impassableBorderKind('hill', 'plain')).toBe('cliff')
    expect(impassableBorderKind('forest', 'forest')).toBe('cliff')
  })

  it('turns an escarpment towards the lower relief, then the larger code', () => {
    const hill = square('ABC', 0, [], [], 'hill')
    const plain = square('ABD', 0, [], [], 'plain')
    expect(lowerTerritory(hill, plain).id).toBe('ABD')
    expect(lowerTerritory(plain, hill).id).toBe('ABD')
    expect(lowerTerritory(square('ZZZ', 0, [], []), plain).id).toBe('ZZZ')
  })

  it('faces the cliff icons into the lower territory', () => {
    const { sharedBorders } = computeMapBorders(map)
    const icons = impassableBorderIcons(sharedBorders, map.territories, 1)
    expect(icons.every((icon) => icon.kind === 'cliff')).toBe(true)
    // BRU and VAL are both plains: the cliff faces VAL, east of the border.
    for (const { placement } of icons) {
      expect(-Math.sin(placement.rotation)).toBeGreaterThan(0.99)
    }
  })
})
