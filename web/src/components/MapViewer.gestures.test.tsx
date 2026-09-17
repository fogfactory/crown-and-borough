import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { MapViewer } from '@/components/MapViewer'
import type { MapData, StateData } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Alpilles',
      terrain: 'plain',
      village: false,
      points: [
        [0, 0],
        [50, 0],
        [50, 50],
        [0, 50],
      ],
      adjacencies: ['BRU'],
      impassable: [],
    },
    {
      id: 'BRU',
      name: 'Brisecote',
      terrain: 'forest',
      village: false,
      points: [
        [50, 0],
        [100, 0],
        [100, 50],
        [50, 50],
      ],
      adjacencies: ['ROS'],
      impassable: [],
    },
  ],
}

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [],
  territories: [
    { id: 'ROS', owner: 'P1', resources: 0, army: null, infrastructures: [] },
    { id: 'BRU', owner: 'P2', resources: 0, army: null, infrastructures: [] },
  ],
  nobles: [],
}

function renderMap(
  onSelect = vi.fn(),
  props: Record<string, unknown> = {},
  fixtures: { map?: MapData; state?: StateData } = {},
) {
  const result = render(
    <MapViewer
      map={fixtures.map ?? map}
      state={fixtures.state ?? state}
      onSelect={onSelect}
      {...props}
    />,
  )
  const svg = result.container.querySelector(
    'svg[aria-label="Territory map"]',
  ) as SVGSVGElement | null
  const firstTerritory = result.container.querySelector(
    '[data-territory-id="ROS"]',
  ) as SVGPathElement | null
  const mapGroup = svg?.querySelector('g[transform]') ?? null

  if (!svg || !firstTerritory || !mapGroup) {
    throw new Error('Map test fixture did not render the expected SVG elements')
  }

  return { ...result, firstTerritory, mapGroup, onSelect, svg }
}

function touch(
  clientX: number,
  clientY = 100,
  pointerId = 1,
): Record<string, string | number> {
  return { button: 0, clientX, clientY, pointerId, pointerType: 'touch' }
}

describe('MapViewer map geometry', () => {
  it('preserves the map aspect ratio instead of stretching to the container', () => {
    const { svg } = renderMap()

    expect(svg).toHaveAttribute('preserveAspectRatio', 'xMidYMid meet')
  })
})

describe('MapViewer terrain textures', () => {
  it('defines a pattern per terrain and overlays it on the territories', () => {
    const { svg } = renderMap()

    for (const terrain of ['plain', 'forest', 'hill', 'mountain', 'swamp']) {
      expect(svg.querySelector(`#terrain-${terrain}`)).toBeInTheDocument()
    }

    const textureGroup = svg.querySelector('g[aria-label="Terrain textures"]')
    expect(textureGroup).toBeInTheDocument()
    expect(textureGroup).toHaveAttribute('pointer-events', 'none')
    const overlays = textureGroup?.querySelectorAll('path') ?? []
    expect(overlays.length).toBeGreaterThan(0)
    expect(overlays[0]).toHaveAttribute('fill', 'url(#terrain-plain)')
    // The overlay must not intercept pointer events: the terrain layer below
    // handles selection.
    expect(textureGroup?.querySelector('path')).not.toHaveAttribute('data-territory-id')
  })

  it('doubles the pattern tiles for forest, hill and mountain and keeps swamp', () => {
    const { svg } = renderMap()

    // Pattern geometry scales with the annotation scale of the fixture.
    const referenceMeanArea = (1000 * 700) / (8 * 4 + 4 * (4 + 1))
    const s = Math.sqrt((50 * 50) / referenceMeanArea)

    const tile = (id: string) => svg.querySelector(`#${id}`)?.getAttribute('width')

    expect(Number(tile('terrain-plain'))).toBeCloseTo(14 * s)
    expect(Number(tile('terrain-forest'))).toBeCloseTo(20 * s)
    expect(Number(tile('terrain-hill'))).toBeCloseTo(20 * s)
    expect(Number(tile('terrain-mountain'))).toBeCloseTo(18 * s)
    expect(Number(tile('terrain-swamp'))).toBeCloseTo(10 * s)
  })

  it('draws the plain texture as three tiny slashes', () => {
    const { svg } = renderMap()

    const referenceMeanArea = (1000 * 700) / (8 * 4 + 4 * (4 + 1))
    const s = Math.sqrt((50 * 50) / referenceMeanArea)
    const slashes = svg.querySelectorAll('#terrain-plain line')
    expect(slashes).toHaveLength(3)
    slashes.forEach((slash) => {
      expect(
        Math.abs(Number(slash.getAttribute('x2')) - Number(slash.getAttribute('x1'))),
      ).toBeCloseTo(1.2 * s)
      expect(
        Math.abs(Number(slash.getAttribute('y2')) - Number(slash.getAttribute('y1'))),
      ).toBeCloseTo(1.2 * s)
    })
  })
})

describe('MapViewer infrastructure ownership', () => {
  const ownedState: StateData = {
    ...state,
    players: [
      { id: 'P1', name: 'One', color: '#a84632' },
      { id: 'P2', name: 'Two', color: '#2d5f9e' },
    ],
    territories: [
      {
        id: 'ROS',
        owner: 'P1',
        resources: 0,
        army: null,
        infrastructures: [{ type: 'castle', level: 1 }],
      },
      {
        id: 'BRU',
        owner: null,
        resources: 0,
        army: null,
        infrastructures: [{ type: 'village', level: 2 }],
      },
    ],
  }

  function markerFillPath(
    svg: SVGSVGElement,
    dPrefix: string,
    fill: string,
  ): SVGPathElement | undefined {
    const livingLayer = svg.querySelector('g[aria-label="Live layer"]')
    // Each glyph renders three times (halo, owner fill, dark casing); only
    // the owner-fill layer carries the territory color.
    return Array.from(livingLayer?.querySelectorAll('path') ?? []).find(
      (path) =>
        path.getAttribute('d')?.startsWith(dPrefix) && path.getAttribute('fill') === fill,
    )
  }

  it('fills owned settlement glyphs with the owner color', () => {
    const { svg } = renderMap(vi.fn(), {}, { map, state: ownedState })

    expect(markerFillPath(svg, 'M255.95 27.11', '#a84632')).toBeDefined()
  })

  it('keeps unowned settlements in the neutral parchment tone', () => {
    const { svg } = renderMap(vi.fn(), {}, { map, state: ownedState })

    expect(markerFillPath(svg, 'M109.902 35.87', '#efe6d0')).toBeDefined()
  })

  it('renders the village glyph from game-icons', () => {
    const { svg } = renderMap(vi.fn(), {}, { map, state: ownedState })

    expect(markerFillPath(svg, 'M109.902 35.87', '#efe6d0')).toBeInTheDocument()
  })

  it('cases the colored control borders in dark under the player color', () => {
    const { svg } = renderMap(vi.fn(), {}, { map, state: ownedState })
    const controlGroup = svg.querySelector('g[aria-label="Territorial control"]')
    const casing = controlGroup?.querySelector('path[stroke="#30291f"]')

    expect(casing).toBeInTheDocument()
    expect(casing).toHaveAttribute('stroke-width', '11')
    expect(casing?.getAttribute('stroke-opacity')).toBe('0.55')
  })
})

describe('MapViewer touch gestures', () => {
  it('selects a territory with a touch tap', () => {
    const { firstTerritory, onSelect, svg } = renderMap()

    fireEvent.pointerDown(firstTerritory, touch(100))
    fireEvent.pointerUp(svg, touch(100))

    expect(onSelect).toHaveBeenLastCalledWith('ROS')
  })

  it('toggles the selection with a second touch tap', () => {
    const { firstTerritory, onSelect, svg } = renderMap()

    fireEvent.pointerDown(firstTerritory, touch(100))
    fireEvent.pointerUp(svg, touch(100))
    fireEvent.pointerDown(firstTerritory, touch(100))
    fireEvent.pointerUp(svg, touch(100))

    expect(onSelect).toHaveBeenLastCalledWith(null)
  })

  it('keeps touch panning without selecting', async () => {
    const { firstTerritory, onSelect, svg } = renderMap()

    fireEvent.pointerDown(firstTerritory, touch(100))
    fireEvent.pointerMove(svg, { ...touch(300), buttons: 1 })
    fireEvent.pointerUp(svg, touch(300))

    expect(onSelect).not.toHaveBeenCalled()
    await waitFor(() => {
      expect(svg.querySelector('g[transform]')).not.toHaveAttribute(
        'transform',
        'translate(0 0) scale(1)',
      )
    })
  })

  it('does not select when a two-finger gesture ends without zooming', () => {
    const { firstTerritory, onSelect, svg } = renderMap()

    fireEvent.pointerDown(firstTerritory, touch(100))
    fireEvent.pointerDown(svg, touch(120, 100, 2))
    fireEvent.pointerUp(svg, touch(120, 100, 2))
    fireEvent.pointerUp(svg, touch(100, 100, 1))

    expect(onSelect).not.toHaveBeenCalled()
  })

  it('zooms with a pinch spreading the fingers apart', async () => {
    const { firstTerritory, svg } = renderMap()

    fireEvent.pointerDown(firstTerritory, touch(100))
    fireEvent.pointerDown(svg, touch(200, 100, 2))
    fireEvent.pointerMove(svg, { ...touch(350, 100, 2), buttons: 3 })

    await waitFor(() => {
      const transform = svg.querySelector('g[transform]')?.getAttribute('transform')
      expect(transform).toContain('scale(2.5)')
    })
  })
})

describe('MapViewer zoom controls', () => {
  it('zooms in and out from the on-screen buttons', async () => {
    const { container } = renderMap()

    fireEvent.click(screen.getByRole('button', { name: 'Zoom in' }))

    await waitFor(() => {
      const transform = container.querySelector('g[transform]')?.getAttribute('transform')
      expect(transform).toContain('scale(1.4)')
    })

    fireEvent.click(screen.getByRole('button', { name: 'Zoom out' }))

    await waitFor(() => {
      const transform = container.querySelector('g[transform]')?.getAttribute('transform')
      expect(transform).toContain('scale(1)')
    })
  })

  it('recents the view with the recenter button', async () => {
    const { container, svg } = renderMap()

    fireEvent.click(screen.getByRole('button', { name: 'Zoom in' }))
    await waitFor(() => {
      expect(svg.querySelector('g[transform]')?.getAttribute('transform')).toContain(
        'scale(1.4)',
      )
    })

    fireEvent.click(screen.getByRole('button', { name: 'Recenter the map' }))
    await waitFor(() => {
      expect(svg.querySelector('g[transform]')).toHaveAttribute(
        'transform',
        'translate(0 0) scale(1)',
      )
    })
    expect(container).toBeDefined()
  })
})

describe('MapViewer legend overlay', () => {
  it('opens the legend from the map toggle and forwards intention changes', () => {
    const onToggleIntentions = vi.fn()
    renderMap(vi.fn(), { showIntentions: true, onToggleIntentions })

    fireEvent.click(screen.getByRole('button', { name: 'Show legend' }))
    expect(screen.getByText('Legend')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('checkbox'))
    expect(onToggleIntentions).toHaveBeenCalledWith(false)

    fireEvent.click(screen.getByRole('button', { name: 'Hide legend' }))
    expect(screen.queryByText('Legend')).not.toBeInTheDocument()
  })

  it('hides the toggle when no intention handler is provided', () => {
    renderMap()

    expect(screen.queryByRole('button', { name: 'Show legend' })).not.toBeInTheDocument()
  })
})
