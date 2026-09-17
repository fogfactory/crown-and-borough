import { describe, expect, it } from 'vitest'

import {
  MAX_ZOOM,
  MIN_ZOOM,
  clamp,
  clientToMapPoint,
  distanceBetween,
  midpointOf,
  nextSheetSnap,
  pinchView,
  snapFromDrag,
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

describe('sheet snapping', () => {
  it('expands on an upward drag and collapses on a downward drag', () => {
    expect(snapFromDrag('half', -100)).toBe('full')
    expect(snapFromDrag('half', 100)).toBe('peek')
    expect(snapFromDrag('half', -10)).toBe('half')
    expect(snapFromDrag('peek', -100)).toBe('half')
    expect(snapFromDrag('full', 100)).toBe('half')
  })

  it('never leaves the snap range', () => {
    expect(snapFromDrag('full', -100)).toBe('full')
    expect(snapFromDrag('peek', 100)).toBe('peek')
  })

  it('toggles between peek and full with the chevron', () => {
    expect(nextSheetSnap('half')).toBe('full')
    expect(nextSheetSnap('peek')).toBe('full')
    expect(nextSheetSnap('full')).toBe('peek')
  })
})
