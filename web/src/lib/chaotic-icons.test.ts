import { describe, expect, it } from 'vitest'

import { chaoticIconPlacements, territoryRadius } from '@/lib/chaotic-icons'
import { parseSpecialOrderPlacements } from '@/lib/game-icons'
import type { Point } from '@/types'

const SQUARE: Point[] = [
  [0, 0],
  [100, 0],
  [100, 100],
  [0, 100],
]

describe('chaoticIconPlacements', () => {
  it('produces the requested number of placements', () => {
    expect(chaoticIconPlacements(SQUARE, 5, 'plague-ROS-AAA')).toHaveLength(5)
  })

  it('is deterministic for one seed key', () => {
    const first = chaoticIconPlacements(SQUARE, 4, 'bad_weather-ROS-BBB')
    const second = chaoticIconPlacements(SQUARE, 4, 'bad_weather-ROS-BBB')
    expect(first).toEqual(second)
  })

  it('varies the layout between territories', () => {
    const first = chaoticIconPlacements(SQUARE, 4, 'plague-ROS-AAA')
    const second = chaoticIconPlacements(SQUARE, 4, 'plague-ROS-BBB')
    expect(first).not.toEqual(second)
  })

  it('keeps icons inside the territory radius', () => {
    const radius = territoryRadius(SQUARE)
    for (const placement of chaoticIconPlacements(SQUARE, 6, 'famine-ROS-CCC')) {
      const distance = Math.hypot(
        placement.x - 50,
        placement.y - 50,
      )
      expect(distance).toBeLessThanOrEqual(radius)
      expect(placement.size).toBeGreaterThan(0)
      expect(Math.abs(placement.rotation)).toBeLessThanOrEqual(0.3)
    }
  })

  it('returns nothing without points or for a non-positive count', () => {
    expect(chaoticIconPlacements([], 3, 'plague-ROS-AAA')).toEqual([])
    expect(chaoticIconPlacements(SQUARE, 0, 'plague-ROS-AAA')).toEqual([])
  })
})

describe('parseSpecialOrderPlacements', () => {
  it('parses playable card orders', () => {
    expect(
      parseSpecialOrderPlacements('P BT ROS\nP RE BRU\nP RA BOI'),
    ).toEqual([
      { kind: 'fair_weather', target: 'ROS' },
      { kind: 'revolt', target: 'BRU' },
      { kind: 'abundant_harvest', target: 'BOI' },
    ])
  })

  it('ignores discards and malformed lines', () => {
    expect(
      parseSpecialOrderPlacements('D C BT\nP BT\nP PE ROS\nP RE\nhello'),
    ).toEqual([])
  })

  it('accepts the english aliases', () => {
    expect(parseSpecialOrderPlacements('P FW ROS\nP RV BRU')).toEqual([
      { kind: 'fair_weather', target: 'ROS' },
      { kind: 'revolt', target: 'BRU' },
    ])
  })
})
