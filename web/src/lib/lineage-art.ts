/**
 * Drawing helpers of the lineage tapestry. The layout (`lineage.ts`) hands out
 * right-angled stems; these turn them into the curved vines drawn on the cloth.
 */

/** A roundel: its centre, its radius and half the width of its box. */
export interface Roundel {
  x: number
  y: number
  r: number
  half: number
}

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
 * Pulls the ends of a line that stops at the edge of a noble's box in to the
 * rim of its roundel, so a stem or a vine visibly reaches the figure.
 */
export function reachInto(points: Point[], roundels: Roundel[]): Point[] {
  const moved = points.map((point) => ({ ...point }))
  const reach = (index: number) => {
    const point = moved[index]
    for (const roundel of roundels) {
      const dx = roundel.x - point.x
      if (
        Math.abs(point.y - roundel.y) > 0.5 ||
        Math.abs(Math.abs(dx) - roundel.half) > 0.5
      )
        continue
      moved[index] = { x: roundel.x - Math.sign(dx) * roundel.r, y: point.y }
      return
    }
  }
  if (moved.length > 1) {
    reach(0)
    reach(moved.length - 1)
  }
  return moved
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
    const prev = points[i - 1]
    const back = Math.hypot(here.x - prev.x, here.y - prev.y)
    const forth = Math.hypot(next.x - here.x, next.y - here.y)
    const r = Math.min(radius, back / 2, forth / 2)
    if (r < 0.5) {
      d += ` L${here.x},${here.y}`
      continue
    }
    const before = {
      x: here.x + ((prev.x - here.x) / back) * r,
      y: here.y + ((prev.y - here.y) / back) * r,
    }
    const after = {
      x: here.x + ((next.x - here.x) / forth) * r,
      y: here.y + ((next.y - here.y) / forth) * r,
    }
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
