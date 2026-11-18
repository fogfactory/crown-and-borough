import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { CandidateMapData } from '@/types'

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
} as unknown as CandidateMapData

describe('MapPreview', () => {
  it('draws every territory tinted by region, with its terrain symbol and a border per region', () => {
    const { container } = render(<MapPreview map={map} label="Candidate" />)
    expect(container.querySelectorAll('[data-preview-territory]')).toHaveLength(3)
    const first = container.querySelector('[data-preview-territory="AAA"]')
    const third = container.querySelector('[data-preview-territory="CCC"]')
    expect(first?.getAttribute('fill')).not.toBe('none')
    expect(first?.getAttribute('fill')).not.toBe(third?.getAttribute('fill'))
    expect(container.querySelectorAll('[data-preview-terrain="plain"]')).toHaveLength(3)
    expect(container.querySelectorAll('path[data-preview-region]')).toHaveLength(2)
    expect(container.querySelector('svg')?.getAttribute('aria-label')).toBe('Candidate')
  })

  it('draws a heavy wall on each impassable frontier only', () => {
    const walled = {
      territories: [
        { ...territory('AAA', 0), adjacencies: ['BBB'], impassable: ['CCC'] },
        { ...territory('BBB', 10), adjacencies: ['AAA'], impassable: [] },
        { ...territory('CCC', 20), adjacencies: [], impassable: ['AAA'] },
      ],
      regions: [],
      starts: [],
    } as unknown as CandidateMapData
    // AAA and CCC do not share an edge here, so nothing is drawn; BBB-CCC share
    // one but are declared neither passable nor impassable.
    expect(render(<MapPreview map={walled} label="x" />).container.querySelectorAll('[data-preview-wall]')).toHaveLength(0)
    const adjacent = {
      territories: [
        { ...territory('AAA', 0), impassable: ['BBB'] },
        { ...territory('BBB', 10), impassable: ['AAA'] },
        { ...territory('CCC', 20), adjacencies: ['BBB'] },
      ],
      regions: [],
      starts: [],
    } as unknown as CandidateMapData
    const { container } = render(<MapPreview map={adjacent} label="x" />)
    expect(container.querySelectorAll('[data-preview-wall]')).toHaveLength(1)
  })

  it('marks the start castles, the villages and the region seats', () => {
    const marked = {
      territories: [
        { ...territory('AAA', 0), village: false },
        { ...territory('BBB', 10), village: true },
        { ...territory('CCC', 20), village: true },
      ],
      regions: [
        { id: 'R1', name: 'North', seed: 'BBB', territories: ['AAA', 'BBB'] },
        { id: 'R2', name: 'South', seed: 'CCC', territories: ['CCC'] },
      ],
      starts: ['AAA', 'CCC'],
    } as unknown as CandidateMapData
    const { container } = render(<MapPreview map={marked} label="Candidate" />)
    expect(container.querySelectorAll('[data-preview-castle]')).toHaveLength(2)
    // A village that holds a castle is drawn as the castle only.
    expect(container.querySelectorAll('[data-preview-village]')).toHaveLength(1)
    expect(container.querySelectorAll('[data-preview-seat]')).toHaveLength(2)
  })

  it('draws nothing for an empty map', () => {
    const { container } = render(
      <MapPreview map={{ territories: [], regions: [], starts: [] } as unknown as CandidateMapData} label="x" />,
    )
    expect(container.querySelector('svg')).toBeNull()
  })
})
