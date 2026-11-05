import { describe, expect, it } from 'vitest'

import {
  MAX_ZOOM,
  MIN_ZOOM,
  clamp,
  clientToMapPoint,
  distanceBetween,
  frameView,
  midpointOf,
  pinchView,
  viewportScale,
  zoomAtCenter,
  zoomAtPoint,
  type ViewState,
} from '@/lib/map-gestures'

const baseView: ViewState = { x: 0, y: 0, k: 1 }

describe('clamp', () => {
  it('bounds the value between minimum and maximum', () => {
    expect(clamp(5, 0, 10)).toBe(5)
    expect(clamp(-1, 0, 10)).toBe(0)
    expect(clamp(11, 0, 10)).toBe(10)
  })
})

describe('geometry helpers', () => {
  it('computes distances and midpoints', () => {
    expect(distanceBetween([0, 0], [3, 4])).toBe(5)
    expect(midpointOf([10, 20], [30, 40])).toEqual([20, 30])
  })
})

describe('viewportScale', () => {
  it('uses the most constrained axis', () => {
    expect(viewportScale(100, 50, 200, 50)).toBe(1)
    expect(viewportScale(100, 50, 200, 100)).toBe(2)
    expect(viewportScale(100, 50, 50, 300)).toBe(0.5)
  })

  it('falls back to 1 for degenerate sizes', () => {
    expect(viewportScale(0, 50, 200, 100)).toBe(1)
    expect(viewportScale(100, 50, 0, 100)).toBe(1)
  })
})

describe('clientToMapPoint', () => {
  const bounds = { left: 0, top: 0, width: 200, height: 100 }

  it('maps element corners onto map corners in a matching container', () => {
    expect(clientToMapPoint(0, 0, 100, 50, bounds)).toEqual([0, 0])
    expect(clientToMapPoint(200, 100, 100, 50, bounds)).toEqual([100, 50])
  })

  it('maps the element center onto the map center', () => {
    expect(clientToMapPoint(100, 50, 100, 50, bounds)).toEqual([50, 25])
  })

  it('accounts for horizontal letterboxing in a narrow container', () => {
    const narrow = { left: 0, top: 0, width: 100, height: 300 }
    const scale = 1
    const offsetY = (300 - 50 * scale) / 2

    // Top-left of the letterboxed map content.
    expect(clientToMapPoint(0, offsetY, 100, 50, narrow)).toEqual([0, 0])
    // Bottom-right of the letterboxed map content.
    expect(clientToMapPoint(100, offsetY + 50, 100, 50, narrow)).toEqual([100, 50])
    // Element corners fall outside the map (negative map coordinates).
    expect(clientToMapPoint(0, 0, 100, 50, narrow)).toEqual([0, -offsetY])
  })

  it('accounts for the element offset within the page', () => {
    const shifted = { left: 40, top: 60, width: 200, height: 100 }
    expect(clientToMapPoint(40, 60, 100, 50, shifted)).toEqual([0, 0])
    expect(clientToMapPoint(140, 110, 100, 50, shifted)).toEqual([50, 25])
  })
})

describe('zoomAtPoint', () => {
  it('keeps the cursor anchored while zooming in', () => {
    const next = zoomAtPoint(baseView, [50, 50], 2)

    expect(next.k).toBe(2)
    expect(next.x).toBe(-50)
    expect(next.y).toBe(-50)
  })

  it('keeps the cursor anchored while zooming out of a panned view', () => {
    const next = zoomAtPoint({ x: -50, y: -50, k: 2 }, [50, 50], 0.5)

    expect(next.k).toBe(1)
    expect(next.x).toBe(0)
    expect(next.y).toBe(0)
  })

  it('clamps the zoom within the allowed range', () => {
    const zoomedIn = zoomAtPoint(baseView, [50, 50], 100)

    expect(zoomedIn.k).toBe(MAX_ZOOM)

    const zoomedOut = zoomAtPoint(baseView, [50, 50], 0.001)

    expect(zoomedOut.k).toBe(MIN_ZOOM)
  })

  it('zooms around the viewport center', () => {
    const next = zoomAtCenter(baseView, 100, 100, 2)

    expect(next.k).toBe(2)
    expect(next.x).toBe(-50)
    expect(next.y).toBe(-50)
  })
})

describe('pinchView', () => {
  it('scales the view around the gesture midpoint', () => {
    const next = pinchView(baseView, [50, 50], 40, [50, 50], 80)

    expect(next.k).toBe(2)
    expect(next.x).toBe(-50)
    expect(next.y).toBe(-50)
  })

  it('pans when the fingers translate without spreading', () => {
    const next = pinchView(baseView, [50, 50], 40, [60, 70], 40)

    expect(next.k).toBe(1)
    expect(next.x).toBe(10)
    expect(next.y).toBe(20)
  })

  it('clamps the zoom while pinching far beyond the maximum', () => {
    const next = pinchView(baseView, [50, 50], 40, [50, 50], 4000)

    expect(next.k).toBe(MAX_ZOOM)
    expect(next.x).toBe(50 - 50 * MAX_ZOOM)
  })

  it('clamps the zoom while collapsing below the minimum', () => {
    const next = pinchView(baseView, [50, 50], 40, [50, 50], 0.4)

    expect(next.k).toBe(MIN_ZOOM)
  })

  it('returns the initial view for a degenerate gesture', () => {
    expect(pinchView(baseView, [50, 50], 0, [50, 50], 80)).toEqual(baseView)
  })
})

describe('frameView', () => {
  it('centers and zooms on the framed bounds', () => {
    // A 100 × 100 viewBox on a square element: a 20-unit box padded by 5
    // fills it at 100 / 30, and a tiny box stops at the maximum zoom.
    const view = frameView([40, 40, 60, 60], 100, 100, 100, 100, 400, 400, 5)

    expect(view.k).toBeCloseTo(100 / 30)
    expect(frameView([49, 49, 50, 50], 100, 100, 100, 100, 400, 400).k).toBe(MAX_ZOOM)
    // The center of the box lands on the center of the map.
    expect(50 * view.k + view.x).toBeCloseTo(50)
    expect(50 * view.k + view.y).toBeCloseTo(50)
  })

  it('uses the extra height of a portrait element', () => {
    // A tall box framed in a landscape viewBox shown on a phone.
    const landscape = frameView([0, 0, 50, 100], 200, 100, 200, 100, 0, 0)
    const portrait = frameView([0, 0, 50, 100], 200, 100, 200, 100, 375, 600)

    expect(landscape.k).toBe(1)
    expect(portrait.k).toBeGreaterThan(landscape.k)
  })

  it('keeps the map filling the screen when the box is at its edge', () => {
    // A box in the bottom-right corner of a 100 × 100 map, zoomed × 2.5.
    const view = frameView([80, 80, 100, 100], 100, 100, 100, 100, 400, 400, 10)

    // The map's bottom-right corner stays on the viewport's corner.
    expect(100 * view.k + view.x).toBeCloseTo(100)
    expect(100 * view.k + view.y).toBeCloseTo(100)
  })

  it('centers the map on an axis where it is smaller than the screen', () => {
    // A box near the top of a square map shown on a tall phone, at zoom 1.
    const view = frameView([0, 0, 100, 40], 100, 100, 100, 100, 300, 900)

    expect(view.k).toBe(1)
    expect(view.y).toBe(0)
  })

  it('never zooms out below the whole map', () => {
    expect(frameView([0, 0, 500, 500], 100, 100, 100, 100, 100, 100).k).toBe(1)
  })
})
