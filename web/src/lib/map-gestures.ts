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

/**
 * Uniform scale used when a map of `mapWidth × mapHeight` map units is fitted
 * inside a `pixelWidth × pixelHeight` viewport with "meet" semantics: the map
 * is never stretched, only scaled down to the most constrained axis.
 */
export function viewportScale(
  mapWidth: number,
  mapHeight: number,
  pixelWidth: number,
  pixelHeight: number,
): number {
  if (mapWidth <= 0 || mapHeight <= 0 || pixelWidth <= 0 || pixelHeight <= 0) {
    return 1
  }
  return Math.min(pixelWidth / mapWidth, pixelHeight / mapHeight)
}

/**
 * Convert a browser client position into map coordinates for an SVG that
 * preserves its aspect ratio ("xMidYMid meet"): the map is scaled uniformly
 * and centered in the element, so any leftover space becomes letterboxing.
 */
export function clientToMapPoint(
  clientX: number,
  clientY: number,
  mapWidth: number,
  mapHeight: number,
  bounds: { left: number; top: number; width: number; height: number },
): MapPoint {
  const scale = viewportScale(mapWidth, mapHeight, bounds.width || 1, bounds.height || 1)
  const offsetX = ((bounds.width || 1) - mapWidth * scale) / 2
  const offsetY = ((bounds.height || 1) - mapHeight * scale) / 2

  return [
    (clientX - bounds.left - offsetX) / scale,
    (clientY - bounds.top - offsetY) / scale,
  ]
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

/** Bounding box in map units: [minX, minY, maxX, maxY]. */
export type MapBounds = [number, number, number, number]

/**
 * View that frames `bounds` (plus `padding` map units around it) in the
 * middle of the viewport. The SVG keeps its aspect ratio ("meet"), so the
 * visible area may be wider or taller than the viewBox: `elementWidth` and
 * `elementHeight` (pixels) account for it. The view never zooms out below the
 * whole map.
 */
export function frameView(
  bounds: MapBounds,
  viewWidth: number,
  viewHeight: number,
  mapWidth: number,
  mapHeight: number,
  elementWidth: number,
  elementHeight: number,
  padding = 0,
): ViewState {
  const [minX, minY, maxX, maxY] = bounds
  const ratio = elementWidth > 0 && elementHeight > 0 ? elementWidth / elementHeight : 0
  const visibleWidth = ratio > 0 ? Math.max(viewWidth, viewHeight * ratio) : viewWidth
  const visibleHeight = ratio > 0 ? Math.max(viewHeight, viewWidth / ratio) : viewHeight
  const width = Math.max(maxX - minX, 1) + padding * 2
  const height = Math.max(maxY - minY, 1) + padding * 2
  const k = clamp(Math.min(visibleWidth / width, visibleHeight / height), 1, MAX_ZOOM)
  const centerX = (minX + maxX) / 2
  const centerY = (minY + maxY) / 2

  return {
    k,
    x: keepFilled(mapWidth / 2 - centerX * k, k, viewWidth, mapWidth, visibleWidth),
    y: keepFilled(mapHeight / 2 - centerY * k, k, viewHeight, mapHeight, visibleHeight),
  }
}

/**
 * Shifts an offset so the drawing (the map and its frame band, `viewSize`
 * wide and centered on the map) covers the visible span when it can,
 * instead of leaving empty paper past its edge; a drawing smaller than the
 * visible span is centered.
 */
function keepFilled(
  offset: number,
  k: number,
  viewSize: number,
  mapSize: number,
  visibleSize: number,
): number {
  const center = mapSize / 2
  const visibleStart = center - visibleSize / 2
  const visibleEnd = center + visibleSize / 2
  const drawingStart = center - viewSize / 2
  const drawingEnd = center + viewSize / 2
  if ((drawingEnd - drawingStart) * k < visibleSize) return center * (1 - k)
  return clamp(offset, visibleEnd - drawingEnd * k, visibleStart - drawingStart * k)
}
