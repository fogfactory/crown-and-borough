import { fireEvent, render, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { MapViewer } from '@/components/MapViewer'
import type { MapData, StateData } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'T1',
      code: 'ALP',
      name: 'Alpilles',
      terrain: 'plain',
      village: false,
      points: [
        [0, 0],
        [50, 0],
        [50, 50],
        [0, 50],
      ],
      adjacencies: ['T2'],
      impassable: [],
    },
    {
      id: 'T2',
      code: 'BRI',
      name: 'Brisecote',
      terrain: 'forest',
      village: false,
      points: [
        [50, 0],
        [100, 0],
        [100, 50],
        [50, 50],
      ],
      adjacencies: ['T1'],
      impassable: [],
    },
  ],
}

const state: StateData = {
  turn: 1,
  season: 'spring',
  asOf: { T1: 1, T2: 1 },
  players: [],
  territories: [
    { id: 'T1', owner: 'P1', resources: 0, army: null, infrastructures: [] },
    { id: 'T2', owner: 'P2', resources: 0, army: null, infrastructures: [] },
  ],
  nobles: [],
}

function renderMap(
  testMap: MapData = map,
  testState: StateData = state,
  onSelect = vi.fn(),
) {
  const result = render(<MapViewer map={testMap} state={testState} onSelect={onSelect} />)
  const svg = result.container.querySelector(
    'svg[aria-label="Carte des territoires"]',
  ) as SVGSVGElement | null
  const firstTerritory = result.container.querySelector(
    '[data-territory-id="T1"]',
  ) as SVGPathElement | null

  if (!svg || !firstTerritory) {
    throw new Error('Map test fixture did not render the expected SVG elements')
  }

  return { ...result, firstTerritory, onSelect, svg }
}

function pointerPosition(
  clientX = 100,
  clientY = 100,
  pointerType: 'mouse' | 'touch' = 'mouse',
) {
  return { button: 0, clientX, clientY, pointerId: 1, pointerType }
}

function clickTarget(svg: SVGSVGElement, target: Element, clientX = 100) {
  fireEvent.pointerDown(target, pointerPosition(clientX))
  fireEvent.pointerUp(svg, pointerPosition(clientX))
}

describe('MapViewer selection and panning', () => {
  it('selects and toggles a territory with a left click', () => {
    const { firstTerritory, onSelect, svg } = renderMap()

    clickTarget(svg, firstTerritory)
    expect(onSelect).toHaveBeenLastCalledWith('T1')

    clickTarget(svg, firstTerritory)
    expect(onSelect).toHaveBeenLastCalledWith(null)
  })

  it('clears the selection when the SVG background is clicked', () => {
    const { firstTerritory, onSelect, svg } = renderMap()
    const background = svg.querySelector('rect')

    if (!background) {
      throw new Error('Map test fixture did not render the SVG background')
    }

    clickTarget(svg, firstTerritory)
    clickTarget(svg, background, 900)

    expect(onSelect).toHaveBeenLastCalledWith(null)
  })

  it('uses the same toggle rule for keyboard selection', () => {
    const { firstTerritory, onSelect } = renderMap()

    fireEvent.keyDown(firstTerritory, { key: 'Enter' })
    expect(onSelect).toHaveBeenLastCalledWith('T1')

    fireEvent.keyDown(firstTerritory, { key: ' ' })
    expect(onSelect).toHaveBeenLastCalledWith(null)
  })

  it('pans after the drag threshold without selecting the territory', async () => {
    const { firstTerritory, onSelect, svg } = renderMap()
    const mapGroup = svg.querySelector('g[transform]')

    if (!mapGroup) {
      throw new Error('Map test fixture did not render the transformed map group')
    }

    fireEvent.pointerDown(firstTerritory, pointerPosition())
    fireEvent.pointerMove(firstTerritory, { ...pointerPosition(300), buttons: 1 })
    fireEvent.pointerUp(svg, pointerPosition(300))

    expect(onSelect).not.toHaveBeenCalled()
    await waitFor(() => {
      expect(mapGroup).not.toHaveAttribute('transform', 'translate(0 0) scale(1)')
    })
  })

  it('keeps a sub-threshold movement as a click', () => {
    const { firstTerritory, onSelect, svg } = renderMap()

    fireEvent.pointerDown(firstTerritory, pointerPosition())
    fireEvent.pointerMove(svg, { ...pointerPosition(110), buttons: 1 })
    fireEvent.pointerUp(svg, pointerPosition(110))

    expect(onSelect).toHaveBeenLastCalledWith('T1')
  })

  it('keeps touch dragging as a pan without selecting', async () => {
    const { firstTerritory, onSelect, svg } = renderMap()
    const mapGroup = svg.querySelector('g[transform]')

    if (!mapGroup) {
      throw new Error('Map test fixture did not render the transformed map group')
    }

    fireEvent.pointerDown(firstTerritory, pointerPosition(100, 100, 'touch'))
    fireEvent.pointerMove(svg, {
      ...pointerPosition(300, 100, 'touch'),
      buttons: 1,
    })
    fireEvent.pointerUp(svg, pointerPosition(300, 100, 'touch'))

    expect(onSelect).not.toHaveBeenCalled()
    await waitFor(() => {
      expect(mapGroup).not.toHaveAttribute('transform', 'translate(0 0) scale(1)')
    })
  })
})

describe('MapViewer territorial overlays', () => {
  it('adds a light winter veil without changing the non-winter map', () => {
    const { svg: winterSvg } = renderMap(map, { ...state, season: 'winter' })
    const winterVeil = winterSvg.querySelector('g[aria-label="Voile hivernal"]')

    expect(winterVeil).toBeInTheDocument()
    expect(winterVeil?.querySelector('rect')).toHaveAttribute('fill', '#eaf3ff')
    expect(winterVeil?.querySelector('rect')).toHaveAttribute('opacity', '0.14')
    expect(winterVeil).toHaveAttribute('pointer-events', 'none')

    const { svg: springSvg } = renderMap()
    expect(
      springSvg.querySelector('g[aria-label="Voile hivernal"]'),
    ).not.toBeInTheDocument()
  })

  it.each([
    ['plain', '#b8d99a'],
    ['forest', '#3f7854'],
    ['hill', '#ad8565'],
    ['mountain', '#89929a'],
    ['swamp', '#66a6a0'],
  ] as const)(
    'keeps the %s terrain readable through the winter veil',
    (terrain, color) => {
      const testMap: MapData = {
        territories: [{ ...map.territories[0], terrain }],
      }
      const testState: StateData = {
        ...state,
        season: 'winter',
        territories: [state.territories[0]],
      }
      const { svg } = renderMap(testMap, testState)
      const terrainPath = svg.querySelector('[data-territory-id="T1"]')

      expect(terrainPath).toHaveAttribute('fill', color)
      expect(svg.querySelector('g[aria-label="Voile hivernal"]')).toBeInTheDocument()
    },
  )

  it('renders clipped interior control and selection strokes above terrain', () => {
    const { firstTerritory, svg } = renderMap(map, { ...state, season: 'winter' })
    const controlPath = svg.querySelector('g[aria-label="Contrôle territorial"] path')
    const winterVeil = svg.querySelector('g[aria-label="Voile hivernal"]')

    if (!controlPath) {
      throw new Error('Map test fixture did not render the control stroke')
    }

    expect(controlPath).toHaveAttribute('fill', 'none')
    expect(controlPath).toHaveAttribute('stroke-width', '8')
    expect(controlPath).toHaveAttribute('clip-path', 'url(#territory-clip-T1)')
    expect(controlPath).not.toHaveAttribute('opacity')
    expect(svg.querySelector('#territory-clip-T1')).toBeInTheDocument()
    expect(winterVeil?.compareDocumentPosition(controlPath as Node)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )

    clickTarget(svg, firstTerritory)
    const selectionPath = svg.querySelector('g[aria-label="Sélection"] path')

    if (!selectionPath) {
      throw new Error('Map test fixture did not render the selection stroke')
    }

    expect(selectionPath).toHaveAttribute('fill', 'none')
    expect(selectionPath).toHaveAttribute('clip-path', 'url(#territory-clip-T1)')
    expect(
      svg.querySelector('g[aria-label="Sélection"] path[stroke-dasharray]'),
    ).toHaveAttribute('stroke-dasharray', '4 3')

    const borderGroup = svg.querySelector('g[aria-label="Frontières"]')
    const passableBorder = borderGroup?.querySelector('line')
    const outerBorder = svg.querySelector('g[aria-label="Contours extérieurs"] line')
    expect(borderGroup).toBeInTheDocument()
    expect(passableBorder).toHaveAttribute('stroke-width', '2')
    expect(passableBorder).toHaveAttribute('stroke-dasharray', '4 3')
    expect(outerBorder).toHaveAttribute('stroke-width', '2')
    expect(selectionPath.compareDocumentPosition(borderGroup as Node)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
    expect(controlPath.compareDocumentPosition(borderGroup as Node)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
    expect(
      svg.querySelector('g[aria-label="Contrôle territorial"] path[stroke-dasharray]'),
    ).toHaveAttribute('stroke-dasharray', '4 3')
  })

  it('uses a thicker continuous stroke for impassable borders', () => {
    const impassableMap: MapData = {
      ...map,
      territories: map.territories.map((territory) => ({
        ...territory,
        adjacencies: [],
        impassable: [territory.id === 'T1' ? 'T2' : 'T1'],
      })),
    }
    const { svg } = renderMap(impassableMap, { ...state, season: 'winter' })
    const impassableBorder = svg.querySelector('g[aria-label="Frontières"] line')
    const winterVeil = svg.querySelector('g[aria-label="Voile hivernal"]')

    expect(impassableBorder).toHaveAttribute('stroke-width', '4')
    expect(impassableBorder).not.toHaveAttribute('stroke-dasharray')
    expect(winterVeil).toBeInTheDocument()
    expect(winterVeil?.compareDocumentPosition(impassableBorder as Node)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
    expect(
      svg.querySelector('g[aria-label="Contrôle territorial"] path[stroke-dasharray]'),
    ).not.toBeInTheDocument()
    expect(
      svg.querySelector('g[aria-label="Sélection"] path[stroke-dasharray]'),
    ).not.toBeInTheDocument()
  })
})
