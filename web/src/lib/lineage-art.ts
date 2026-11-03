/**
 * Drawing helpers of the lineage tapestry. The layout (`lineage.ts`) hands out
 * right-angled stems; these turn them into the curved vines drawn on the cloth.
 */

export interface Point {
  x: number
  y: number
}

/**
 * Points of an axis-aligned path made of `M`, `L`, `H` and `V` commands (the
 * only ones `placeTree` writes). Anything else yields an empty list.
 */
export function pathPoints(d: string): Point[] {
  const points: Point[] = []
  let x = 0
  let y = 0
  for (const [, command, args] of d.matchAll(/([MLHV])\s*([-\d.,\s]+)/g)) {
    const numbers = args
      .split(/[\s,]+/)
      .filter(Boolean)
      .map(Number)
    if (numbers.some(Number.isNaN)) return []
    if (command === 'H') x = numbers[0]
    else if (command === 'V') y = numbers[0]
    else {
      x = numbers[0]
      y = numbers[1]
    }
    points.push({ x, y })
  }
  return points
}

/**
 * The two ends of the rounded corner at `here`, as wide as the neighbouring
 * segments allow (at most `radius`), or null when the corner is too tight to round.
 */
function corner(
  prev: Point,
  here: Point,
  next: Point,
  radius: number,
): { before: Point; after: Point } | null {
  const back = Math.hypot(here.x - prev.x, here.y - prev.y)
  const forth = Math.hypot(next.x - here.x, next.y - here.y)
  const r = Math.min(radius, back / 2, forth / 2)
  if (r < 0.5) return null
  return {
    before: {
      x: here.x + ((prev.x - here.x) / back) * r,
      y: here.y + ((prev.y - here.y) / back) * r,
    },
    after: {
      x: here.x + ((next.x - here.x) / forth) * r,
      y: here.y + ((next.y - here.y) / forth) * r,
    },
  }
}

/**
 * Smooth path through the points: every corner is rounded by a quadratic
 * curve, as wide as the neighbouring segments allow (at most `radius`).
 */
export function roundedPath(points: Point[], radius = 18): string {
  if (points.length === 0) return ''
  let d = `M${points[0].x},${points[0].y}`
  for (let i = 1; i < points.length; i++) {
    const here = points[i]
    const next = points[i + 1]
    if (!next) {
      d += ` L${here.x},${here.y}`
      break
    }
    const round = corner(points[i - 1], here, next, radius)
    if (!round) {
      d += ` L${here.x},${here.y}`
      continue
    }
    const { before, after } = round
    d += ` L${before.x},${before.y} Q${here.x},${here.y} ${after.x},${after.y}`
  }
  return d
}

/** Direction (radians) of the last segment of a polyline. */
export function endAngle(points: Point[]): number {
  const last = points[points.length - 1]
  const before = points[points.length - 2]
  if (!last || !before) return 0
  return Math.atan2(last.y - before.y, last.x - before.x)
}

/** A leaf of the ivy: where it is rooted, the way the stem runs, the side it hangs on. */
export interface IvyLeaf extends Point {
  /** Direction (radians) of the bond at the leaf. */
  angle: number
  /** `1` on one side of the bond, `-1` on the other. */
  side: 1 | -1
  /** Deterministic tilt (degrees) added to the leaf's usual slant. */
  tilt: number
}

/**
 * The polyline `roundedPath` draws, with every quadratic corner sampled into
 * `steps` short segments (same radius rule), so measures made on it follow the
 * curve actually drawn.
 */
export function roundedPolyline(points: Point[], radius = 18, steps = 6): Point[] {
  if (points.length < 3) return points.slice()
  const out: Point[] = [points[0]]
  for (let i = 1; i < points.length - 1; i++) {
    const here = points[i]
    const round = corner(points[i - 1], here, points[i + 1], radius)
    if (!round) {
      out.push(here)
      continue
    }
    const { before, after } = round
    for (let k = 0; k <= steps; k++) {
      const t = k / steps
      const u = 1 - t
      out.push({
        x: u * u * before.x + 2 * u * t * here.x + t * t * after.x,
        y: u * u * before.y + 2 * u * t * here.y + t * t * after.y,
      })
    }
  }
  out.push(points[points.length - 1])
  return out
}

/**
 * Leaves of the ivy along the path as drawn (corners rounded by `radius`, like
 * `roundedPath`), on both sides of the stem. They are spread evenly over the
 * length (about `spacing` apart, at least two stations), each station bearing
 * one leaf per side, so the count follows the bond and the placement is
 * deterministic. A small tilt varies from station to station (a fixed cycle,
 * no randomness). A leaf never sits within `clearAround` of the middle (the
 * knot); on a short bond the clearance shrinks with it so some leaves remain.
 */
export function ivyLeaves(
  points: Point[],
  radius = 10,
  spacing = 8,
  clearAround = 11,
): IvyLeaf[] {
  const path = roundedPolyline(points, radius)
  if (path.length < 2) return []
  const lengths: number[] = []
  let total = 0
  for (let i = 1; i < path.length; i++) {
    const length = Math.hypot(path[i].x - path[i - 1].x, path[i].y - path[i - 1].y)
    lengths.push(length)
    total += length
  }
  if (total < 6) return []
  const count = Math.max(2, Math.floor(total / spacing))
  const clear = Math.min(clearAround, total * 0.24)
  const tilts = [0, 9, -6, 12, -10, 4]
  const leaves: IvyLeaf[] = []
  for (let n = 0; n < count; n++) {
    const at = ((n + 0.5) / count) * total
    if (Math.abs(at - total / 2) < clear) continue
    let segment = 0
    let start = 0
    while (segment < lengths.length - 1 && start + lengths[segment] < at) {
      start += lengths[segment]
      segment++
    }
    const from = path[segment]
    const to = path[segment + 1]
    const ratio = lengths[segment] === 0 ? 0 : (at - start) / lengths[segment]
    const base = {
      x: from.x + (to.x - from.x) * ratio,
      y: from.y + (to.y - from.y) * ratio,
      angle: Math.atan2(to.y - from.y, to.x - from.x),
    }
    leaves.push({ ...base, side: 1, tilt: tilts[n % tilts.length] })
    leaves.push({ ...base, side: -1, tilt: tilts[(n + 3) % tilts.length] })
  }
  return leaves
}
