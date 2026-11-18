import { useMemo } from 'react'

import { HERALDIC_COLORS, MAP_INK, MAP_PAPER_FILL } from '@/lib/map-palette'
import { boundsOf, centroid, pointsToPath } from '@/lib/map-svg-geometry'
import { MARKER_GLYPHS } from '@/components/MapMarkers'
import { computeRegionOutlines } from '@/lib/region-geometry'
import type { MapData } from '@/types'

/** Margin around the drawing, in map units. */
const PREVIEW_MARGIN = 12

/**
 * A read-only drawing of a map: territories, tinted by region, with the
 * regions' borders (their boundary rings) drawn heavier than the territories'.
 * It marks what a player weighs when voting: the start castles (a set, no
 * house attached), the villages and the region seats (a cross).
 */
const CASTLE_SIZE = 22
const VILLAGE_SIZE = 14
const GLYPH_BOX = 512
// Latin cross, in a 24x24 box: the seat of a bishopric.
const CROSS_PATH = 'M10 2h4v6h6v4h-6v10h-4V12H4V8h6z'

function Glyph({ kind, center, size, id }: { kind: 'castle' | 'village'; center: [number, number]; size: number; id: string }) {
  const scale = size / GLYPH_BOX
  return (
    <path
      {...{ [`data-preview-${kind}`]: id }}
      d={MARKER_GLYPHS[kind]}
      fill={MAP_INK}
      stroke={MAP_PAPER_FILL}
      strokeWidth={14}
      paintOrder="stroke"
      transform={`translate(${center[0] - size / 2} ${center[1] - size / 2}) scale(${scale})`}
    />
  )
}

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
    const centers = new Map(map.territories.map((territory) => [territory.id, centroid(territory.points)] as const))
    const castles = (map.starts ?? []).filter((id) => centers.has(id))
    const villages = map.territories.filter((territory) => territory.village && !castles.includes(territory.id)).map((territory) => territory.id)
    const seats = (map.regions ?? []).map((region) => region.seed).filter((id) => centers.has(id))
    return { regions, regionOf, bounds, centers, castles, villages, seats }
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
      {drawing.villages.map((id) => (
        <Glyph key={`v-${id}`} kind="village" center={drawing.centers.get(id)!} size={VILLAGE_SIZE} id={id} />
      ))}
      {drawing.castles.map((id) => (
        <Glyph key={`c-${id}`} kind="castle" center={drawing.centers.get(id)!} size={CASTLE_SIZE} id={id} />
      ))}
      {drawing.seats.map((id) => {
        const [x, y] = drawing.centers.get(id)!
        return (
          <path
            key={`s-${id}`}
            data-preview-seat={id}
            d={CROSS_PATH}
            fill={MAP_PAPER_FILL}
            stroke={MAP_INK}
            strokeWidth={1.2}
            strokeLinejoin="round"
            transform={`translate(${x - 5} ${y - 22}) scale(0.8)`}
          />
        )
      })}
    </svg>
  )
}
