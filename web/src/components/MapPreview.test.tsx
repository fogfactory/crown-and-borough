import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { MapData } from '@/types'

import { MapPreview } from './MapPreview'

function territory(id: string, x: number) {
  return {
    id,
    name: id,
    terrain: 'plain',
    village: false,
    points: [
      [x, 0],
      [x + 10, 0],
      [x + 10, 10],
      [x, 10],
    ],
    adjacencies: [],
    impassable: [],
  }
}

const map = {
  territories: [territory('AAA', 0), territory('BBB', 10), territory('CCC', 20)],
  regions: [
    { id: 'R1', name: 'North', territories: ['AAA', 'BBB'] },
    { id: 'R2', name: 'South', territories: ['CCC'] },
  ],
  starts: [],
} as unknown as MapData

describe('MapPreview', () => {
  it('draws every territory tinted by region and one border path per region', () => {
    const { container } = render(<MapPreview map={map} label="Candidate" />)
    expect(container.querySelectorAll('[data-preview-territory]')).toHaveLength(3)
    const tinted = container.querySelector('[data-preview-territory="AAA"]')
    expect(tinted?.getAttribute('fill')).not.toBe('none')
    const borders = container.querySelectorAll('path[data-preview-region]')
    expect(borders).toHaveLength(2)
    for (const border of borders) {
      expect(border.getAttribute('d')).toMatch(/^M /)
      expect(border.getAttribute('fill')).toBe('none')
    }
    expect(container.querySelector('svg')?.getAttribute('aria-label')).toBe('Candidate')
  })

  it('draws nothing for an empty map', () => {
    const { container } = render(
      <MapPreview map={{ territories: [], regions: [], starts: [] } as unknown as MapData} label="x" />,
    )
    expect(container.querySelector('svg')).toBeNull()
  })
})
