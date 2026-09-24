import { describe, expect, it } from 'vitest'

import { computeMapBorders, impassableBorderIcons } from '@/lib/map-borders'
import type { MapData, Territory } from '@/types'

function square(
  id: string,
  x: number,
  adjacencies: string[],
  impassable: string[],
): Territory {
  return {
    id,
    name: id,
    terrain: 'plain',
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
    const { outerBorders, sharedBorders, passableBoundaryKeys } = computeMapBorders(map)
    expect(outerBorders).toHaveLength(8)
    expect(sharedBorders.map((border) => border.passable)).toEqual([true, false])
    expect(passableBoundaryKeys.size).toBe(1)
  })

  it('falls back to a unit map without territories', () => {
    const { mapWidth, mapHeight } = computeMapBorders({ territories: [] })
    expect([mapWidth, mapHeight]).toEqual([1, 1])
  })
})

describe('impassableBorderIcons', () => {
  it('chains icons along impassable borders only, depth-sorted', () => {
    const { sharedBorders } = computeMapBorders(map)
    const icons = impassableBorderIcons(sharedBorders, 1)
    const impassableKey = sharedBorders.find((border) => !border.passable)?.key
    expect(icons.length).toBeGreaterThan(0)
    expect(icons.every((icon) => icon.key.startsWith(`${impassableKey}-`))).toBe(true)
    const ys = icons.map((icon) => icon.placement.y)
    expect(ys).toEqual([...ys].sort((first, second) => first - second))
  })
})
