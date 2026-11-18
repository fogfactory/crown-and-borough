import { useMemo } from 'react'

import { HERALDIC_COLORS, MAP_INK, MAP_PAPER_FILL } from '@/lib/map-palette'
import { boundsOf, edgeKey, pointsToPath, polygonEdges } from '@/lib/map-svg-geometry'
import type { MapData, Point } from '@/types'

/** Margin around the drawing, in map units. */
const PREVIEW_MARGIN = 12

/**
 * Edges of a region's outline: the polygon edges held by one of its
 * territories only, i.e. not shared with another territory of the region.
 */
function regionOutline(
  territoryPoints: Point[][],
): Array<[Point, Point]> {
  const counts = new Map<string, number>()
  for (const points of territoryPoints) {
    for (const [from, to] of polygonEdges(points)) {
      const key = edgeKey(from, to)
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }
  }
  const outline: Array<[Point, Point]> = []
  for (const points of territoryPoints) {
    for (const [from, to] of polygonEdges(points)) {
      if (counts.get(edgeKey(from, to)) === 1) outline.push([from, to])
    }
  }
  return outline
}

/**
 * A read-only drawing of a map: territories, tinted by region, with the
 * regions' frontiers drawn heavier than the territories'. It carries no
 * marker, label or placement: a candidate map shows nothing but its shape.
 */
export function MapPreview({ map, label }: { map: MapData; label: string }) {
  const drawing = useMemo(() => {
    const byId = new Map(map.territories.map((territory) => [territory.id, territory]))
    const regions = (map.regions ?? []).map((region, index) => {
      const points = region.territories.flatMap((id) => {
        const territory = byId.get(id)
        return territory ? [territory.points] : []
      })
      return {
        id: region.id,
        color: HERALDIC_COLORS[index % HERALDIC_COLORS.length],
        territories: region.territories,
        outline: regionOutline(points),
      }
    })
    const regionOf = new Map<string, string>()
    for (const region of regions) {
      for (const id of region.territories) regionOf.set(id, region.color)
    }
    const bounds = boundsOf(map.territories.map((territory) => territory.points))
    return { regions, regionOf, bounds }
  }, [map])

  if (!drawing.bounds) return null
  const [minX, minY, maxX, maxY] = drawing.bounds
  const viewBox = `${minX - PREVIEW_MARGIN} ${minY - PREVIEW_MARGIN} ${maxX - minX + 2 * PREVIEW_MARGIN} ${maxY - minY + 2 * PREVIEW_MARGIN}`

  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={viewBox}
      className="h-auto w-full rounded-md border border-line bg-paper-100"
    >
      {map.territories.map((territory) => (
        <path
          key={territory.id}
          data-preview-territory={territory.id}
          d={pointsToPath(territory.points)}
          fill={drawing.regionOf.get(territory.id) ?? MAP_PAPER_FILL}
          fillOpacity={0.4}
          stroke={MAP_INK}
          strokeOpacity={0.45}
          strokeWidth={0.8}
          strokeLinejoin="round"
        />
      ))}
      {drawing.regions.map((region) => (
        <g key={region.id} data-preview-region={region.id}>
          {region.outline.map(([from, to], index) => (
            <line
              key={index}
              x1={from[0]}
              y1={from[1]}
              x2={to[0]}
              y2={to[1]}
              stroke={MAP_INK}
              strokeWidth={2.6}
              strokeLinecap="round"
            />
          ))}
        </g>
      ))}
    </svg>
  )
}
