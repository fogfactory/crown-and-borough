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

function renderMap(onSelect = vi.fn(), props: Record<string, unknown> = {}) {
  const result = render(
    <MapViewer map={map} state={state} onSelect={onSelect} {...props} />,
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
