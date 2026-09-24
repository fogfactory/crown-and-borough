import { pointsToPath } from '@/lib/map-svg-geometry'
import { regionStyle, type RegionStyle } from '@/lib/region-color'
import {
  loopIsHole,
  polygonContains,
  polylineLength,
  type RegionOutlineResult,
} from '@/lib/region-geometry'
import type { Point, Region, Territory } from '@/types'

/** Padding around the map forming the regional name frame. */
export const REGION_BAND_MARGIN = 26

/** One region's slice of the name frame, with its label anchor. */
export interface RegionBand {
  regionId: string
  piecePath: string
  labelX: number
  labelY: number
  labelAngle: number
  labelWidth: number
}

export function regionIdByTerritory(regions: Region[] | undefined): Map<string, string> {
  const result = new Map<string, string>()
  for (const region of regions ?? []) {
    for (const territoryID of region.territories) result.set(territoryID, region.id)
  }
  return result
}

/** Heraldic style per region, assigned in sorted region-id order. */
export function regionStylesById(
  regionByTerritory: Map<string, string>,
): Map<string, RegionStyle> {
  return new Map<string, RegionStyle>(
    [...new Set(regionByTerritory.values())]
      .sort()
      .map((regionID, index, regionIDs) => [
        regionID,
        regionStyle(index, regionIDs.length),
      ]),
  )
}

/** Regions that never touch the map edge: they get a badge instead of a band. */
export function interiorRegionIds(outlines: RegionOutlineResult | null): Set<string> {
  const interior = new Set<string>()
  if (!outlines) {
    return interior
  }
  for (const [regionId, outline] of outlines.outlines) {
    if (outline.outerSegments.length === 0) {
      interior.add(regionId)
    }
  }
  return interior
}

export function territoryLocator(
  territories: Territory[],
): (point: Point) => string | undefined {
  return (point: Point): string | undefined => {
    for (const territory of territories) {
      if (polygonContains(territory.points, point)) {
        return territory.id
      }
    }
    return undefined
  }
}

/**
 * Slices the frame around a `mapWidth × mapHeight` map into one band per
 * outer stretch of each region, sweeping radially from the map center.
 */
export function computeRegionBands(
  outlines: RegionOutlineResult | null,
  mapWidth: number,
  mapHeight: number,
): RegionBand[] {
  if (!outlines) {
    return []
  }
  const center: Point = [mapWidth / 2, mapHeight / 2]
  const far = 2 * Math.hypot(mapWidth, mapHeight)
  const radialOut = (point: Point): Point => {
    const dx = point[0] - center[0]
    const dy = point[1] - center[1]
    const length = Math.hypot(dx, dy) || 1
    return [point[0] + (dx / length) * far, point[1] + (dy / length) * far]
  }
  // Distance from the center to the frame rectangle along a ray, so label
  // anchors always land inside the frame padding, never under the map.
  const rayRectDistance = (unitX: number, unitY: number): number => {
    let distance = far
    if (Math.abs(unitX) > 1e-9) {
      const edgeX = unitX > 0 ? mapWidth + REGION_BAND_MARGIN : -REGION_BAND_MARGIN
      distance = Math.min(distance, (edgeX - center[0]) / unitX)
    }
    if (Math.abs(unitY) > 1e-9) {
      const edgeY = unitY > 0 ? mapHeight + REGION_BAND_MARGIN : -REGION_BAND_MARGIN
      distance = Math.min(distance, (edgeY - center[1]) / unitY)
    }
    return distance
  }
  const bands: RegionBand[] = []
  for (const [regionId, outline] of outlines.outlines) {
    for (const stretch of outline.outerLoops) {
      const piece: Point[] = [...stretch]
      piece.push(radialOut(stretch[stretch.length - 1]))
      // Unwrap the stretch's polar angles so the far arc sweeps back over
      // the same sector without flipping through the map.
      const angles: number[] = []
      let previous = Math.atan2(stretch[0][1] - center[1], stretch[0][0] - center[0])
      angles.push(previous)
      for (let index = 1; index < stretch.length; index += 1) {
        const angle = Math.atan2(
          stretch[index][1] - center[1],
          stretch[index][0] - center[0],
        )
        let delta = angle - previous
        while (delta > Math.PI) {
          delta -= 2 * Math.PI
        }
        while (delta < -Math.PI) {
          delta += 2 * Math.PI
        }
        previous += delta
        angles.push(previous)
      }
      const sweep = previous - angles[0]
      const steps = Math.max(8, Math.ceil(Math.abs(sweep) / (Math.PI / 12)))
      for (let step = 1; step <= steps; step += 1) {
        const angle = previous - (sweep * step) / steps
        piece.push([center[0] + Math.cos(angle) * far, center[1] + Math.sin(angle) * far])
      }
      // Label anchor: the longest straight run of the piece's outer edge
      // (inset from the frame rectangle by half the padding), so corner-
      // spanning sections center their name on a readable straight edge.
      const arcSteps = Math.max(8, Math.ceil(Math.abs(sweep) / (Math.PI / 12)))
      const labelInset = REGION_BAND_MARGIN * 0.5
      const arcPoints: Point[] = []
      for (let step = 0; step <= arcSteps; step += 1) {
        const angle = angles[0] + (sweep * step) / arcSteps
        const distance = rayRectDistance(Math.cos(angle), Math.sin(angle)) - labelInset
        arcPoints.push([
          center[0] + Math.cos(angle) * distance,
          center[1] + Math.sin(angle) * distance,
        ])
      }
      const runs: Array<{ points: Point[]; vertical: boolean }> = []
      for (let index = 1; index < arcPoints.length; index += 1) {
        const from = arcPoints[index - 1]
        const to = arcPoints[index]
        const vertical = Math.abs(to[0] - from[0]) < Math.abs(to[1] - from[1])
        const last = runs[runs.length - 1]
        if (last && last.vertical === vertical) {
          last.points.push(to)
        } else {
          runs.push({ points: [from, to], vertical })
        }
      }
      let bestRun = runs[0]
      let bestLength = 0
      for (const run of runs) {
        const length = polylineLength(run.points)
        if (length > bestLength) {
          bestLength = length
          bestRun = run
        }
      }
      const runPoints = bestRun.points
      const labelWidth = bestLength
      const from = runPoints[0]
      const to = runPoints[runPoints.length - 1]
      const labelX = (from[0] + to[0]) / 2
      const labelY = (from[1] + to[1]) / 2
      const labelAngle = bestRun.vertical ? (to[1] > from[1] ? 90 : -90) : 0
      bands.push({
        regionId,
        piecePath: pointsToPath(piece),
        labelX,
        labelY,
        labelAngle,
        labelWidth,
      })
    }
  }
  return bands
}

/** Per-region predicate telling which outline loops are holes of the region. */
export function regionHoleTesters(
  outlines: RegionOutlineResult | null,
  territoryAt: (point: Point) => string | undefined,
  regionByTerritory: Map<string, string>,
): Map<string, (loop: Point[]) => boolean> {
  const testers = new Map<string, (loop: Point[]) => boolean>()
  if (!outlines) {
    return testers
  }
  for (const regionId of outlines.outlines.keys()) {
    testers.set(regionId, (loop) =>
      loopIsHole(
        loop,
        territoryAt,
        (territoryId) => regionByTerritory.get(territoryId),
        regionId,
      ),
    )
  }
  return testers
}
