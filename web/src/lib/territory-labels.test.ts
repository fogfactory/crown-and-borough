import { describe, expect, it } from 'vitest'

import {
  nameLines,
  particularTerritoryIds,
  territoryLabelLayout,
} from '@/lib/territory-labels'
import type { StateData, Territory } from '@/types'

const territory = (id: string, name: string): Territory => ({
  id,
  name,
  terrain: 'plain',
  village: false,
  points: [],
  adjacencies: [],
  impassable: [],
})

describe('nameLines', () => {
  it('breaks names over 14 characters at their middle hyphen', () => {
    expect(nameLines('Fougères-du-Marais')).toEqual(['Fougères-', 'du-Marais'])
    expect(nameLines('Rochevent')).toEqual(['Rochevent'])
    // No hyphen to break at: the name stays on one line.
    expect(nameLines('Montsereinville')).toEqual(['Montsereinville'])
  })
})

describe('territoryLabelLayout', () => {
  it('writes the code alone, the banner left of it, when the name is hidden', () => {
    const layout = territoryLabelLayout(territory('ROS', 'Rochevent'), false)
    expect(layout.nameLines).toEqual([])
    expect(layout.bannerAnchor.y).toBe(layout.codeBaseline)
  })

  it('writes the name above the code, the banner on the name baseline', () => {
    const layout = territoryLabelLayout(territory('FOU', 'Fougères-du-Marais'), true)
    expect(layout.nameLines).toHaveLength(2)
    expect(layout.nameBaselines[0]).toBeLessThan(layout.nameBaselines[1])
    expect(layout.codeBaseline).toBeGreaterThan(layout.nameBaselines[1])
    expect(layout.bannerAnchor.y).toBe(layout.nameBaselines[1])
    // The banner stands clear of the wider name, not of the code.
    expect(layout.bannerAnchor.x).toBeLessThan(
      territoryLabelLayout(territory('FOU', 'Fougères-du-Marais'), false).bannerAnchor.x,
    )
  })
})

describe('particularTerritoryIds', () => {
  it('collects capitals, fief capitals and chef-lieux', () => {
    const state = {
      players: [{ id: 'P1', name: 'One', color: '#000', capitalTerritory: 'CAP' }],
      fiefs: [{ capital: 'FIE', title: 'barony', territories: [], owner: 'P1' }],
    } as unknown as StateData
    expect([...particularTerritoryIds(state, ['CHE'])].sort()).toEqual([
      'CAP',
      'CHE',
      'FIE',
    ])
  })
})
