import { borderIconPlacements, type IconPlacement } from '@/lib/chaotic-icons'
import { edgeKey, meanTerritoryArea, polygonEdges } from '@/lib/map-svg-geometry'
import { polygonContains } from '@/lib/region-geometry'
import type { MapData, Point, Terrain, Territory } from '@/types'

const REFERENCE_MAP_PLAYERS = 4
const REFERENCE_MAP_WIDTH = 1000
const REFERENCE_MAP_HEIGHT = 700
const REFERENCE_MAP_TERRITORIES =
  8 * REFERENCE_MAP_PLAYERS + 4 * (REFERENCE_MAP_PLAYERS + 1)
const REFERENCE_MEAN_TERRITORY_AREA =
  (REFERENCE_MAP_WIDTH * REFERENCE_MAP_HEIGHT) / REFERENCE_MAP_TERRITORIES

export interface OuterBorder {
  key: string
  from: Point
  to: Point
}

export interface SharedBorder {
  key: string
  from: Point
  to: Point
  passable: boolean
  /** Codes of the two territories the border separates, sorted. */
  between: [string, string]
}

export interface MapBorders {
  mapWidth: number
  mapHeight: number
  /** Polygon edges that belong to a single territory: the map's outline. */
  outerBorders: OuterBorder[]
  /** Edges shared by two declared neighbours, passable or not. */
  sharedBorders: SharedBorder[]
}

/** Icon series drawn along an impassable frontier, by adjacent terrains. */
export type ImpassableBorderKind = 'river' | 'marsh' | 'peaks' | 'cliff'

export interface ImpassableBorderIcon {
  key: string
  kind: ImpassableBorderKind
  /** Rotation in radians; a cliff faces +y of its rotated frame. */
  placement: IconPlacement
}

/** Relief rank used to turn an escarpment towards the lower territory. */
const RELIEF_RANK: Record<Terrain, number> = {
  mountain: 4,
  hill: 3,
  forest: 2,
  plain: 1,
  swamp: 0,
}

/**
 * Icon series of an impassable frontier, from the two terrains it separates:
 * marsh if either side is a swamp, peaks if either is a mountain, otherwise
 * an escarpment. Rivers (cartographie.md) are not generated yet.
 */
export function impassableBorderKind(
  first: Terrain,
  second: Terrain,
): ImpassableBorderKind {
  if (first === 'swamp' || second === 'swamp') return 'marsh'
  if (first === 'mountain' || second === 'mountain') return 'peaks'
  return 'cliff'
}

/**
 * The territory an escarpment faces: the lower relief, or on a tie the
 * larger territory code.
 */
export function lowerTerritory(first: Territory, second: Territory): Territory {
  const difference = RELIEF_RANK[first.terrain] - RELIEF_RANK[second.terrain]
  if (difference !== 0) return difference < 0 ? first : second
  return first.id > second.id ? first : second
}

/**
 * Scale annotations from the actual territory footprint, not the player
 * count, so markers keep a constant size relative to their territory.
 */
export function mapAnnotationScale(territories: Territory[]): number {
  const meanArea = meanTerritoryArea(territories)
  return meanArea > 0 ? Math.sqrt(meanArea / REFERENCE_MEAN_TERRITORY_AREA) : 1
}

/** Map extent and the outer/shared border segments derived from the polygons. */
export function computeMapBorders(map: MapData): MapBorders {
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  const territoriesById = new Map(
    map.territories.map((territory) => [territory.id, territory]),
  )
  const edges = new Map<string, { from: Point; to: Point; occurrences: number }>()
  const pairs = new Map<string, { ids: [string, string]; passable: boolean }>()

  const addPair = (firstId: string, secondId: string, passable: boolean) => {
    if (firstId === secondId || !territoriesById.has(secondId)) {
      return
    }

    const ids: [string, string] =
      firstId < secondId ? [firstId, secondId] : [secondId, firstId]
    const key = JSON.stringify(ids)
    const pair = pairs.get(key)
    if (pair) {
      pair.passable = pair.passable && passable
      return
    }

    pairs.set(key, { ids, passable })
  }

  for (const territory of map.territories) {
    for (const [x, y] of territory.points) {
      minX = Math.min(minX, x)
      minY = Math.min(minY, y)
      maxX = Math.max(maxX, x)
      maxY = Math.max(maxY, y)
    }

    for (const [from, to] of polygonEdges(territory.points)) {
      const key = edgeKey(from, to)
      const edge = edges.get(key)
      if (edge) {
        edge.occurrences += 1
      } else {
        edges.set(key, { from, to, occurrences: 1 })
      }
    }

    for (const adjacentId of territory.adjacencies) {
      addPair(territory.id, adjacentId, true)
    }
    for (const impassableId of territory.impassable) {
      addPair(territory.id, impassableId, false)
    }
  }

  const outerBorders: OuterBorder[] = []
  for (const [key, edge] of edges) {
    if (edge.occurrences === 1) {
      outerBorders.push({ key, from: edge.from, to: edge.to })
    }
  }

  const sharedBorders: SharedBorder[] = []
  const renderedEdges = new Set<string>()
  for (const [pairKey, pair] of pairs) {
    const first = territoriesById.get(pair.ids[0])
    const second = territoriesById.get(pair.ids[1])
    if (!first || !second) {
      continue
    }

    const secondEdges = new Set(
      polygonEdges(second.points).map(([from, to]) => edgeKey(from, to)),
    )
    for (const [from, to] of polygonEdges(first.points)) {
      const key = edgeKey(from, to)
      if (!secondEdges.has(key) || renderedEdges.has(key)) {
        continue
      }

      renderedEdges.add(key)
      sharedBorders.push({
        key: `${pairKey}-${key}`,
        from,
        to,
        passable: pair.passable,
        between: pair.ids,
      })
    }
  }

  return {
    mapWidth: Number.isFinite(minX) ? maxX + minX : 1,
    mapHeight: Number.isFinite(minY) ? maxY + minY : 1,
    outerBorders,
    sharedBorders,
  }
}

/**
 * Impassable frontiers render as a chaotic chain of icons along each shared
 * border segment instead of a plain stroke, the series chosen from the
 * adjacent terrains. Deterministic per segment so the chains never jump
 * between renders.
 */
export function impassableBorderIcons(
  sharedBorders: SharedBorder[],
  territories: Territory[],
  annotationScale: number,
): ImpassableBorderIcon[] {
  const territoriesById = new Map(
    territories.map((territory) => [territory.id, territory]),
  )
  const items: ImpassableBorderIcon[] = []
  for (const border of sharedBorders) {
    const first = territoriesById.get(border.between[0])
    const second = territoriesById.get(border.between[1])
    if (border.passable || !first || !second) {
      continue
    }
    const kind = impassableBorderKind(first.terrain, second.terrain)
    let rotation = 0
    if (kind === 'cliff') {
      // Turn the frame so +y points from the border into the lower side:
      // probe just off the segment's left normal, which stays right even
      // when the lower territory is concave (its centroid may lie anywhere).
      const lower = lowerTerritory(first, second)
      const dx = border.to[0] - border.from[0]
      const dy = border.to[1] - border.from[1]
      const length = Math.hypot(dx, dy) || 1
      const probe: Point = [
        (border.from[0] + border.to[0]) / 2 + (-dy / length) * 0.5,
        (border.from[1] + border.to[1]) / 2 + (dx / length) * 0.5,
      ]
      const facesLeft = polygonContains(lower.points, probe)
      rotation = Math.atan2(dy, dx) + (facesLeft ? 0 : Math.PI)
    }
    const placements = borderIconPlacements(
      border.from,
      border.to,
      border.key,
      18 * annotationScale,
      17 * annotationScale,
    )
    placements.forEach((placement, index) => {
      items.push({
        key: `${border.key}-${index}`,
        kind,
        placement: { ...placement, rotation },
      })
    })
  }
  // Depth-sort by canvas position: icons lower on the map paint over the
  // ones they overlap above them.
  return items.sort((first, second) => first.placement.y - second.placement.y)
}
