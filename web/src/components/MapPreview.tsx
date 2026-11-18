import { useMemo } from 'react'

import { HERALDIC_COLORS, MAP_INK, MAP_PAPER_FILL } from '@/lib/map-palette'
import { boundsOf, pointsToPath } from '@/lib/map-svg-geometry'
import { computeRegionOutlines } from '@/lib/region-geometry'
import type { MapData } from '@/types'

/** Margin around the drawing, in map units. */
const PREVIEW_MARGIN = 12

/**
 * A read-only drawing of a map: territories, tinted by region, with the
 * regions' borders (their boundary rings) drawn heavier than the territories'. It carries no
 * marker, label or placement: a candidate map shows nothing but its shape.
 */
export function MapPreview({ map, label }: { map: MapData; label: string }) {
  const drawing = useMemo(() => {
    const { outlines } = computeRegionOutlines(map.territories, map.regions ?? [])
    const regions = (map.regions ?? []).map((region, index) => ({
      id: region.id,
      color: HERALDIC_COLORS[index % HERALDIC_COLORS.length],
      territories: region.territories,
      // Every boundary ring of the region, as one path.
      path: (outlines.get(region.id)?.loops ?? []).map(pointsToPath).join(' '),
    }))
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
        <path
          key={region.id}
          data-preview-region={region.id}
          d={region.path}
          fill="none"
          stroke={MAP_INK}
          strokeWidth={2.6}
          strokeLinejoin="round"
        />
      ))}
    </svg>
  )
}
