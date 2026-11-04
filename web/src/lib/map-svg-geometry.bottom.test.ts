import { describe, expect, it } from 'vitest'

import { bottomAt } from '@/lib/map-svg-geometry'

describe('bottomAt', () => {
  const triangle: Array<[number, number]> = [
    [0, 0],
    [10, 0],
    [5, 10],
  ]

  it('finds the lower edge of a polygon under a vertical line', () => {
    expect(bottomAt(triangle, 5)).toBeCloseTo(10)
    expect(bottomAt(triangle, 2.5)).toBeCloseTo(5)
  })

  it('takes the lowest of several crossings', () => {
    const concave: Array<[number, number]> = [
      [0, 0],
      [10, 0],
      [10, 10],
      [5, 4],
      [0, 10],
    ]
    expect(bottomAt(concave, 1)).toBeCloseTo(8.8)
    expect(bottomAt(concave, 5)).toBeCloseTo(4)
  })

  it('is null outside the polygon', () => {
    expect(bottomAt(triangle, 20)).toBeNull()
  })
})
