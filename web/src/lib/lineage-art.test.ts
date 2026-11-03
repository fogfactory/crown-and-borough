import { describe, expect, it } from 'vitest'

import { endAngle, pathPoints, roundedPath } from './lineage-art'

describe('lineage art', () => {
  it('reads the points of a stem path', () => {
    expect(pathPoints('M10,20 V50 H80')).toEqual([
      { x: 10, y: 20 },
      { x: 10, y: 50 },
      { x: 80, y: 50 },
    ])
    expect(pathPoints('M0,0 L5,5')).toHaveLength(2)
  })

  it('rounds each corner within half of its neighbouring segments', () => {
    const points = pathPoints('M0,0 V10 H100')
    // The 10px drop only leaves a 5px radius.
    expect(roundedPath(points, 18)).toBe('M0,0 L0,5 Q0,10 5,10 L100,10')
    expect(roundedPath(pathPoints('M0,0 H40'))).toBe('M0,0 L40,0')
  })

  it('gives the direction of the last segment', () => {
    expect(endAngle(pathPoints('M0,0 V10 H20'))).toBe(0)
    expect(endAngle(pathPoints('M20,0 V10 H0'))).toBeCloseTo(Math.PI)
  })
})
