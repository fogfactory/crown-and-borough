import type { Point, Region, Territory } from '@/types'

export interface RegionOutline {
  /** Closed boundary rings of the region: outer borders and inter-region borders. */
  loops: Point[][]
  /** Boundary segments lying on the outer map border. */
  outerSegments: Array<[Point, Point]>
  /** Open polylines chaining the outer-border segments of the region. */
  outerLoops: Point[][]
}

export interface RegionOutlineResult {
  outlines: Map<string, RegionOutline>
  bounds: { minX: number; minY: number; maxX: number; maxY: number }
}

function pointKey([x, y]: Point): string {
  return `${x},${y}`
}

function pointsEqual(first: Point, second: Point): boolean {
  return first[0] === second[0] && first[1] === second[1]
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

function signedArea(points: Point[]): number {
  let twiceArea = 0
  for (let index = 0; index < points.length; index += 1) {
    const [x, y] = points[index]
    const [nextX, nextY] = points[(index + 1) % points.length]
    twiceArea += x * nextY - nextX * y
  }
  return twiceArea / 2
}

function counterClockwise(points: Point[]): Point[] {
  return signedArea(points) < 0 ? [...points].reverse() : [...points]
}

/**
 * Point-in-polygon test (ray casting) used to locate the territory enclosed
 * by a chained boundary loop.
 */
export function polygonContains(points: Point[], test: Point): boolean {
  let inside = false
  for (let index = 0, j = points.length - 1; index < points.length; j = index++) {
    const [xi, yi] = points[index]
    const [xj, yj] = points[j]
    const intersects =
      yi > test[1] !== yj > test[1] &&
      test[0] < ((xj - xi) * (test[1] - yi)) / (yj - yi) + xi
    if (intersects) {
      inside = !inside
    }
  }
  return inside
}

function inwardNormal(from: Point, to: Point): Point {
  const dx = to[0] - from[0]
  const dy = to[1] - from[1]
  const length = Math.hypot(dx, dy) || 1
  return [-dy / length, dx / length]
}

/**
 * Offset a closed polygon by `distance` map units toward its interior
 * (negative distances expand it). Vertices join with a miter, clamped to
 * three times the distance to avoid spikes on near-degenerate corners.
 */
export function insetPolygon(points: Point[], distance: number): Point[] {
  if (points.length < 3 || distance === 0) {
    return [...points]
  }
  const ring = counterClockwise(points)
  const count = ring.length
  const limit = Math.abs(distance) * 3
  const result: Point[] = []
  for (let index = 0; index < count; index += 1) {
    const previous = ring[(index - 1 + count) % count]
    const current = ring[index]
    const next = ring[(index + 1) % count]
    const first = inwardNormal(previous, current)
    const second = inwardNormal(current, next)
    const cosine = first[0] * second[0] + first[1] * second[1]
    const denominator = 1 + cosine
    let offsetX: number
    let offsetY: number
    if (denominator < 1e-3) {
      offsetX = first[0] * distance
      offsetY = first[1] * distance
    } else {
      const scale = distance / denominator
      offsetX = (first[0] + second[0]) * scale
      offsetY = (first[1] + second[1]) * scale
      const length = Math.hypot(offsetX, offsetY)
      if (length > limit) {
        offsetX = (offsetX / length) * limit
        offsetY = (offsetY / length) * limit
      }
    }
    result.push([current[0] + offsetX, current[1] + offsetY])
  }
  return result
}

/**
 * Chain unordered boundary segments into closed loops by matching exact
 * endpoint coordinates. Open chains are closed as-is so the caller can still
 * render them.
 */
export function chainSegments(segments: Array<[Point, Point]>): Point[][] {
  const incident = new Map<string, number[]>()
  segments.forEach((segment, index) => {
    for (const point of segment) {
      const key = pointKey(point)
      const list = incident.get(key) ?? []
      list.push(index)
      incident.set(key, list)
    }
  })

  const used = segments.map(() => false)
  const loops: Point[][] = []
  for (let start = 0; start < segments.length; start += 1) {
    if (used[start]) {
      continue
    }
    used[start] = true
    const loop: Point[] = [segments[start][0], segments[start][1]]
    let cursor = segments[start][1]
    while (!pointsEqual(cursor, loop[0])) {
      const candidates = incident.get(pointKey(cursor)) ?? []
      const nextIndex = candidates.find((index) => !used[index])
      if (nextIndex === undefined) {
        break
      }
      used[nextIndex] = true
      const [from, to] = segments[nextIndex]
      cursor = pointsEqual(from, cursor) ? to : from
      loop.push(cursor)
    }
    if (loop.length >= 3) {
      if (pointsEqual(loop[0], loop[loop.length - 1])) {
        loop.pop()
      }
      loops.push(loop)
    }
  }
  return loops
}

/**
 * Compute the boundary rings of every region: segments shared with another
 * region's territory or lying on the outer map border. Segments between two
 * territories of the same region are internal and excluded.
 */
export function computeRegionOutlines(
  territories: Territory[],
  regions: Region[],
): RegionOutlineResult {
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  const regionByTerritory = new Map<string, string>()
  for (const region of regions) {
    for (const territoryId of region.territories) {
      regionByTerritory.set(territoryId, region.id)
    }
  }

  interface EdgeRecord {
    from: Point
    to: Point
    first: string
    second: string | null
  }
  const edges = new Map<string, EdgeRecord>()
  for (const territory of territories) {
    for (const [x, y] of territory.points) {
      minX = Math.min(minX, x)
      minY = Math.min(minY, y)
      maxX = Math.max(maxX, x)
      maxY = Math.max(maxY, y)
    }
    for (let index = 0; index < territory.points.length; index += 1) {
      const from = territory.points[index]
      const to = territory.points[(index + 1) % territory.points.length]
      const key =
        pointKey(from) < pointKey(to)
          ? `${pointKey(from)}|${pointKey(to)}`
          : `${pointKey(to)}|${pointKey(from)}`
      const edge = edges.get(key)
      if (edge) {
        edge.second = territory.id
      } else {
        edges.set(key, { from, to, first: territory.id, second: null })
      }
    }
  }

  const segmentsByRegion = new Map<string, Array<[Point, Point]>>()
  const addSegment = (regionId: string, segment: [Point, Point]) => {
    const list = segmentsByRegion.get(regionId) ?? []
    list.push(segment)
    segmentsByRegion.set(regionId, list)
  }

  const outerSegmentsByRegion = new Map<string, Array<[Point, Point]>>()
  const addOuterSegment = (regionId: string, segment: [Point, Point]) => {
    const list = outerSegmentsByRegion.get(regionId) ?? []
    list.push(segment)
    outerSegmentsByRegion.set(regionId, list)
  }

  const outlines = new Map<string, RegionOutline>()
  const outlineFor = (regionId: string): RegionOutline => {
    const existing = outlines.get(regionId)
    if (existing) {
      return existing
    }
    const created: RegionOutline = { loops: [], outerSegments: [], outerLoops: [] }
    outlines.set(regionId, created)
    return created
  }

  for (const edge of edges.values()) {
    const firstRegion = regionByTerritory.get(edge.first)
    const secondRegion = edge.second ? regionByTerritory.get(edge.second) : undefined
    const segment: [Point, Point] = [edge.from, edge.to]
    if (!secondRegion) {
      if (firstRegion) {
        addSegment(firstRegion, segment)
        addOuterSegment(firstRegion, segment)
        outlineFor(firstRegion).outerSegments.push(segment)
      }
      continue
    }
    if (firstRegion && secondRegion && firstRegion !== secondRegion) {
      addSegment(firstRegion, segment)
      addSegment(secondRegion, segment)
    }
  }

  for (const [regionId, segments] of segmentsByRegion) {
    outlineFor(regionId).loops = chainSegments(segments)
  }
  for (const [regionId, segments] of outerSegmentsByRegion) {
    outlineFor(regionId).outerLoops = chainSegments(segments)
  }

  return {
    outlines,
    bounds: { minX, minY, maxX, maxY },
  }
}

function loopCentroid(points: Point[]): Point {
  const total = points.reduce(
    (sum, [x, y]) => ({ x: sum.x + x, y: sum.y + y }),
    { x: 0, y: 0 },
  )
  return [total.x / points.length, total.y / points.length]
}

/**
 * Whether a boundary loop encloses another region (a hole in this region,
 * such as an enclave): the probe at the loop's vertex centroid then falls in
 * a foreign territory.
 */
export function loopIsHole(
  loop: Point[],
  territoryAt: (point: Point) => string | undefined,
  regionOfTerritory: (territoryId: string) => string | undefined,
  regionId: string,
): boolean {
  const owner = territoryAt(loopCentroid(loop))
  if (!owner) {
    return false
  }
  return regionOfTerritory(owner) !== regionId
}

/**
 * Replaces `regionAnchor`/`partitionSide` usage: the name band now follows the
 * region's outer border geometry instead of a rectangular frame, so the label
 * rides on the longest outer polyline.
 */

const CHAR_WIDTH_RATIO = 0.62

/**
 * Largest font size (capped at `base`, floored at `minimum`) whose estimated
 * bold width fits `maxWidth`.
 */
export function fitLabelFontSize(
  text: string,
  maxWidth: number,
  base = 11,
  minimum = 6.5,
): number {
  if (text.length === 0) {
    return base
  }
  const fitted = maxWidth / (CHAR_WIDTH_RATIO * text.length)
  return Math.max(minimum, Math.min(base, fitted))
}

/** Total length of a polyline in map units. */
export function polylineLength(points: Point[]): number {
  let total = 0
  for (let index = 1; index < points.length; index += 1) {
    total += Math.hypot(points[index][0] - points[index - 1][0], points[index][1] - points[index - 1][1])
  }
  return total
}

/**
 * Flip a polyline so text running along it reads left-to-right (horizontal
 * stretches) or top-to-bottom (vertical stretches).
 */
export function normalizePolylineDirection(points: Point[]): Point[] {
  if (points.length < 2) {
    return [...points]
  }
  const dx = points[points.length - 1][0] - points[0][0]
  const dy = points[points.length - 1][1] - points[0][1]
  const reversed = Math.abs(dx) >= Math.abs(dy) ? dx < 0 : dy < 0
  return reversed ? [...points].reverse() : [...points]
}

/**
 * Longest sub-polyline whose segments never turn back against their shared
 * heading (dot product of consecutive directions stays above
 * `minAlignment`): text riding this run never renders upside down.
 */
export function longestReadableRun(points: Point[], minAlignment = 0.35): Point[] {
  if (points.length <= 2) {
    return [...points]
  }
  let bestStart = 0
  let bestEnd = 1
  let bestLength = 0
  for (let start = 0; start < points.length - 1; start += 1) {
    const refDx = points[start + 1][0] - points[start][0]
    const refDy = points[start + 1][1] - points[start][1]
    const refLength = Math.hypot(refDx, refDy) || 1
    const refX = refDx / refLength
    const refY = refDy / refLength
    let length = refLength
    if (length > bestLength) {
      bestLength = length
      bestStart = start
      bestEnd = start + 1
    }
    for (let end = start + 2; end < points.length; end += 1) {
      const dx = points[end][0] - points[end - 1][0]
      const dy = points[end][1] - points[end - 1][1]
      const segmentLength = Math.hypot(dx, dy) || 1
      if ((dx / segmentLength) * refX + (dy / segmentLength) * refY < minAlignment) {
        break
      }
      length += segmentLength
      if (length > bestLength) {
        bestLength = length
        bestStart = start
        bestEnd = end
      }
    }
  }
  return points.slice(bestStart, bestEnd + 1)
}

/**
 * Which side of the walked polyline faces away from the region: probe just
 * off the middle segment; unowned ground (outside the map) means the +1 side
 * is outward.
 */
export function polylineOutwardDirection(
  points: Point[],
  territoryAt: (point: Point) => string | undefined,
  regionOfTerritory: (territoryId: string) => string | undefined,
  regionId: string,
): 1 | -1 {
  if (points.length < 2) {
    return 1
  }
  const midIndex = Math.floor((points.length - 1) / 2)
  const from = points[midIndex]
  const to = points[midIndex + 1]
  const dx = to[0] - from[0]
  const dy = to[1] - from[1]
  const length = Math.hypot(dx, dy) || 1
  const probe: Point = [
    (from[0] + to[0]) / 2 + (-dy / length) * 2,
    (from[1] + to[1]) / 2 + (dx / length) * 2,
  ]
  const owner = territoryAt(probe)
  if (owner === undefined) {
    return 1
  }
  return regionOfTerritory(owner) === regionId ? -1 : 1
}

/**
 * Offset an open polyline sideways by `distance` map units (+1 offsets to the
 * left of the walk direction, -1 to the right). Interior vertices join with a
 * clamped miter.
 */
export function offsetPolyline(
  points: Point[],
  distance: number,
  direction: 1 | -1,
): Point[] {
  if (points.length < 2 || distance === 0) {
    return [...points]
  }
  const limit = Math.abs(distance) * 3
  const segmentNormal = (index: number): Point => {
    const dx = points[index + 1][0] - points[index][0]
    const dy = points[index + 1][1] - points[index][1]
    const length = Math.hypot(dx, dy) || 1
    return [(direction * -dy) / length, (direction * dx) / length]
  }
  const result: Point[] = []
  for (let index = 0; index < points.length; index += 1) {
    const current = points[index]
    if (index === 0) {
      const normal = segmentNormal(0)
      result.push([current[0] + normal[0] * distance, current[1] + normal[1] * distance])
      continue
    }
    if (index === points.length - 1) {
      const normal = segmentNormal(index - 1)
      result.push([current[0] + normal[0] * distance, current[1] + normal[1] * distance])
      continue
    }
    const first = segmentNormal(index - 1)
    const second = segmentNormal(index)
    const cosine = first[0] * second[0] + first[1] * second[1]
    const denominator = 1 + cosine
    let offsetX: number
    let offsetY: number
    if (denominator < 1e-3) {
      offsetX = first[0] * distance
      offsetY = first[1] * distance
    } else {
      const scale = distance / denominator
      offsetX = (first[0] + second[0]) * scale
      offsetY = (first[1] + second[1]) * scale
      const offsetLength = Math.hypot(offsetX, offsetY)
      if (offsetLength > limit) {
        offsetX = (offsetX / offsetLength) * limit
        offsetY = (offsetY / offsetLength) * limit
      }
    }
    result.push([current[0] + offsetX, current[1] + offsetY])
  }
  return result
}
