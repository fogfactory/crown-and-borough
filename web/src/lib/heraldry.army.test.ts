import { describe, expect, it } from 'vitest'

import {
  armyFigures,
  desaturate,
  greyArms,
  hashSeed,
  shadowCount,
  FIGURE_HEAD_TOP,
} from '@/lib/heraldry'

describe('desaturate', () => {
  it('keeps the luminance and drops the color', () => {
    expect(desaturate('#ffffff')).toBe('#ffffff')
    expect(desaturate('#000000')).toBe('#000000')
    const grey = desaturate('#b3322b')
    expect(grey.slice(1, 3)).toBe(grey.slice(3, 5))
    expect(grey.slice(3, 5)).toBe(grey.slice(5, 7))
  })

  it('greys the field and the metal of arms, not their charge', () => {
    const arms = {
      field: '#2b5c9e',
      charge: { kind: 'fess' as const, metal: '#d8ab35' },
    }
    const grey = greyArms(arms)
    expect(grey.charge?.kind).toBe('fess')
    expect(grey.field).toBe(desaturate('#2b5c9e'))
    expect(grey.charge?.metal).toBe(desaturate('#d8ab35'))
    expect(greyArms({ field: '#7d766a', charge: null }).charge).toBeNull()
  })
})

describe('armyFigures', () => {
  it('draws one figurine per troop, six at most', () => {
    expect(armyFigures(1, 1).soldiers).toHaveLength(1)
    expect(armyFigures(4, 1).soldiers).toHaveLength(4)
    expect(armyFigures(6, 1).soldiers).toHaveLength(6)
    expect(armyFigures(40, 1).soldiers).toHaveLength(6)
  })

  it('adds grey shadows only beyond six troops', () => {
    expect([6, 7, 9, 10, 19, 20, 99].map(shadowCount)).toEqual([0, 2, 2, 3, 3, 4, 4])
    expect(armyFigures(6, 1).shadows).toHaveLength(0)
    expect(armyFigures(12, 1).shadows).toHaveLength(3)
  })

  it('is stable for a seed and differs between seeds', () => {
    expect(armyFigures(6, hashSeed('ROS'))).toEqual(armyFigures(6, hashSeed('ROS')))
    expect(armyFigures(6, hashSeed('ROS'))).not.toEqual(armyFigures(6, hashSeed('BRU')))
  })

  it('names the leftmost soldier as the bearer', () => {
    for (const seed of [1, 2, 3, 4, 5]) {
      const { soldiers, bearer } = armyFigures(6, seed)
      expect(soldiers[bearer].x).toBe(Math.min(...soldiers.map((soldier) => soldier.x)))
    }
  })

  it('keeps shadows behind the soldiers, below the banner clearance', () => {
    const { shadows } = armyFigures(25, 7)
    for (const shadow of shadows) {
      expect(shadow.y).toBeLessThan(-5)
      expect(shadow.y + FIGURE_HEAD_TOP).toBeLessThan(0)
    }
  })
})
