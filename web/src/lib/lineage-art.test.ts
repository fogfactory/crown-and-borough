import { describe, expect, it } from 'vitest'

import { endAngle, ivyLeaves, pathPoints, roundedPath, roundedPolyline } from './lineage-art'

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

  it('grows deterministic ivy on both sides of a bond, away from the knot', () => {
    const bond = [
      { x: 0, y: 0 },
      { x: 88, y: 0 },
    ]
    const leaves = ivyLeaves(bond)
    const above = leaves.filter((l) => l.side === -1)
    expect(above.length).toBeGreaterThanOrEqual(2)
    expect(leaves.filter((l) => l.side === 1)).toHaveLength(above.length)
    for (const leaf of leaves) expect(Math.abs(leaf.x - 44)).toBeGreaterThanOrEqual(11)
    expect(ivyLeaves(bond)).toEqual(leaves)
  })

  it('keeps a few leaves on a short bond', () => {
    const leaves = ivyLeaves([
      { x: 0, y: 0 },
      { x: 18, y: 0 },
    ])
    expect(leaves.length).toBeGreaterThanOrEqual(2)
    expect(leaves.filter((l) => l.side === 1).length).toBe(
      leaves.filter((l) => l.side === -1).length,
    )
  })

  it('places leaves on the rounded path, not on the sharp corner', () => {
    const points = pathPoints('M0,0 V40 H80')
    const dense = roundedPolyline(points, 10)
    // The sampled curve cuts the corner: no sample sits on (0,40).
    expect(dense.some((p) => p.x === 0 && p.y === 40)).toBe(false)
    const leaves = ivyLeaves(points, 10)
    expect(leaves.length).toBeGreaterThan(0)
    for (const leaf of leaves) {
      // Never on the sharp corner's two legs beyond the curve: either on a leg
      // or within the quarter-circle-ish bend (x and y both inside the corner box).
      const onLeg = leaf.x === 0 || leaf.y === 40
      const inBend = leaf.x > 0 && leaf.y < 40 && leaf.x <= 10 && leaf.y >= 30
      expect(onLeg || inBend).toBe(true)
    }
    // A leaf in the bend leans between the two leg directions.
    const bend = leaves.find((l) => l.x > 0.5 && l.y < 39.5 && l.x <= 10 && l.y >= 30)
    if (bend) expect(bend.angle).toBeGreaterThan(0)
    expect(bend ? bend.angle : 0.1).toBeLessThan(Math.PI / 2)
  })
})
