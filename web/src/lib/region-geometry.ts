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
 * Chain unordered edges carrying a payload into ordered sequences joined by
 * exact endpoint matching (the global map border walk). Closed borders come
 * back to their starting point; open chains stop when no continuation is
 * unused. Each returned edge is re-oriented so its `from` continues the
 * previous edge's `to`.
 */
function chainEdgeRuns<T extends { from: Point; to: Point }>(
  edges: T[],
): T[][] {
  const incident = new Map<string, number[]>()
  edges.forEach((edge, index) => {
    for (const point of [edge.from, edge.to]) {
      const key = pointKey(point)
      const list = incident.get(key) ?? []
      list.push(index)
      incident.set(key, list)
    }
  })

  const used = edges.map(() => false)
  const sequences: T[][] = []
  for (let start = 0; start < edges.length; start += 1) {
    if (used[start]) {
      continue
    }
    used[start] = true
    const sequence = [edges[start]]
    let cursor = edges[start].to
    while (!pointsEqual(cursor, edges[start].from)) {
      const candidates = incident.get(pointKey(cursor)) ?? []
      const nextIndex = candidates.find((index) => !used[index])
      if (nextIndex === undefined) {
        break
      }
      used[nextIndex] = true
      const edge = edges[nextIndex]
      cursor = pointsEqual(edge.from, cursor) ? edge.to : edge.from
      sequence.push(pointsEqual(edge.from, sequence[sequence.length - 1].to) ? edge : { ...edge, from: edge.to, to: edge.from })
    }
    if (sequence.length >= 2) {
      sequences.push(sequence)
    }
  }
  return sequences
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

  const outerEdges: Array<{ from: Point; to: Point; regionId: string }> = []
  for (const edge of edges.values()) {
    const firstRegion = regionByTerritory.get(edge.first)
    const secondRegion = edge.second ? regionByTerritory.get(edge.second) : undefined
    const segment: [Point, Point] = [edge.from, edge.to]
    if (!secondRegion) {
      if (firstRegion) {
        addSegment(firstRegion, segment)
        outlineFor(firstRegion).outerSegments.push(segment)
        outerEdges.push({ from: edge.from, to: edge.to, regionId: firstRegion })
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

  // Chain the outer map border globally (every boundary vertex meets exactly
  // two outer edges), then split it into the maximal contiguous stretches of
  // each region, so a region hugging the border on several sides still gets
  // one continuous strip per contact run.
for (const edgeLoop of chainEdgeRuns(outerEdges)) {
    const closed = pointsEqual(
      edgeLoop[edgeLoop.length - 1].to,
      edgeLoop[0].from,
    )
    let pivot = 0
    if (closed) {
      for (let index = 0; index < edgeLoop.length; index += 1) {
        const previous = edgeLoop[(index - 1 + edgeLoop.length) % edgeLoop.length]
        if (previous.regionId !== edgeLoop[index].regionId) {
          pivot = index
          break
        }
      }
    }
    const rotated = [...edgeLoop.slice(pivot), ...edgeLoop.slice(0, pivot)]
    let runStart = 0
    for (let index = 1; index <= rotated.length; index += 1) {
      const boundary =
        index === rotated.length ||
        rotated[index].regionId !== rotated[runStart].regionId
      if (!boundary) {
        continue
      }
      const runEdges = rotated.slice(runStart, index)
      const polyline: Point[] = [runEdges[0].from, ...runEdges.map((edge) => edge.to)]
      if (pointsEqual(polyline[0], polyline[polyline.length - 1])) {
        polyline.pop()
      }
      if (polyline.length >= 2) {
        outlineFor(rotated[runStart].regionId).outerLoops.push(polyline)
      }
      runStart = index
    }
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

