import type { Point } from '@/types'

/** One scattered overlay icon, positioned around a territory centroid. */
export interface IconPlacement {
  x: number
  y: number
  size: number
  rotation: number
}

/** Mulberry32: tiny deterministic PRNG, reproducible across renders. */
function mulberry32(seed: number): () => number {
  let state = seed >>> 0
  return () => {
    state = (state + 0x6d2b79f5) >>> 0
    let t = state
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function hashSeed(key: string): number {
  let hash = 2166136261
  for (let index = 0; index < key.length; index += 1) {
    hash ^= key.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

function polygonRadius(points: Point[], center: Point): number {
  let radius = 0
  for (const [x, y] of points) {
    const distance = Math.hypot(x - center[0], y - center[1])
    if (distance > radius) {
      radius = distance
    }
  }
  return radius
}

/** Half the largest distance from the territory centroid, in map units. */
export function territoryRadius(points: Point[]): number {
  if (points.length === 0) {
    return 0
  }
  const center: Point = [0, 0]
  for (const [x, y] of points) {
    center[0] += x / points.length
    center[1] += y / points.length
  }
  return polygonRadius(points, center)
}

/**
 * Scatters `count` icon placements inside a territory polygon. The layout is
 * chaotic like the winter snow yet fully deterministic for one
 * (territory, seedKey) pair, so icons never jump between renders.
 */
export function chaoticIconPlacements(
  points: Point[],
  count: number,
  seedKey: string,
): IconPlacement[] {
  if (points.length === 0 || count <= 0) {
    return []
  }
  const center: Point = [0, 0]
  for (const [x, y] of points) {
    center[0] += x / points.length
    center[1] += y / points.length
  }
  const radius = polygonRadius(points, center)
  if (radius <= 0) {
    return []
  }
  const random = mulberry32(hashSeed(seedKey))
  return Array.from({ length: count }, () => {
    const angle = random() * 2 * Math.PI
    const distance = (0.18 + random() * 0.3) * radius
    return {
      x: center[0] + Math.cos(angle) * distance,
      y: center[1] + Math.sin(angle) * distance,
      size: (0.16 + random() * 0.08) * radius,
      rotation: (random() - 0.5) * 0.6,
    }
  })
}

/**
 * Chains icons along a border segment, spaced roughly every `spacing` map
 * units with perpendicular jitter and chaotic tilts. Fully deterministic
 * for one (segment, seedKey) pair.
 */
export function borderIconPlacements(
  from: Point,
  to: Point,
  seedKey: string,
  spacing: number,
  baseSize: number,
): IconPlacement[] {
  const length = Math.hypot(to[0] - from[0], to[1] - from[1])
  if (length <= 0 || spacing <= 0) {
    return []
  }
  const count = Math.max(1, Math.round(length / spacing))
  const random = mulberry32(hashSeed(seedKey))
  const unitX = (to[0] - from[0]) / length
  const unitY = (to[1] - from[1]) / length
  return Array.from({ length: count }, (_, index) => {
    const along = ((index + 0.5) / count) * length
    const lateral = (random() - 0.5) * spacing * 0.5
    return {
      x: from[0] + unitX * along - unitY * lateral,
      y: from[1] + unitY * along + unitX * lateral,
      size: baseSize * (0.85 + random() * 0.3),
      rotation: 0,
    }
  })
}
