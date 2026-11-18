import { useId, useMemo } from 'react'

import { HERALDIC_COLORS, MAP_INK, MAP_PAPER_FILL } from '@/lib/map-palette'
import { boundsOf, centroid, pointsToPath } from '@/lib/map-svg-geometry'
import { TerrainGlyph } from '@/components/MapDecorations'
import { MARKER_GLYPHS } from '@/components/MapMarkers'
import { computeMapBorders } from '@/lib/map-borders'
import { computeRegionOutlines } from '@/lib/region-geometry'
import type { CandidateMapData, Point } from '@/types'

/** Margin around the drawing, in map units. */
const PREVIEW_MARGIN = 12

const CASTLE_SIZE = 22
const VILLAGE_SIZE = 14
const GLYPH_BOX = 512
// Latin cross, in a 24x24 box: the seat of a bishopric.
const CROSS_PATH = 'M10 2h4v6h6v4h-6v10h-4V12H4V8h6z'
const CROSS_BOX = 24
const CROSS_SCALE = 0.8
const WALL_WIDTH = 2.2
// Impassable frontiers are broken lines: a tooth every WALL_TOOTH units, WALL_AMPLITUDE off the frontier.
const WALL_TOOTH = 7
const WALL_AMPLITUDE = 2.4
// A region's border is a coloured band along the inside of its outline: the
// stroke is this wide but clipped to the region, so half of it shows.
const REGION_BORDER_WIDTH = 6
// The engraved symbols are tuned for the full map: enlarged to read at preview size.
const TERRAIN_SCALE = 1.7

// Layout of one territory, around its centre: a marker (castle or village) and
// the seat's cross share the top slot (the cross to the left of a marker), and
// the terrain symbol sits in the middle, or below the top slot when it is used.
const TOP_SLOT_RISE = 9
const CROSS_BESIDE_MARKER = 20
const TERRAIN_BELOW_TOP_SLOT = 12

/** A zigzag from one end of a frontier segment to the other, ends included. */
function zigzagPath([x1, y1]: Point, [x2, y2]: Point): string {
  const dx = x2 - x1
  const dy = y2 - y1
  const length = Math.hypot(dx, dy)
  const teeth = Math.max(2, Math.round(length / WALL_TOOTH))
  const nx = -dy / (length || 1)
  const ny = dx / (length || 1)
  const points = [`M ${x1} ${y1}`]
  for (let step = 1; step < teeth; step += 1) {
    const along = step / teeth
    const side = step % 2 === 1 ? WALL_AMPLITUDE : -WALL_AMPLITUDE
    points.push(`L ${x1 + dx * along + nx * side} ${y1 + dy * along + ny * side}`)
  }
  points.push(`L ${x2} ${y2}`)
  return points.join(' ')
}

type Marker = { id: string; center: Point }

function Glyph({
  kind,
  center,
  size,
  dataAttribute,
}: {
  kind: 'castle' | 'village'
  center: Point
  size: number
  dataAttribute: Record<string, string>
}) {
  const scale = size / GLYPH_BOX
  return (
    <path
      {...dataAttribute}
      d={MARKER_GLYPHS[kind]}
      fill={MAP_INK}
      stroke={MAP_PAPER_FILL}
      strokeWidth={14}
      paintOrder="stroke"
      transform={`translate(${center[0] - size / 2} ${center[1] - size / 2}) scale(${scale})`}
    />
  )
}

/**
 * A read-only drawing of a map: territories tinted by region with a thin
 * coloured border along each region (under the walls), a heavy line on each
 * impassable frontier, and one engraved symbol of the terrain per territory. It marks
 * what a player weighs when voting: the start castles (a set, no house
 * attached), the villages and the region seats (a cross).
 */
export function MapPreview({ map, label }: { map: CandidateMapData; label: string }) {
  const clipPrefix = useId()
  const drawing = useMemo(() => {
    const { outlines } = computeRegionOutlines(map.territories, map.regions ?? [])
    const regions = (map.regions ?? []).map((region, index) => ({
      id: region.id,
      color: HERALDIC_COLORS[index % HERALDIC_COLORS.length],
      territories: region.territories,
      path: (outlines.get(region.id)?.loops ?? []).map(pointsToPath).join(' '),
    }))
    const regionOf = new Map<string, string>()
    for (const region of regions) {
      for (const id of region.territories) regionOf.set(id, region.color)
    }
    const bounds = boundsOf(map.territories.map((territory) => territory.points))
    const territoryById = new Map(map.territories.map((territory) => [territory.id, territory] as const))
    const centers = new Map(map.territories.map((territory) => [territory.id, centroid(territory.points)] as const))
    const castleSet = new Set(map.starts.filter((id) => centers.has(id)))
    const villageSet = new Set(
      map.territories.filter((territory) => territory.village && !castleSet.has(territory.id)).map((territory) => territory.id),
    )
    const seatSet = new Set((map.regions ?? []).map((region) => region.seed).filter((id) => centers.has(id)))
    const walls = computeMapBorders(map).sharedBorders.filter((border) => !border.passable)
    const marked = (id: string) => castleSet.has(id) || villageSet.has(id)
    const topSlot = (id: string): Point => {
      const [x, y] = centers.get(id)!
      return [x, y - TOP_SLOT_RISE]
    }
    const markersOf = (ids: Set<string>): Marker[] => [...ids].map((id) => ({ id, center: topSlot(id) }))
    const seats: Marker[] = [...seatSet].map((id) => {
      const [x, y] = topSlot(id)
      return { id, center: [marked(id) ? x - CROSS_BESIDE_MARKER : x, y] }
    })
    const terrains = map.territories.map((territory) => {
      const [x, y] = centers.get(territory.id)!
      const lowered = marked(territory.id) || seatSet.has(territory.id)
      return { id: territory.id, terrain: territoryById.get(territory.id)!.terrain, center: [x, y + (lowered ? TERRAIN_BELOW_TOP_SLOT : 0)] as Point }
    })
    return { regions, regionOf, bounds, castles: markersOf(castleSet), villages: markersOf(villageSet), seats, walls, terrains }
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
      {drawing.regions.map((region, index) => (
        <g key={region.id}>
          <clipPath id={`${clipPrefix}-region-${index}`}>
            {region.territories.map((id) => (
              <path key={id} d={pointsToPath(map.territories.find((territory) => territory.id === id)?.points ?? [])} />
            ))}
          </clipPath>
          <path
            data-preview-region={region.id}
            d={region.path}
            fill="none"
            stroke={region.color}
            strokeWidth={REGION_BORDER_WIDTH}
            strokeLinejoin="round"
            clipPath={`url(#${clipPrefix}-region-${index})`}
          />
        </g>
      ))}
      {drawing.terrains.map(({ id, terrain, center: [x, y] }) => (
        <g key={`t-${id}`} data-preview-terrain={terrain} transform={`translate(${x} ${y}) scale(${TERRAIN_SCALE})`}>
          <TerrainGlyph terrain={terrain} />
        </g>
      ))}
      {drawing.walls.map((wall) => (
        <path
          key={wall.key}
          data-preview-wall={wall.between.join('|')}
          d={zigzagPath(wall.from, wall.to)}
          fill="none"
          stroke={MAP_INK}
          strokeWidth={WALL_WIDTH}
          strokeLinejoin="round"
          strokeLinecap="round"
        />
      ))}
      {drawing.villages.map(({ id, center }) => (
        <Glyph key={`v-${id}`} kind="village" center={center} size={VILLAGE_SIZE} dataAttribute={{ 'data-preview-village': id }} />
      ))}
      {drawing.castles.map(({ id, center }) => (
        <Glyph key={`c-${id}`} kind="castle" center={center} size={CASTLE_SIZE} dataAttribute={{ 'data-preview-castle': id }} />
      ))}
      {drawing.seats.map(({ id, center: [x, y] }) => (
        <path
          key={`s-${id}`}
          data-preview-seat={id}
          d={CROSS_PATH}
          fill={MAP_PAPER_FILL}
          stroke={MAP_INK}
          strokeWidth={1.2}
          strokeLinejoin="round"
          transform={`translate(${x - (CROSS_BOX * CROSS_SCALE) / 2} ${y - (CROSS_BOX * CROSS_SCALE) / 2}) scale(${CROSS_SCALE})`}
        />
      ))}
    </svg>
  )
}
