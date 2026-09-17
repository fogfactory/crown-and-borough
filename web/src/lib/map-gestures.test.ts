import { describe, expect, it } from 'vitest'

import {
  MAX_ZOOM,
  MIN_ZOOM,
  clamp,
  distanceBetween,
  midpointOf,
  nextSheetSnap,
  pinchView,
  snapFromDrag,
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
