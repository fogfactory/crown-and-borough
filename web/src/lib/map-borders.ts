import { borderIconPlacements, type IconPlacement } from '@/lib/chaotic-icons'
import { edgeKey, meanTerritoryArea, polygonEdges } from '@/lib/map-svg-geometry'
import type { MapData, Point, Territory } from '@/types'

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
}

export interface MapBorders {
  mapWidth: number
  mapHeight: number
  /** Polygon edges that belong to a single territory: the map's outline. */
  outerBorders: OuterBorder[]
  /** Edges shared by two declared neighbours, passable or not. */
  sharedBorders: SharedBorder[]
  /** Edge keys of the passable shared borders, for the selection outline. */
  passableBoundaryKeys: Set<string>
}

export interface ImpassableBorderIcon {
  key: string
  placement: IconPlacement
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
      })
    }
  }

  return {
    mapWidth: Number.isFinite(minX) ? maxX + minX : 1,
    mapHeight: Number.isFinite(minY) ? maxY + minY : 1,
    outerBorders,
    sharedBorders,
    passableBoundaryKeys: new Set(
      sharedBorders
        .filter((border) => border.passable)
        .map((border) => edgeKey(border.from, border.to)),
    ),
  }
}

/**
 * Impassable frontiers render as a chaotic chain of mountain icons along
 * each shared border segment instead of a plain stroke. Deterministic per
 * segment so the chains never jump between renders.
 */
export function impassableBorderIcons(
  sharedBorders: SharedBorder[],
  annotationScale: number,
): ImpassableBorderIcon[] {
  const items: ImpassableBorderIcon[] = []
  for (const border of sharedBorders) {
    if (border.passable) {
      continue
    }
    const placements = borderIconPlacements(
      border.from,
      border.to,
      border.key,
      18 * annotationScale,
      17 * annotationScale,
    )
    placements.forEach((placement, index) => {
      items.push({ key: `${border.key}-${index}`, placement })
    })
  }
  // Depth-sort by canvas position: icons lower on the map paint over the
  // ones they overlap above them.
  return items.sort((first, second) => first.placement.y - second.placement.y)
}
