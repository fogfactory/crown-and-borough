export interface ViewState {
  x: number
  y: number
  k: number
}

export type MapPoint = [number, number]

export const MIN_ZOOM = 0.5
export const MAX_ZOOM = 4
export const DRAG_THRESHOLD = 4
export const ZOOM_BUTTON_FACTOR = 1.4
export const WHEEL_ZOOM_FACTOR = 1.15

export function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum)
}

export function distanceBetween(from: MapPoint, to: MapPoint): number {
  return Math.hypot(to[0] - from[0], to[1] - from[1])
}

export function midpointOf(from: MapPoint, to: MapPoint): MapPoint {
  return [(from[0] + to[0]) / 2, (from[1] + to[1]) / 2]
}

/**
 * Zoom the view by `zoomFactor` keeping the given map-space point fixed,
 * exactly like the wheel handler.
 */
export function zoomAtPoint(
  view: ViewState,
  cursor: MapPoint,
  zoomFactor: number,
  minimum = MIN_ZOOM,
  maximum = MAX_ZOOM,
): ViewState {
  const nextZoom = clamp(view.k * zoomFactor, minimum, maximum)
  const ratio = nextZoom / view.k

  return {
    k: nextZoom,
    x: cursor[0] - (cursor[0] - view.x) * ratio,
    y: cursor[1] - (cursor[1] - view.y) * ratio,
  }
}

/** Zoom centered on the middle of the viewport (used by the +/- buttons). */
export function zoomAtCenter(
  view: ViewState,
  mapWidth: number,
  mapHeight: number,
  zoomFactor: number,
): ViewState {
  return zoomAtPoint(view, [mapWidth / 2, mapHeight / 2], zoomFactor)
}

/**
 * Compute the view for a pinch gesture: the initial gesture midpoint (in map
 * coordinates) stays anchored under the current finger midpoint while the
 * spread between the two fingers drives the zoom. Two-finger panning falls out
 * of the same formula.
 */
export function pinchView(
  initialView: ViewState,
  initialMidpoint: MapPoint,
  initialDistance: number,
  currentMidpoint: MapPoint,
  currentDistance: number,
  minimum = MIN_ZOOM,
  maximum = MAX_ZOOM,
): ViewState {
  if (initialDistance <= 0 || initialView.k <= 0) {
    return initialView
  }

  const scale = clamp(
    (initialView.k * currentDistance) / (initialDistance * initialView.k),
    minimum / initialView.k,
    maximum / initialView.k,
  )
  const mapMidpoint: MapPoint = [
    (initialMidpoint[0] - initialView.x) / initialView.k,
    (initialMidpoint[1] - initialView.y) / initialView.k,
  ]
  const nextZoom = initialView.k * scale

  return {
    k: nextZoom,
    x: currentMidpoint[0] - mapMidpoint[0] * nextZoom,
    y: currentMidpoint[1] - mapMidpoint[1] * nextZoom,
  }
}

/**
 * Which snap state a bottom sheet should take after a vertical drag, given the
 * distance travelled since drag start (positive = downward).
 */
export type SheetSnap = 'peek' | 'half' | 'full'

const SHEET_SNAP_ORDER: SheetSnap[] = ['peek', 'half', 'full']

export function snapFromDrag(current: SheetSnap, deltaY: number): SheetSnap {
  const index = SHEET_SNAP_ORDER.indexOf(current)
  if (deltaY <= -60) {
    return SHEET_SNAP_ORDER[Math.min(index + 1, SHEET_SNAP_ORDER.length - 1)]
  }
  if (deltaY >= 60) {
    return SHEET_SNAP_ORDER[Math.max(index - 1, 0)]
  }
  return current
}

export function nextSheetSnap(current: SheetSnap): SheetSnap {
  return current === 'full' ? 'peek' : 'full'
}
