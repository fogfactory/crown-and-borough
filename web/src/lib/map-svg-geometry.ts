import { clientToMapPoint } from '@/lib/map-gestures'
import { insetPolygon } from '@/lib/region-geometry'
import type { MapData, Point } from '@/types'

export function pointsToPath(points: Point[]): string {
  if (points.length === 0) {
    return ''
  }

  const [first, ...rest] = points
  return `M ${first[0]},${first[1]} ${rest.map(([x, y]) => `L ${x},${y}`).join(' ')} Z`
}

/**
 * Path of one gradient ring: each boundary loop paired with its inset twin
 * (holes expand outward so the ring stays inside the region), filled with
 * the even-odd rule.
 */
export function regionRingPath(
  loops: Point[][],
  inset: number,
  isHole: (loop: Point[]) => boolean,
): string {
  return loops
    .map((loop) => {
      const offset = insetPolygon(loop, isHole(loop) ? -inset : inset)
      return `${pointsToPath(loop)} ${pointsToPath(offset)}`
    })
    .join(' ')
}

export function pointKey([x, y]: Point): string {
  return `${x},${y}`
}

export function edgeKey(from: Point, to: Point): string {
  const fromKey = pointKey(from)
  const toKey = pointKey(to)

  return fromKey < toKey ? `${fromKey}|${toKey}` : `${toKey}|${fromKey}`
}

export function polygonEdges(points: Point[]): Array<[Point, Point]> {
  if (points.length < 2) {
    return []
  }

  return points.map((point, index): [Point, Point] => [
    point,
    points[(index + 1) % points.length],
  ])
}

export function segmentsToPath(segments: Array<[Point, Point]>): string {
  return segments
    .map(([[fromX, fromY], [toX, toY]]) => `M ${fromX},${fromY} L ${toX},${toY}`)
    .join(' ')
}

export function splitBoundaryPaths(
  points: Point[],
  passableBoundaryKeys: Set<string>,
): { solidPath: string; passablePath: string } {
  const solidSegments: Array<[Point, Point]> = []
  const passableSegments: Array<[Point, Point]> = []

  for (const segment of polygonEdges(points)) {
    const [from, to] = segment
    if (passableBoundaryKeys.has(edgeKey(from, to))) {
      passableSegments.push(segment)
    } else {
      solidSegments.push(segment)
    }
  }

  return {
    solidPath: segmentsToPath(solidSegments),
    passablePath: segmentsToPath(passableSegments),
  }
}

export function centroid(points: Point[]): Point {
  if (points.length === 0) {
    return [0, 0]
  }

  const total = points.reduce((sum, [x, y]) => ({ x: sum.x + x, y: sum.y + y }), {
    x: 0,
    y: 0,
  })

  return [total.x / points.length, total.y / points.length]
}

export function polygonArea(points: Point[]): number {
  if (points.length < 3) {
    return 0
  }

  let twiceArea = 0
  for (let index = 0; index < points.length; index += 1) {
    const [x, y] = points[index]
    const [nextX, nextY] = points[(index + 1) % points.length]
    twiceArea += x * nextY - nextX * y
  }
  return Math.abs(twiceArea) / 2
}

export function meanTerritoryArea(territories: MapData['territories']): number {
  if (territories.length === 0) {
    return 0
  }

  return (
    territories.reduce((total, territory) => total + polygonArea(territory.points), 0) /
    territories.length
  )
}

export function clientToSvgPoint(
  svg: SVGSVGElement,
  clientX: number,
  clientY: number,
  viewWidth: number,
  viewHeight: number,
  originX: number,
  originY: number,
): Point {
  const bounds = svg.getBoundingClientRect()
  const [x, y] = clientToMapPoint(clientX, clientY, viewWidth, viewHeight, bounds)
  return [x + originX, y + originY]
}

export function getTerritoryIdFromTarget(target: EventTarget | null): string | null {
  if (!(target instanceof Element)) {
    return null
  }

  return (
    target.closest<SVGPathElement>('[data-territory-id]')?.dataset.territoryId ?? null
  )
}
