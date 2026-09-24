import {
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
  type RefObject,
} from 'react'

import {
  DRAG_THRESHOLD,
  WHEEL_ZOOM_FACTOR,
  distanceBetween,
  midpointOf,
  pinchView,
  viewportScale,
  zoomAtCenter,
  zoomAtPoint,
  type MapPoint,
  type ViewState,
} from '@/lib/map-gestures'
import { clientToSvgPoint, getTerritoryIdFromTarget } from '@/lib/map-svg-geometry'

interface TrackedPointer {
  start: MapPoint
  last: MapPoint
}

interface GestureState {
  primaryId: number
  pointers: Map<number, TrackedPointer>
  mode: 'pending' | 'pan' | 'pinch'
  dragged: boolean
  territoryId: string | null
  initialView: ViewState
  initialDistance: number
  initialMidpoint: MapPoint
  /** Drag threshold in map units, derived from on-screen pixels. */
  threshold: number
}

type SvgPointerHandler = (event: ReactPointerEvent<SVGSVGElement>) => void

export interface MapGesturesOptions {
  /** SVG viewBox size, frame padding included. */
  viewWidth: number
  viewHeight: number
  /** Frame padding on each side; the viewBox starts at `-bandMargin`. */
  bandMarginX: number
  bandMarginY: number
  mapWidth: number
  mapHeight: number
  /** A press released without dragging: the territory under it, or null. */
  onTap: (territoryId: string | null) => void
}

export interface MapGestures {
  svgRef: RefObject<SVGSVGElement | null>
  view: ViewState
  isDragging: boolean
  pointerHandlers: {
    onPointerDown: SvgPointerHandler
    onPointerMove: SvgPointerHandler
    onPointerUp: SvgPointerHandler
    onPointerCancel: SvgPointerHandler
  }
  /** Zoom around the map center; a factor of 0 resets the view. */
  zoomBy: (zoomFactor: number) => void
}

/**
 * Pan and zoom state for the map SVG: mouse and touch drag with a pixel
 * threshold separating taps from pans, two-finger pinch, wheel zoom at the
 * cursor and the on-screen zoom buttons.
 */
export function useMapGestures({
  viewWidth,
  viewHeight,
  bandMarginX,
  bandMarginY,
  mapWidth,
  mapHeight,
  onTap,
}: MapGesturesOptions): MapGestures {
  const svgRef = useRef<SVGSVGElement>(null)
  const gestureRef = useRef<GestureState | null>(null)
  const [view, setView] = useState<ViewState>({ x: 0, y: 0, k: 1 })
  const [isDragging, setIsDragging] = useState(false)

  useEffect(() => {
    const svg = svgRef.current
    if (!svg) {
      return
    }

    const handleWheel = (event: WheelEvent) => {
      event.preventDefault()
      const cursor = clientToSvgPoint(
        svg,
        event.clientX,
        event.clientY,
        viewWidth,
        viewHeight,
        -bandMarginX,
        -bandMarginY,
      )
      const zoomFactor = event.deltaY < 0 ? WHEEL_ZOOM_FACTOR : 1 / WHEEL_ZOOM_FACTOR

      setView((current) => zoomAtPoint(current, cursor, zoomFactor))
    }

    svg.addEventListener('wheel', handleWheel, { passive: false })
    return () => svg.removeEventListener('wheel', handleWheel)
  }, [mapHeight, mapWidth, viewHeight, viewWidth, bandMarginX, bandMarginY])

  const handlePointerDown = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (event.button !== 0) {
      if (event.button === 1) {
        event.preventDefault()
      }
      return
    }

    const point = clientToSvgPoint(
      event.currentTarget,
      event.clientX,
      event.clientY,
      viewWidth,
      viewHeight,
      -bandMarginX,
      -bandMarginY,
    )
    const bounds = event.currentTarget.getBoundingClientRect()
    const mapUnitsPerPx =
      1 / viewportScale(viewWidth, viewHeight, bounds.width || 1, bounds.height || 1)
    const threshold = Math.max(
      DRAG_THRESHOLD,
      (event.pointerType === 'touch' ? 8 : DRAG_THRESHOLD) * mapUnitsPerPx,
    )
    event.currentTarget.setPointerCapture(event.pointerId)
    event.preventDefault()

    const gesture = gestureRef.current
    if (!gesture) {
      gestureRef.current = {
        primaryId: event.pointerId,
        pointers: new Map([[event.pointerId, { start: point, last: point }]]),
        mode: 'pending',
        dragged: false,
        territoryId: getTerritoryIdFromTarget(event.target),
        initialView: view,
        initialDistance: 0,
        initialMidpoint: point,
        threshold,
      }
      return
    }

    if (gesture.pointers.has(event.pointerId)) {
      return
    }

    gesture.pointers.set(event.pointerId, { start: point, last: point })
    if (gesture.pointers.size >= 2) {
      const [first, second] = Array.from(gesture.pointers.values())
      gesture.mode = 'pinch'
      gesture.dragged = true
      gesture.initialView = view
      gesture.initialDistance = Math.max(distanceBetween(first.last, second.last), 1)
      gesture.initialMidpoint = midpointOf(first.last, second.last)
      setIsDragging(true)
    }
  }

  const handlePointerMove = (event: ReactPointerEvent<SVGSVGElement>) => {
    const gesture = gestureRef.current
    if (!gesture) {
      return
    }
    const tracked = gesture.pointers.get(event.pointerId)
    if (!tracked) {
      return
    }

    const point = clientToSvgPoint(
      event.currentTarget,
      event.clientX,
      event.clientY,
      viewWidth,
      viewHeight,
      -bandMarginX,
      -bandMarginY,
    )
    const deltaX = point[0] - tracked.last[0]
    const deltaY = point[1] - tracked.last[1]
    tracked.last = point

    if (gesture.mode === 'pending') {
      const primary = gesture.pointers.get(gesture.primaryId)
      if (!primary) {
        return
      }
      if (distanceBetween(primary.start, primary.last) < gesture.threshold) {
        return
      }
      gesture.mode = 'pan'
      gesture.dragged = true
      setIsDragging(true)
    }

    if (gesture.mode === 'pinch') {
      const [first, second] = Array.from(gesture.pointers.values())
      if (!first || !second) {
        return
      }
      const currentMidpoint = midpointOf(first.last, second.last)
      const currentDistance = distanceBetween(first.last, second.last)
      setView(
        pinchView(
          gesture.initialView,
          gesture.initialMidpoint,
          gesture.initialDistance,
          currentMidpoint,
          currentDistance,
        ),
      )
      return
    }

    if (gesture.mode === 'pan' && (deltaX !== 0 || deltaY !== 0)) {
      setView((current) => ({
        ...current,
        x: current.x + deltaX,
        y: current.y + deltaY,
      }))
    }
  }

  const endGesture = (
    event: ReactPointerEvent<SVGSVGElement>,
    allowSelection: boolean,
  ) => {
    const gesture = gestureRef.current
    if (!gesture || !gesture.pointers.has(event.pointerId)) {
      return
    }

    const wasPending = gesture.mode === 'pending'
    gesture.pointers.delete(event.pointerId)
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }

    if (gesture.pointers.size === 0) {
      if (allowSelection && wasPending && !gesture.dragged) {
        onTap(gesture.territoryId)
      }
      gestureRef.current = null
      setIsDragging(false)
      return
    }

    if (gesture.mode === 'pinch' && gesture.pointers.size === 1) {
      gesture.mode = 'pan'
    }
  }

  const handlePointerUp = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (event.button !== 0) {
      return
    }
    endGesture(event, true)
  }

  const handlePointerCancel = (event: ReactPointerEvent<SVGSVGElement>) => {
    endGesture(event, false)
  }

  const zoomBy = (zoomFactor: number) => {
    if (zoomFactor === 0) {
      setView({ x: 0, y: 0, k: 1 })
      return
    }
    setView((current) => zoomAtCenter(current, mapWidth, mapHeight, zoomFactor))
  }

  return {
    svgRef,
    view,
    isDragging,
    pointerHandlers: {
      onPointerDown: handlePointerDown,
      onPointerMove: handlePointerMove,
      onPointerUp: handlePointerUp,
      onPointerCancel: handlePointerCancel,
    },
    zoomBy,
  }
}
