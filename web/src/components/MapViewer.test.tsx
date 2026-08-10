import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { MapViewer } from '@/components/MapViewer'
import type { MapData, StateData, SupplyLine } from '@/types'

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
  supply: SupplyLine | null = null,
) {
  const result = render(
    <MapViewer map={testMap} state={testState} onSelect={onSelect} supply={supply} />,
  )
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
  it('does not render the legend inside the map canvas', () => {
    renderMap()

    expect(screen.queryByText('Légende')).not.toBeInTheDocument()
  })

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
  it('renders the selected army supply zone and path', () => {
    const supplyState: StateData = {
      ...state,
      players: [{ id: 'P1', name: 'One', color: '#a84632' }],
      territories: [
        {
          id: 'T1',
          owner: 'P1',
          resources: 0,
          army: { owner: 'P1', size: 2, chain: null },
          infrastructures: [],
        },
        {
          id: 'T2',
          owner: 'P1',
          resources: 0,
          army: null,
          infrastructures: [{ type: 'castle', level: 1 }],
        },
      ],
    }
    const supply: SupplyLine = {
      kind: 'army',
      territory: 'T1',
      armyOwner: 'P1',
      armySize: 2,
      rations: 1,
      demand: 1,
      source: 'T2',
      distance: 1,
      path: ['T2', 'T1'],
      reachable: ['T1', 'T2'],
      selfSupplied: false,
    }
    const { svg } = renderMap(map, supplyState, vi.fn(), supply)
    const armyTerritory = svg.querySelector('[data-territory-id="T1"]')

    if (!armyTerritory) {
      throw new Error('Supply test fixture did not render the army territory')
    }
    clickTarget(svg, armyTerritory)

    const supplyZone = svg.querySelector('g[aria-label="Zone de ravitaillement"]')
    const supplyPath = svg.querySelector('g[aria-label="Ligne de ravitaillement"]')
    expect(supplyZone).toBeInTheDocument()
    expect(supplyZone?.querySelectorAll('[data-supply-territory-id]')).toHaveLength(2)
    expect(supplyZone?.querySelector('path')).toHaveAttribute(
      'fill',
      'url(#supply-zone-hatch)',
    )
    expect(svg.querySelector('#supply-zone-hatch line')).toHaveAttribute(
      'stroke-opacity',
      '0.5',
    )
    expect(svg.querySelector('#supply-zone-hatch line')).toHaveAttribute(
      'stroke',
      '#808080',
    )
    expect(supplyPath).toBeInTheDocument()
    expect(supplyPath?.querySelector('polyline')).toHaveAttribute('points', '75,25 25,25')
    expect(supplyPath).toHaveAttribute('pointer-events', 'none')
  })

  it('renders the reachable zone when a source territory is selected', () => {
    const sourceState: StateData = {
      ...state,
      players: [{ id: 'P1', name: 'One', color: '#a84632' }],
      territories: [
        {
          id: 'T1',
          owner: 'P1',
          resources: 0,
          army: null,
          infrastructures: [{ type: 'castle', level: 1 }],
        },
        { id: 'T2', owner: 'P1', resources: 0, army: null, infrastructures: [] },
      ],
    }
    const sourceZone: SupplyLine = {
      kind: 'source',
      territory: 'T1',
      armyOwner: 'P1',
      armySize: 0,
      rations: 0,
      demand: 0,
      source: 'T1',
      distance: 0,
      path: [],
      reachable: ['T1', 'T2'],
      selfSupplied: false,
    }
    const { firstTerritory, svg } = renderMap(map, sourceState, vi.fn(), sourceZone)
    clickTarget(svg, firstTerritory)

    expect(
      svg.querySelector('g[aria-label="Zone de ravitaillement"]'),
    ).toBeInTheDocument()
    expect(svg.querySelectorAll('[data-supply-territory-id]')).toHaveLength(2)
    expect(
      svg.querySelector('g[aria-label="Ligne de ravitaillement"]'),
    ).not.toBeInTheDocument()
  })

  it('does not render a supply overlay for a selected territory without an army', () => {
    const { firstTerritory, svg } = renderMap(map, state, vi.fn(), {
      kind: 'army',
      territory: 'T1',
      armyOwner: 'P1',
      armySize: 1,
      rations: 1,
      demand: 0,
      source: null,
      distance: 0,
      path: [],
      reachable: ['T1'],
      selfSupplied: true,
    })
    clickTarget(svg, firstTerritory)

    expect(
      svg.querySelector('g[aria-label="Zone de ravitaillement"]'),
    ).not.toBeInTheDocument()
    expect(
      svg.querySelector('g[aria-label="Ligne de ravitaillement"]'),
    ).not.toBeInTheDocument()
  })

  it('does not treat a depot as a supply source', () => {
    const depotState: StateData = {
      ...state,
      players: [{ id: 'P1', name: 'One', color: '#a84632' }],
      territories: [
        {
          ...state.territories[0],
          owner: 'P1',
          infrastructures: [{ type: 'supply_depot', level: 1 }],
        },
        state.territories[1],
      ],
    }
    const { firstTerritory, svg } = renderMap(map, depotState, vi.fn(), {
      kind: 'source',
      territory: 'T1',
      armyOwner: 'P1',
      armySize: 0,
      rations: 0,
      demand: 0,
      source: 'T1',
      distance: 0,
      path: [],
      reachable: ['T1', 'T2'],
      selfSupplied: false,
    })
    clickTarget(svg, firstTerritory)

    expect(
      svg.querySelector('g[aria-label="Zone de ravitaillement"]'),
    ).not.toBeInTheDocument()
  })

  it('does not render a stale supply overlay from another territory', () => {
    const armyState: StateData = {
      ...state,
      players: [{ id: 'P1', name: 'One', color: '#a84632' }],
      territories: [
        {
          ...state.territories[0],
          owner: 'P1',
          army: { owner: 'P1', size: 2, chain: null },
        },
        state.territories[1],
      ],
    }
    const { firstTerritory, svg } = renderMap(map, armyState, vi.fn(), {
      kind: 'army',
      territory: 'T2',
      armyOwner: 'P1',
      armySize: 2,
      rations: 1,
      demand: 1,
      source: 'T2',
      distance: 0,
      path: ['T2'],
      reachable: ['T2'],
      selfSupplied: false,
    })
    clickTarget(svg, firstTerritory)

    expect(
      svg.querySelector('g[aria-label="Zone de ravitaillement"]'),
    ).not.toBeInTheDocument()
    expect(
      svg.querySelector('g[aria-label="Ligne de ravitaillement"]'),
    ).not.toBeInTheDocument()
  })

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
