import { describe, expect, it } from 'vitest'

import { terrainSymbolPlacements } from '@/lib/terrain-engraving'
import type { Terrain, Territory } from '@/types'

function square(terrain: Terrain, id = 'ROS'): Territory {
  return {
    id,
    name: id,
    terrain,
    village: false,
    points: [
      [0, 0],
      [200, 0],
      [200, 200],
      [0, 200],
    ],
    adjacencies: [],
    impassable: [],
  }
}

describe('terrainSymbolPlacements', () => {
  it('is deterministic for one territory code and varies with it', () => {
    expect(terrainSymbolPlacements(square('forest'), 1)).toEqual(
      terrainSymbolPlacements(square('forest'), 1),
    )
    expect(terrainSymbolPlacements(square('forest', 'BRU'), 1)).not.toEqual(
      terrainSymbolPlacements(square('forest'), 1),
    )
  })

  it('keeps clear of the borders and of the label ellipse', () => {
    const placements = terrainSymbolPlacements(square('forest'), 1)
    expect(placements.length).toBeGreaterThan(100)
    for (const { x, y } of placements) {
      expect(Math.min(x, y, 200 - x, 200 - y)).toBeGreaterThanOrEqual(6.5)
      expect(((x - 100) / 30) ** 2 + ((y - 100) / 27) ** 2).toBeGreaterThanOrEqual(1)
    }
  })

  it('spaces the symbols by terrain, plains being the sparsest', () => {
    const count = (terrain: Terrain) => terrainSymbolPlacements(square(terrain), 1).length
    expect(count('forest')).toBeGreaterThan(count('swamp'))
    expect(count('swamp')).toBeGreaterThan(count('hill'))
    expect(count('hill')).toBeGreaterThan(count('plain'))
  })

  it('paints from the top of the territory down', () => {
    const ys = terrainSymbolPlacements(square('hill'), 1).map(({ y }) => y)
    expect(ys).toEqual([...ys].sort((first, second) => first - second))
  })
})
