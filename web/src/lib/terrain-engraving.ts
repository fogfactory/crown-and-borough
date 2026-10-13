import { hashSeed, mulberry32 } from '@/lib/chaotic-icons'
import { centroid } from '@/lib/map-svg-geometry'
import type { Point, Terrain, Territory } from '@/types'

/** Spacing between two terrain symbols, in map units (graphic charter). */
export const TERRAIN_SYMBOL_STEP: Record<Terrain, number> = {
  forest: 12.5,
  hill: 17,
  mountain: 16,
  swamp: 15,
  plain: 23,
}

/** Symbols stay this far from the territory's borders. */
const BORDER_CLEARANCE = 6.5
/** Semi-axes of the ellipse around the centroid kept for labels and pieces. */
const RESERVED_RADIUS_X = 30
const RESERVED_RADIUS_Y = 27
/** Share of the step a symbol may drift from its grid node. */
const JITTER = 0.7

export interface TerrainSymbolPlacement {
  x: number
  y: number
  /** Size multiplier around 1, so the scatter never reads as a grid. */
  scale: number
}

function insidePolygon([x, y]: Point, points: Point[]): boolean {
  let inside = false
  for (
    let index = 0, previous = points.length - 1;
    index < points.length;
    previous = index++
  ) {
    const [xi, yi] = points[index]
    const [xj, yj] = points[previous]
    if (yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) {
      inside = !inside
    }
  }
  return inside
}

function distanceToSegment([x, y]: Point, [ax, ay]: Point, [bx, by]: Point): number {
  const dx = bx - ax
  const dy = by - ay
  const lengthSquared = dx * dx + dy * dy
  const t =
    lengthSquared === 0
      ? 0
      : Math.max(0, Math.min(1, ((x - ax) * dx + (y - ay) * dy) / lengthSquared))
  return Math.hypot(x - (ax + t * dx), y - (ay + t * dy))
}

function distanceToBoundary(point: Point, points: Point[]): number {
  let distance = Infinity
  for (let index = 0; index < points.length; index += 1) {
    const next = points[(index + 1) % points.length]
    distance = Math.min(distance, distanceToSegment(point, points[index], next))
  }
  return distance
}

/**
 * Scatters the engraved terrain symbols of one territory: a jittered grid at
 * the terrain's step, seeded by the territory code so the layout never moves
 * between renders, clear of the borders and of the label ellipse.
 */
export function terrainSymbolPlacements(
  territory: Territory,
  annotationScale: number,
): TerrainSymbolPlacement[] {
  const { points } = territory
  if (points.length < 3) {
    return []
  }

  const step = TERRAIN_SYMBOL_STEP[territory.terrain] * annotationScale
  const clearance = BORDER_CLEARANCE * annotationScale
  const radiusX = RESERVED_RADIUS_X * annotationScale
  const radiusY = RESERVED_RADIUS_Y * annotationScale
  const [centerX, centerY] = centroid(points)
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const [x, y] of points) {
    minX = Math.min(minX, x)
    minY = Math.min(minY, y)
    maxX = Math.max(maxX, x)
    maxY = Math.max(maxY, y)
  }

  const random = mulberry32(hashSeed(`terrain-${territory.id}`))
  const placements: TerrainSymbolPlacement[] = []
  // Stagger every other row by half a step, like an engraver's hand.
  for (let row = 0, y = minY + step / 2; y < maxY; row += 1, y += step * 0.87) {
    const offset = row % 2 === 0 ? 0 : step / 2
    for (let x = minX + step / 2 + offset; x < maxX; x += step) {
      const point: Point = [
        x + (random() - 0.5) * step * JITTER,
        y + (random() - 0.5) * step * JITTER,
      ]
      const scale = 0.85 + random() * 0.3
      const ellipse =
        ((point[0] - centerX) / radiusX) ** 2 + ((point[1] - centerY) / radiusY) ** 2
      if (
        ellipse < 1 ||
        !insidePolygon(point, points) ||
        distanceToBoundary(point, points) < clearance
      ) {
        continue
      }
      placements.push({ x: point[0], y: point[1], scale })
    }
  }
  // Paint from top to bottom so lower symbols overlap the ones behind them.
  return placements.sort((first, second) => first.y - second.y)
}
