import { describe, expect, it } from 'vitest'

import { footprintFits, placeMarker } from './map-svg-geometry'

const square: Array<[number, number]> = [
  [0, 0],
  [10, 0],
  [10, 10],
  [0, 10],
]
const wide: Array<[number, number]> = [
  [-2, 0],
  [2, 0],
]

describe('placeMarker', () => {
  it('keeps a marker whose footprint fits', () => {
    expect(placeMarker([7, 5], [], square, [5, 5], wide)).toEqual([7, 5])
  })

  it('pulls a marker that spills over a neighbour back inside', () => {
    const point = placeMarker([20, 5], [], square, [5, 5], wide)
    expect(footprintFits(point, square, wide)).toBe(true)
    expect(point[0]).toBeGreaterThan(7)
    expect(point[1]).toBe(5)
  })

  it('pulls a marker back until the whole footprint is inside', () => {
    const [x] = placeMarker([9, 5], [], square, [5, 5], wide)
    expect(x).toBeLessThanOrEqual(8)
    expect(x).toBeGreaterThan(7)
  })

  it('takes the first fallback that fits when the center does not', () => {
    const point = placeMarker(
      [20, 5],
      [
        [-5, 5],
        [5, 8],
      ],
      square,
      [-5, 5],
      wide,
    )
    expect(point).toEqual([5, 8])
  })

  it('takes the first fallback when none fits', () => {
    const point = placeMarker(
      [20, 5],
      [
        [-5, 5],
        [-6, 5],
      ],
      square,
      [-5, 5],
      wide,
    )
    expect(point).toEqual([-5, 5])
  })
})
