import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react'

import { MapLegend, TERRAIN_COLORS, TERRAIN_LABEL_KEYS } from '@/components/MapLegend'
import { MapControls } from '@/components/MapControls'
import {
  Snowflake,
  TERRAIN_PATTERNS,
  TerrainPattern,
  WINTER_SNOW_FLAKES,
  WINTER_SNOW_TILE,
  WINTER_SNOW_VARIANT_ROTATIONS,
  snowPatternVariant,
} from '@/components/MapDecorations'
import { IntentionsOverlay } from '@/components/MapIntentionsOverlay'
import {
  DRAFT_INTENTION_COLOR,
  GameIconGlyph,
  INTENT_OUTLINE_COLOR,
  InfrastructureMarker,
  NobleMarker,
  OwnershipBadge,
  WinterBadge,
  WinterMarkerTriangle,
  WinterTransferArrow,
  winterMarkerTitle,
  winterReasonText,
} from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import type { Intention } from '@/lib/intent-overlay'
import type { WinterIntention } from '@/lib/winter-overlay'
import {
  CALAMITY_ICONS,
  CANCELED_KIND_BY_CARD,
  CARD_ICONS,
  parseSpecialOrderPlacements,
} from '@/lib/game-icons'
import {
  GAME_ICON_GLYPHS,
  type GameIconGlyph as GameIconGlyphSpec,
} from '@/lib/game-icon-glyphs'
import {
  borderIconPlacements,
  chaoticIconPlacements,
  type IconPlacement,
} from '@/lib/chaotic-icons'
import { NEUTRAL_PLAYER_ID } from '@/types'
import {
  DRAG_THRESHOLD,
  WHEEL_ZOOM_FACTOR,
  distanceBetween,
  midpointOf,
  pinchView,
  viewportScale,
  zoomAtCenter,
  zoomAtPoint,
  type MapPoint,
  type ViewState,
} from '@/lib/map-gestures'
import { regionStyle, type RegionStyle } from '@/lib/region-color'
import {
  computeRegionOutlines,
  fitLabelFontSize,
  loopIsHole,
  polygonContains,
  polylineLength,
} from '@/lib/region-geometry'
import {
  centroid,
  clientToSvgPoint,
  edgeKey,
  getTerritoryIdFromTarget,
  meanTerritoryArea,
  pointsToPath,
  polygonEdges,
  regionRingPath,
  splitBoundaryPaths,
} from '@/lib/map-svg-geometry'
import { formatCardCode, formatCardLabel } from '@/lib/card-hand'
import { hasSupplySource } from '@/lib/supply'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { CardKind, MapData, PlayerId, Point, StateData, SupplyLine } from '@/types'

const OUTER_BORDER_WIDTH = 2
const PASSABLE_BORDER_WIDTH = 2
const REFERENCE_MAP_PLAYERS = 4
const REFERENCE_MAP_WIDTH = 1000
const REFERENCE_MAP_HEIGHT = 700
const REFERENCE_MAP_TERRITORIES =
  8 * REFERENCE_MAP_PLAYERS + 4 * (REFERENCE_MAP_PLAYERS + 1)
const REFERENCE_MEAN_TERRITORY_AREA =
  (REFERENCE_MAP_WIDTH * REFERENCE_MAP_HEIGHT) / REFERENCE_MAP_TERRITORIES

const PLAYER_PALETTE = ['#a84632', '#2d5f9e', '#7052a1', '#0e7490', '#ad7a25']
const CALAMITY_KINDS: CardKind[] = ['plague', 'bad_weather', 'famine']

/** Padding around the map forming the regional name frame. */
const REGION_BAND_MARGIN = 26
/** Width of the gradient liseré hugging each region boundary, in map units. */
const REGION_BORDER_WIDTH = 10
/** Stepped falloff of the regional liseré: inset fractions and opacities. */
const REGION_BORDER_STEPS = [
  { fraction: 1, opacity: 0.18 },
  { fraction: 0.55, opacity: 0.42 },
  { fraction: 0.32, opacity: 0.72 },
  { fraction: 0.18, opacity: 1 },
]

const WINTER_ERROR_COLOR = '#c43b2a'
const WINTER_WARNING_COLOR = '#e07a30'

/** Rebel armies answer to no crown: they render in a neutral gray. */
const NEUTRAL_ARMY_COLOR = '#6b7280'

interface TrackedPointer {
  start: MapPoint
  last: MapPoint
}

interface GestureState {
  primaryId: number
  pointers: Map<number, TrackedPointer>
  mode: 'pending' | 'pan' | 'pinch'
  dragged: boolean
  territoryId: string | null
  initialView: ViewState
  initialDistance: number
  initialMidpoint: MapPoint
  /** Drag threshold in map units, derived from on-screen pixels. */
  threshold: number
}

interface MapViewerProps {
  map: MapData
  state: StateData
  onSelect?: (id: string | null) => void
  supply?: SupplyLine | null
  intentions?: Intention[]
  winterIntentions?: WinterIntention[]
  showIntentions?: boolean
  intentionsColor?: string
  onToggleIntentions?: (show: boolean) => void
  showOwnership?: boolean
  onToggleOwnership?: (show: boolean) => void
  showRegions?: boolean
  onToggleRegions?: (show: boolean) => void
  showCalamities?: boolean
  onToggleCalamities?: (show: boolean) => void
  showCards?: boolean
  onToggleCards?: (show: boolean) => void
  /** Deck-order drafts per player, parsed into the card overlay. */
  specialOrders?: Array<{ player: PlayerId; text: string }>
}

export function MapViewer({
  map,
  state,
  onSelect,
  supply,
  intentions = [],
  winterIntentions = [],
  showIntentions = false,
  intentionsColor = '#a84632',
  onToggleIntentions,
  showOwnership = true,
  onToggleOwnership,
  showRegions = false,
  onToggleRegions,
  showCalamities = true,
  onToggleCalamities,
  showCards = true,
  onToggleCards,
  specialOrders = [],
}: MapViewerProps) {
  const { t } = useLanguage()
  const svgRef = useRef<SVGSVGElement>(null)
  const gestureRef = useRef<GestureState | null>(null)
  const [view, setView] = useState<ViewState>({ x: 0, y: 0, k: 1 })
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [legendOpen, setLegendOpen] = useState(false)
  const { mapWidth, mapHeight, outerBorders, sharedBorders, passableBoundaryKeys } =
    useMemo(() => {
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

      const outerBorders: Array<{ key: string; from: Point; to: Point }> = []
      for (const [key, edge] of edges) {
        if (edge.occurrences === 1) {
          outerBorders.push({ key, from: edge.from, to: edge.to })
        }
      }

      const sharedBorders: Array<{
        key: string
        from: Point
        to: Point
        passable: boolean
      }> = []
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
    }, [map])

  const regionByTerritory = useMemo(() => {
    const result = new Map<string, string>()
    for (const region of map.regions ?? []) {
      for (const territoryID of region.territories) result.set(territoryID, region.id)
    }
    return result
  }, [map.regions])
  const regionStyleByID = new Map<string, RegionStyle>(
    [...new Set(regionByTerritory.values())]
      .sort()
      .map((regionID, index, regionIDs) => [
        regionID,
        regionStyle(index, regionIDs.length),
      ]),
  )

  // Scale annotations from the actual territory footprint, not the player count.
  const annotationScale = (() => {
    const meanArea = meanTerritoryArea(map.territories)
    return meanArea > 0 ? Math.sqrt(meanArea / REFERENCE_MEAN_TERRITORY_AREA) : 1
  })()

  const regionsActive = showRegions && (map.regions?.length ?? 0) > 0
  const regionsById = useMemo(
    () => new Map((map.regions ?? []).map((region) => [region.id, region])),
    [map.regions],
  )
  const regionBySeed = useMemo(
    () => new Map((map.regions ?? []).map((region) => [region.seed, region])),
    [map.regions],
  )

  const regionOutlines = useMemo(
    () =>
      regionsActive ? computeRegionOutlines(map.territories, map.regions ?? []) : null,
    [regionsActive, map.territories, map.regions],
  )

  const interiorRegionIDs = useMemo(() => {
    const interior = new Set<string>()
    if (!regionOutlines) {
      return interior
    }
    for (const [regionId, outline] of regionOutlines.outlines) {
      if (outline.outerSegments.length === 0) {
        interior.add(regionId)
      }
    }
    return interior
  }, [regionOutlines])

  const territoryAt = useMemo(() => {
    const entries = map.territories
    return (point: Point): string | undefined => {
      for (const territory of entries) {
        if (polygonContains(territory.points, point)) {
          return territory.id
        }
      }
      return undefined
    }
  }, [map.territories])

  const regionBands = useMemo(() => {
    if (!regionOutlines) {
      return [] as Array<{
        regionId: string
        piecePath: string
        labelX: number
        labelY: number
        labelAngle: number
        labelWidth: number
      }>
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
    const bands: Array<{
      regionId: string
      piecePath: string
      labelX: number
      labelY: number
      labelAngle: number
      labelWidth: number
    }> = []
    for (const [regionId, outline] of regionOutlines.outlines) {
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
          piece.push([
            center[0] + Math.cos(angle) * far,
            center[1] + Math.sin(angle) * far,
          ])
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
  }, [regionOutlines, mapWidth, mapHeight])

  const regionHoleTesters = useMemo(() => {
    const testers = new Map<string, (loop: Point[]) => boolean>()
    if (!regionOutlines) {
      return testers
    }
    for (const regionId of regionOutlines.outlines.keys()) {
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
  }, [regionOutlines, territoryAt, regionByTerritory])

  /**
   * Impassable frontiers render as a chaotic chain of mountain icons along
   * each shared border segment instead of a plain stroke. Deterministic per
   * segment so the chains never jump between renders.
   */
  const impassableBorderIcons = useMemo(() => {
    const items: Array<{ key: string; placement: IconPlacement }> = []
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
  }, [sharedBorders, annotationScale])

  const bandMarginX = regionsActive && regionBands.length > 0 ? REGION_BAND_MARGIN : 0
  const bandMarginY = bandMarginX
  const viewWidth = mapWidth + bandMarginX * 2
  const viewHeight = mapHeight + bandMarginY * 2

  const colorsByPlayer = new Map(state.players.map((player) => [player.id, player.color]))
  const owners = Array.from(
    new Set<string>([
      ...state.players.map((player) => player.id),
      ...state.territories.flatMap((territoryState) =>
        territoryState.owner ? [territoryState.owner] : [],
      ),
      ...state.territories.flatMap((territoryState) =>
        territoryState.army ? [territoryState.army.owner] : [],
      ),
      ...state.nobles.map((noble) => noble.owner),
    ]),
  ).sort((first, second) => first.localeCompare(second))
  const playerColors = new Map(
    owners.map((owner, index) => [
      owner,
      owner === NEUTRAL_PLAYER_ID
        ? NEUTRAL_ARMY_COLOR
        : (colorsByPlayer.get(owner) ?? PLAYER_PALETTE[index % PLAYER_PALETTE.length]),
    ]),
  )
  const selectedTerritoryState = state.territories.find(
    (territoryState) => territoryState.id === selectedId,
  )
  const selectedSupply =
    supply?.territory === selectedId &&
    ((supply.kind === 'army' && Boolean(selectedTerritoryState?.army)) ||
      (supply.kind === 'source' &&
        !selectedTerritoryState?.army &&
        hasSupplySource(selectedTerritoryState)))
      ? supply
      : null
  const supplyReachable = new Set(selectedSupply?.reachable ?? [])
  const supplyColor = playerColors.get(selectedSupply?.armyOwner ?? '') ?? '#a84632'
  const supplyPathPoints = (selectedSupply?.path ?? []).flatMap((territoryID) => {
    const territory = map.territories.find((candidate) => candidate.id === territoryID)
    return territory ? [centroid(territory.points)] : []
  })
  const passableBorderDash = `${4 * annotationScale} ${3 * annotationScale}`
  const passableBorderDots = `0.1 ${4.5 * annotationScale}`
  const supplyPathDash = `${8 * annotationScale} ${5 * annotationScale}`
  const supplyEndpointDash = `${3 * annotationScale} ${3 * annotationScale}`

  /**
   * Drafted canceling cards per (canceled calamity, region): BT against bad
   * weather, RA against famine. The first card cancels the calamity; per the
   * rules a second card of the same kind applies its regional bonus instead.
   */
  const canceledCardCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const { text } of specialOrders) {
      for (const placement of parseSpecialOrderPlacements(text)) {
        const canceledKind =
          placement.kind === 'fair_weather' || placement.kind === 'abundant_harvest'
            ? CANCELED_KIND_BY_CARD[placement.kind]
            : null
        if (canceledKind) {
          const key = `${canceledKind}-${placement.target}`
          counts.set(key, (counts.get(key) ?? 0) + 1)
        }
      }
    }
    return counts
  }, [specialOrders])

  /**
   * Regions where the drafted cards fully clear the calamity (two canceling
   * cards or more): the calamity icons disappear and the residual bonus
   * scatters instead. With a single canceling card the calamity icons stay,
   * marked with a cancellation badge.
   */
  const canceledCalamityRegions = useMemo(() => {
    const canceled = new Set<string>()
    for (const [key, count] of canceledCardCounts) {
      if (
        count >= 2 &&
        (state.activeRegionEffects ?? []).some(
          (effect) => `${effect.kind}-${effect.regionSeed}` === key,
        )
      ) {
        canceled.add(key)
      }
    }
    return canceled
  }, [canceledCardCounts, state.activeRegionEffects])

  /**
   * Regions where a single canceling card counters an active calamity: the
   * calamity icons stay until resolution, each marked with a circle-slash
   * badge.
   */
  const singleCanceledRegions = useMemo(() => {
    const regions = new Set<string>()
    for (const [key, count] of canceledCardCounts) {
      if (
        count === 1 &&
        (state.activeRegionEffects ?? []).some(
          (effect) => `${effect.kind}-${effect.regionSeed}` === key,
        )
      ) {
        regions.add(key.split('-')[1] ?? '')
      }
    }
    return regions
  }, [canceledCardCounts, state.activeRegionEffects])

  const calamityIcons = useMemo(() => {
    if (!showCalamities) {
      return []
    }
    const regionsBySeed = new Map(
      (map.regions ?? []).map((region) => [region.seed, region]),
    )
    const territoriesById = new Map(
      map.territories.map((territory) => [territory.id, territory]),
    )
    const items: Array<{
      key: string
      glyph: GameIconGlyphSpec
      fill: string
      stroke?: string
      strokeWidth?: number
      opacity: number
      canceled: boolean
      placement: IconPlacement
    }> = []
    for (const effect of state.activeRegionEffects ?? []) {
      if (canceledCalamityRegions.has(`${effect.kind}-${effect.regionSeed}`)) {
        continue
      }
      const style = CALAMITY_ICONS[effect.kind as keyof typeof CALAMITY_ICONS]
      if (!style) {
        continue
      }
      const region = regionsBySeed.get(effect.regionSeed)
      if (!region) {
        continue
      }
      for (const territoryID of region.territories) {
        const territory = territoriesById.get(territoryID)
        if (!territory) {
          continue
        }
        const seedKey = `${effect.kind}-${effect.regionSeed}-${territory.id}`
        for (const placement of chaoticIconPlacements(
          territory.points,
          style.count,
          seedKey,
        )) {
          items.push({
            key: `${seedKey}-${placement.x.toFixed(1)}-${placement.y.toFixed(1)}`,
            glyph: style.glyph,
            fill: style.fill,
            stroke: style.stroke,
            strokeWidth: style.strokeWidth,
            opacity: style.opacity,
            canceled: singleCanceledRegions.has(effect.regionSeed),
            placement,
          })
        }
      }
    }
    return items
  }, [
    showCalamities,
    map.regions,
    map.territories,
    state.activeRegionEffects,
    canceledCalamityRegions,
    singleCanceledRegions,
  ])

  const cardIcons = useMemo(() => {
    if (!showCards) {
      return { scatterItems: [] }
    }
    const regionsBySeed = new Map(
      (map.regions ?? []).map((region) => [region.seed, region]),
    )
    const territoriesById = new Map(
      map.territories.map((territory) => [territory.id, territory]),
    )
    const activeByRegion = new Map(
      (state.activeRegionEffects ?? []).map((effect) => [
        `${effect.kind}-${effect.regionSeed}`,
        true,
      ]),
    )
    const colorsByPlayer = new Map(
      state.players.map((player) => [player.id, player.color]),
    )
    const scatterItems: Array<{
      key: string
      glyph: GameIconGlyphSpec
      fill: string
      stroke?: string
      strokeWidth?: number
      opacity: number
      placement: IconPlacement
    }> = []
    const scatteredRegions = new Set<string>()
    for (const { player, text } of specialOrders) {
      for (const placement of parseSpecialOrderPlacements(text)) {
        if (placement.kind === 'revolt') {
          const territory = territoriesById.get(placement.target)
          if (!territory) {
            continue
          }
          const style = CARD_ICONS.revolt
          const seedKey = `revolt-${placement.target}`
          for (const placement2 of chaoticIconPlacements(
            territory.points,
            style.count,
            seedKey,
          )) {
            scatterItems.push({
              key: `${seedKey}-${placement2.x.toFixed(1)}-${placement2.y.toFixed(1)}`,
              glyph: style.glyph,
              fill: colorsByPlayer.get(player) ?? '#475569',
              stroke: style.stroke,
              strokeWidth: style.strokeWidth,
              opacity: style.opacity,
              placement: placement2,
            })
          }
          continue
        }
        // One canceling card against an active calamity only marks the
        // calamity with a cancellation badge; two or more cards clear the
        // calamity and the residual bonus scatters instead.
        const canceledKind = CANCELED_KIND_BY_CARD[placement.kind]
        if (
          activeByRegion.has(`${canceledKind}-${placement.target}`) &&
          !canceledCalamityRegions.has(`${canceledKind}-${placement.target}`)
        ) {
          continue
        }
        const region = regionsBySeed.get(placement.target)
        if (!region || scatteredRegions.has(`${placement.kind}-${placement.target}`)) {
          continue
        }
        scatteredRegions.add(`${placement.kind}-${placement.target}`)
        const style = CARD_ICONS[placement.kind]
        for (const territoryID of region.territories) {
          const territory = territoriesById.get(territoryID)
          if (!territory) {
            continue
          }
          const seedKey = `${placement.kind}-${placement.target}-${territory.id}`
          for (const placement2 of chaoticIconPlacements(
            territory.points,
            style.count,
            seedKey,
          )) {
            scatterItems.push({
              key: `${seedKey}-${placement2.x.toFixed(1)}-${placement2.y.toFixed(1)}`,
              glyph: style.glyph,
              fill: style.fill,
              stroke: style.stroke,
              strokeWidth: style.strokeWidth,
              opacity: style.opacity,
              placement: placement2,
            })
          }
        }
      }
    }
    return { scatterItems }
  }, [
    showCards,
    specialOrders,
    map.regions,
    map.territories,
    state.activeRegionEffects,
    state.players,
    canceledCalamityRegions,
  ])

  useEffect(() => {
    const svg = svgRef.current
    if (!svg) {
      return
    }

    const handleWheel = (event: WheelEvent) => {
      event.preventDefault()
      const cursor = clientToSvgPoint(
        svg,
        event.clientX,
        event.clientY,
        viewWidth,
        viewHeight,
        -bandMarginX,
        -bandMarginY,
      )
      const zoomFactor = event.deltaY < 0 ? WHEEL_ZOOM_FACTOR : 1 / WHEEL_ZOOM_FACTOR

      setView((current) => zoomAtPoint(current, cursor, zoomFactor))
    }

    svg.addEventListener('wheel', handleWheel, { passive: false })
    return () => svg.removeEventListener('wheel', handleWheel)
  }, [mapHeight, mapWidth, viewHeight, viewWidth, bandMarginX, bandMarginY])

  const handlePointerDown = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (event.button !== 0) {
      if (event.button === 1) {
        event.preventDefault()
      }
      return
    }

    const point = clientToSvgPoint(
      event.currentTarget,
      event.clientX,
      event.clientY,
      viewWidth,
      viewHeight,
      -bandMarginX,
      -bandMarginY,
    )
    const bounds = event.currentTarget.getBoundingClientRect()
    const mapUnitsPerPx =
      1 / viewportScale(viewWidth, viewHeight, bounds.width || 1, bounds.height || 1)
    const threshold = Math.max(
      DRAG_THRESHOLD,
      (event.pointerType === 'touch' ? 8 : DRAG_THRESHOLD) * mapUnitsPerPx,
    )
    event.currentTarget.setPointerCapture(event.pointerId)
    event.preventDefault()

    const gesture = gestureRef.current
    if (!gesture) {
      gestureRef.current = {
        primaryId: event.pointerId,
        pointers: new Map([[event.pointerId, { start: point, last: point }]]),
        mode: 'pending',
        dragged: false,
        territoryId: getTerritoryIdFromTarget(event.target),
        initialView: view,
        initialDistance: 0,
        initialMidpoint: point,
        threshold,
      }
      return
    }

    if (gesture.pointers.has(event.pointerId)) {
      return
    }

    gesture.pointers.set(event.pointerId, { start: point, last: point })
    if (gesture.pointers.size >= 2) {
      const [first, second] = Array.from(gesture.pointers.values())
      gesture.mode = 'pinch'
      gesture.dragged = true
      gesture.initialView = view
      gesture.initialDistance = Math.max(distanceBetween(first.last, second.last), 1)
      gesture.initialMidpoint = midpointOf(first.last, second.last)
      setIsDragging(true)
    }
  }

  const handlePointerMove = (event: ReactPointerEvent<SVGSVGElement>) => {
    const gesture = gestureRef.current
    if (!gesture) {
      return
    }
    const tracked = gesture.pointers.get(event.pointerId)
    if (!tracked) {
      return
    }

    const point = clientToSvgPoint(
      event.currentTarget,
      event.clientX,
      event.clientY,
      viewWidth,
      viewHeight,
      -bandMarginX,
      -bandMarginY,
    )
    const deltaX = point[0] - tracked.last[0]
    const deltaY = point[1] - tracked.last[1]
    tracked.last = point

    if (gesture.mode === 'pending') {
      const primary = gesture.pointers.get(gesture.primaryId)
      if (!primary) {
        return
      }
      if (distanceBetween(primary.start, primary.last) < gesture.threshold) {
        return
      }
      gesture.mode = 'pan'
      gesture.dragged = true
      setIsDragging(true)
    }

    if (gesture.mode === 'pinch') {
      const [first, second] = Array.from(gesture.pointers.values())
      if (!first || !second) {
        return
      }
      const currentMidpoint = midpointOf(first.last, second.last)
      const currentDistance = distanceBetween(first.last, second.last)
      setView(
        pinchView(
          gesture.initialView,
          gesture.initialMidpoint,
          gesture.initialDistance,
          currentMidpoint,
          currentDistance,
        ),
      )
      return
    }

    if (gesture.mode === 'pan' && (deltaX !== 0 || deltaY !== 0)) {
      setView((current) => ({
        ...current,
        x: current.x + deltaX,
        y: current.y + deltaY,
      }))
    }
  }

  const endGesture = (
    event: ReactPointerEvent<SVGSVGElement>,
    allowSelection: boolean,
  ) => {
    const gesture = gestureRef.current
    if (!gesture || !gesture.pointers.has(event.pointerId)) {
      return
    }

    const wasPending = gesture.mode === 'pending'
    gesture.pointers.delete(event.pointerId)
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }

    if (gesture.pointers.size === 0) {
      if (allowSelection && wasPending && !gesture.dragged) {
        selectTerritory(gesture.territoryId)
      }
      gestureRef.current = null
      setIsDragging(false)
      return
    }

    if (gesture.mode === 'pinch' && gesture.pointers.size === 1) {
      gesture.mode = 'pan'
    }
  }

  const handlePointerUp = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (event.button !== 0) {
      return
    }
    endGesture(event, true)
  }

  const handlePointerCancel = (event: ReactPointerEvent<SVGSVGElement>) => {
    endGesture(event, false)
  }

  const selectTerritory = (id: string | null) => {
    const nextId = id !== null && id !== selectedId ? id : null
    setSelectedId(nextId)
    onSelect?.(nextId)
  }

  const handleTerritoryKeyDown = (
    event: ReactKeyboardEvent<SVGPathElement>,
    id: string,
  ) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return
    }

    event.preventDefault()
    selectTerritory(id)
  }

  const handleControlZoom = (zoomFactor: number) => {
    if (zoomFactor === 0) {
      setView({ x: 0, y: 0, k: 1 })
      return
    }
    setView((current) => zoomAtCenter(current, mapWidth, mapHeight, zoomFactor))
  }

  return (
    <TooltipProvider delayDuration={100}>
      <div
        className={`relative h-full w-full overflow-hidden ${
          isDragging ? 'cursor-grabbing' : 'cursor-grab'
        }`}
      >
        <svg
          ref={svgRef}
          className="h-full w-full select-none"
          viewBox={`${-bandMarginX} ${-bandMarginY} ${viewWidth} ${viewHeight}`}
          preserveAspectRatio="xMidYMid meet"
          role="group"
          aria-label={t('map.territories')}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerCancel}
          style={{ touchAction: 'none' }}
        >
          <rect width={mapWidth} height={mapHeight} fill="#e6d8bb" />
          <g transform={`translate(${view.x} ${view.y}) scale(${view.k})`}>
            <defs>
              {map.territories.map((territory) => (
                <clipPath
                  key={territory.id}
                  id={`territory-clip-${territory.id}`}
                  clipPathUnits="userSpaceOnUse"
                >
                  <path d={pointsToPath(territory.points)} />
                </clipPath>
              ))}
              <pattern
                id="supply-zone-hatch"
                width={8 * annotationScale}
                height={8 * annotationScale}
                patternUnits="userSpaceOnUse"
                patternTransform="rotate(45)"
              >
                <line
                  x1={4 * annotationScale}
                  y1="0"
                  x2={4 * annotationScale}
                  y2={8 * annotationScale}
                  stroke="#808080"
                  strokeWidth={2 * annotationScale}
                  strokeOpacity="0.5"
                />
              </pattern>
              {TERRAIN_PATTERNS.map((terrain) => (
                <TerrainPattern
                  key={terrain.terrain}
                  terrain={terrain}
                  scale={annotationScale}
                />
              ))}
              {WINTER_SNOW_VARIANT_ROTATIONS.map((rotation, index) => (
                <pattern
                  key={rotation}
                  id={`winter-snow-${index}`}
                  width={WINTER_SNOW_TILE * annotationScale}
                  height={WINTER_SNOW_TILE * annotationScale}
                  patternUnits="userSpaceOnUse"
                  patternTransform={`rotate(${rotation})`}
                >
                  {/* Hand-scattered six-spoke snowflakes, three per tile. */}
                  <g
                    stroke="#f8fbff"
                    strokeWidth={0.9 * annotationScale}
                    strokeOpacity={0.8}
                    strokeLinecap="round"
                  >
                    {WINTER_SNOW_FLAKES.map((flake) => (
                      <Snowflake
                        key={`${flake.x}-${flake.y}`}
                        flake={flake}
                        scale={annotationScale}
                      />
                    ))}
                  </g>
                </pattern>
              ))}
              <clipPath id="region-frame-clip">
                <rect
                  x={-bandMarginX}
                  y={-bandMarginY}
                  width={viewWidth}
                  height={viewHeight}
                />
              </clipPath>
              <marker
                id="intent-arrow-outline"
                viewBox="0 0 10 10"
                refX="8.5"
                refY="5"
                markerWidth="3.8"
                markerHeight="3.8"
                orient="auto"
              >
                <path d="M0 0 L10 5 L0 10 Z" fill={INTENT_OUTLINE_COLOR} />
              </marker>
              <marker
                id="intent-arrow"
                viewBox="0 0 10 10"
                refX="8.5"
                refY="5"
                markerWidth="3.2"
                markerHeight="3.2"
                orient="auto"
              >
                <path d="M0 0 L10 5 L0 10 Z" fill={intentionsColor} />
              </marker>
              <marker
                id="intent-arrow-draft"
                viewBox="0 0 10 10"
                refX="8.5"
                refY="5"
                markerWidth="3.2"
                markerHeight="3.2"
                orient="auto"
              >
                <path d="M0 0 L10 5 L0 10 Z" fill={DRAFT_INTENTION_COLOR} />
              </marker>
              <marker
                id="intent-circle-outline"
                viewBox="0 0 10 10"
                refX="5"
                refY="5"
                markerWidth="4"
                markerHeight="4"
                orient="auto"
              >
                <circle
                  cx="5"
                  cy="5"
                  r="3.8"
                  fill="none"
                  stroke={INTENT_OUTLINE_COLOR}
                  strokeWidth="2.8"
                />
              </marker>
              <marker
                id="intent-circle-draft"
                viewBox="0 0 10 10"
                refX="5"
                refY="5"
                markerWidth="3.4"
                markerHeight="3.4"
                orient="auto"
              >
                <circle
                  cx="5"
                  cy="5"
                  r="3.8"
                  fill="none"
                  stroke={DRAFT_INTENTION_COLOR}
                  strokeWidth="2"
                />
              </marker>
              <marker
                id="intent-circle"
                viewBox="0 0 10 10"
                refX="5"
                refY="5"
                markerWidth="3.4"
                markerHeight="3.4"
                orient="auto"
              >
                <circle
                  cx="5"
                  cy="5"
                  r="3.8"
                  fill="none"
                  stroke={intentionsColor}
                  strokeWidth="2"
                />
              </marker>
            </defs>

            {regionBands.length > 0 && (
              <g
                aria-label={t('map.regionBands')}
                pointerEvents="none"
                clipPath="url(#region-frame-clip)"
              >
                {regionBands.map(({ regionId, piecePath }) => (
                  <path
                    key={`region-band-${regionId}-${regionBands.length}`}
                    data-region-band={regionId}
                    d={piecePath}
                    fill={regionStyleByID.get(regionId)?.fill ?? '#315a75'}
                    fillOpacity="0.6"
                    stroke="#30291f"
                    strokeOpacity="0.25"
                    strokeWidth="1"
                    vectorEffect="non-scaling-stroke"
                  />
                ))}
                {regionBands.map(
                  ({ regionId, labelX, labelY, labelAngle, labelWidth }) => {
                    const region = regionsById.get(regionId)
                    const seedTerritory = region
                      ? map.territories.find((territory) => territory.id === region.seed)
                      : undefined
                    if (!region || !seedTerritory) return null
                    const label = t('map.regionLabel', {
                      name: seedTerritory.name,
                      seed: region.seed,
                    })
                    return (
                      <text
                        key={`region-band-label-${regionId}-${labelX.toFixed(1)}`}
                        data-region-label={regionId}
                        transform={`translate(${labelX} ${labelY}) rotate(${labelAngle})`}
                        fill="#fff8e7"
                        fontSize={fitLabelFontSize(label, labelWidth * 0.85)}
                        fontWeight="800"
                        textAnchor="middle"
                        dominantBaseline="central"
                        stroke="#30291f"
                        strokeOpacity="0.45"
                        strokeWidth={3}
                        paintOrder="stroke"
                      >
                        {label}
                      </text>
                    )
                  },
                )}
              </g>
            )}

            <g aria-label={t('map.terrains')}>
              {map.territories.map((territory) => (
                <Tooltip key={territory.id}>
                  <TooltipTrigger asChild>
                    <path
                      data-territory-id={territory.id}
                      d={pointsToPath(territory.points)}
                      fill={TERRAIN_COLORS[territory.terrain]}
                      stroke="none"
                      tabIndex={0}
                      role="button"
                      aria-pressed={territory.id === selectedId}
                      aria-label={t('map.territoryLabel', {
                        name: territory.name,
                        terrain: t(TERRAIN_LABEL_KEYS[territory.terrain]),
                      })}
                      onKeyDown={(event) => handleTerritoryKeyDown(event, territory.id)}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="top" sideOffset={8}>
                    <div className="space-y-0.5">
                      <p className="font-semibold">{territory.name}</p>
                      <p>{t(TERRAIN_LABEL_KEYS[territory.terrain])}</p>
                    </div>
                  </TooltipContent>
                </Tooltip>
              ))}
            </g>

            <g aria-label={t('map.terrainTextures')} pointerEvents="none">
              {map.territories.map((territory) => (
                <path
                  key={territory.id}
                  d={pointsToPath(territory.points)}
                  fill={`url(#terrain-${territory.terrain})`}
                />
              ))}
            </g>
            {showRegions && (map.regions?.length ?? 0) > 0 && (
              <g aria-label={t('map.regions')} pointerEvents="none">
                {[...(regionOutlines?.outlines.entries() ?? [])].map(
                  ([regionId, outline]) => {
                    const color = regionStyleByID.get(regionId)?.fill ?? '#315a75'
                    const isHole = regionHoleTesters.get(regionId) ?? (() => false)
                    return (
                      <g key={`region-ring-${regionId}`} data-region-ring={regionId}>
                        {REGION_BORDER_STEPS.map((step) => (
                          <path
                            key={step.fraction}
                            d={regionRingPath(
                              outline.loops,
                              REGION_BORDER_WIDTH * step.fraction * annotationScale,
                              isHole,
                            )}
                            fill={color}
                            fillOpacity={step.opacity}
                            fillRule="evenodd"
                            stroke="none"
                          />
                        ))}
                      </g>
                    )
                  },
                )}
              </g>
            )}

            {state.season === 'winter' && showIntentions && (
              <IntentionsOverlay
                intentions={intentions}
                annotationScale={annotationScale}
                passableBorderDash={passableBorderDash}
                intentionsColor={intentionsColor}
              />
            )}

            {state.season === 'winter' && (
              <g
                aria-label={t('map.winterVeil')}
                data-winter-veil-overlay="true"
                pointerEvents="none"
              >
                <rect
                  x={-bandMarginX}
                  y={-bandMarginY}
                  width={viewWidth}
                  height={viewHeight}
                  fill="#eaf3ff"
                  opacity="0.2"
                />
              </g>
            )}
            {state.season === 'winter' && (
              <g aria-label={t('map.winterSnow')} pointerEvents="none">
                {map.territories.map((territory) => (
                  <path
                    key={territory.id}
                    d={pointsToPath(territory.points)}
                    fill={`url(#winter-snow-${snowPatternVariant(territory.id)})`}
                  />
                ))}
              </g>
            )}

            {showOwnership && (
              <g aria-label={t('map.control')} pointerEvents="none">
                {map.territories.map((territory) => {
                  const territoryState = state.territories.find(
                    (candidate) => candidate.id === territory.id,
                  )
                  const owner = territoryState?.owner
                  if (!owner) {
                    return null
                  }

                  const [centerX, centerY] = centroid(territory.points)
                  const ownerName =
                    state.players.find((player) => player.id === owner)?.name ?? owner
                  return (
                    <OwnershipBadge
                      key={territory.id}
                      ownerId={owner}
                      x={centerX + (territoryState.army ? -32 : -9) * annotationScale}
                      y={centerY + 26 * annotationScale}
                      color={playerColors.get(owner) ?? '#475569'}
                      scale={annotationScale}
                      label={t('map.ownershipBadge', { owner: ownerName })}
                    />
                  )
                })}
              </g>
            )}

            {supplyReachable.size > 0 && (
              <g aria-label={t('map.supplyZone')} pointerEvents="none">
                {map.territories.map((territory) => {
                  if (!supplyReachable.has(territory.id)) {
                    return null
                  }

                  return (
                    <path
                      key={territory.id}
                      data-supply-territory-id={territory.id}
                      d={pointsToPath(territory.points)}
                      fill="url(#supply-zone-hatch)"
                    />
                  )
                })}
              </g>
            )}

            <g aria-label={t('map.selection')} pointerEvents="none">
              {map.territories.map((territory) => {
                if (territory.id !== selectedId) {
                  return null
                }

                const { solidPath, passablePath } = splitBoundaryPaths(
                  territory.points,
                  passableBoundaryKeys,
                )

                return (
                  <g key={territory.id}>
                    {solidPath && (
                      <path
                        d={solidPath}
                        fill="none"
                        stroke="#d28b22"
                        strokeWidth="5"
                        strokeLinecap="round"
                        clipPath={`url(#territory-clip-${territory.id})`}
                        vectorEffect="non-scaling-stroke"
                      />
                    )}
                    {passablePath && (
                      <path
                        d={passablePath}
                        fill="none"
                        stroke="#d28b22"
                        strokeWidth="5"
                        strokeDasharray={passableBorderDots}
                        strokeLinecap="round"
                        clipPath={`url(#territory-clip-${territory.id})`}
                        vectorEffect="non-scaling-stroke"
                      />
                    )}
                  </g>
                )
              })}
            </g>

            <g aria-label={t('map.outerBorders')} pointerEvents="none">
              {outerBorders.map((border) => (
                <line
                  key={border.key}
                  x1={border.from[0]}
                  y1={border.from[1]}
                  x2={border.to[0]}
                  y2={border.to[1]}
                  stroke="#594b3c"
                  strokeOpacity="0.85"
                  strokeWidth={OUTER_BORDER_WIDTH}
                  strokeLinecap="round"
                  vectorEffect="non-scaling-stroke"
                />
              ))}
            </g>

            <g aria-label={t('map.borders')} pointerEvents="none">
              {sharedBorders
                .filter((border) => border.passable)
                .map((border) => (
                  <line
                    key={border.key}
                    x1={border.from[0]}
                    y1={border.from[1]}
                    x2={border.to[0]}
                    y2={border.to[1]}
                    stroke="#39271b"
                    strokeOpacity="0.85"
                    strokeWidth={PASSABLE_BORDER_WIDTH}
                    strokeDasharray={passableBorderDots}
                    strokeLinecap="round"
                    vectorEffect="non-scaling-stroke"
                  />
                ))}
              <g data-impassable-chain="true">
                {impassableBorderIcons.map((icon) => (
                  <GameIconGlyph
                    key={icon.key}
                    glyph={GAME_ICON_GLYPHS['peaks']}
                    x={icon.placement.x - icon.placement.size / 2}
                    y={icon.placement.y - icon.placement.size / 2}
                    size={icon.placement.size}
                    fill="#30291f"
                    stroke="#f5ecd9"
                    strokeWidth={20}
                    opacity={0.92}
                    rotation={(icon.placement.rotation * 180) / Math.PI}
                  />
                ))}
              </g>
            </g>

            {supplyPathPoints.length > 0 && (
              <g aria-label={t('map.supplyLine')} pointerEvents="none">
                <polyline
                  points={supplyPathPoints.map(([x, y]) => `${x},${y}`).join(' ')}
                  fill="none"
                  stroke={supplyColor}
                  strokeWidth="5"
                  strokeDasharray={supplyPathDash}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  vectorEffect="non-scaling-stroke"
                />
                <circle
                  cx={supplyPathPoints[0][0]}
                  cy={supplyPathPoints[0][1]}
                  r={7 * annotationScale}
                  fill="#fff8e7"
                  stroke={supplyColor}
                  strokeWidth="3"
                  vectorEffect="non-scaling-stroke"
                />
                {supplyPathPoints.length > 1 && (
                  <circle
                    cx={supplyPathPoints[supplyPathPoints.length - 1][0]}
                    cy={supplyPathPoints[supplyPathPoints.length - 1][1]}
                    r={13 * annotationScale}
                    fill="none"
                    stroke={supplyColor}
                    strokeWidth="2"
                    strokeDasharray={supplyEndpointDash}
                    vectorEffect="non-scaling-stroke"
                  />
                )}
              </g>
            )}

            {calamityIcons.length > 0 && (
              <g aria-label={t('map.calamityOverlay')} pointerEvents="none">
                {calamityIcons.map((icon) => (
                  <g key={icon.key}>
                    <GameIconGlyph
                      glyph={icon.glyph}
                      x={icon.placement.x - icon.placement.size / 2}
                      y={icon.placement.y - icon.placement.size / 2}
                      size={icon.placement.size}
                      fill={icon.fill}
                      stroke={icon.stroke}
                      strokeWidth={icon.strokeWidth}
                      opacity={icon.opacity}
                      rotation={(icon.placement.rotation * 180) / Math.PI}
                    />
                    {icon.canceled && (
                      <g pointerEvents="none">
                        <circle
                          cx={icon.placement.x}
                          cy={icon.placement.y}
                          r={icon.placement.size * 0.55}
                          fill="none"
                          stroke="#a84632"
                          strokeWidth={icon.placement.size * 0.16}
                          opacity={0.95}
                        />
                        <line
                          x1={icon.placement.x - icon.placement.size * 0.4}
                          y1={icon.placement.y - icon.placement.size * 0.4}
                          x2={icon.placement.x + icon.placement.size * 0.4}
                          y2={icon.placement.y + icon.placement.size * 0.4}
                          stroke="#a84632"
                          strokeWidth={icon.placement.size * 0.16}
                          opacity={0.95}
                        />
                      </g>
                    )}
                  </g>
                ))}
              </g>
            )}
            {cardIcons.scatterItems.length > 0 && (
              <g aria-label={t('map.cardOverlay')} pointerEvents="none">
                {cardIcons.scatterItems.map((icon) => (
                  <GameIconGlyph
                    key={icon.key}
                    glyph={icon.glyph}
                    x={icon.placement.x - icon.placement.size / 2}
                    y={icon.placement.y - icon.placement.size / 2}
                    size={icon.placement.size}
                    fill={icon.fill}
                    stroke={icon.stroke}
                    strokeWidth={icon.strokeWidth}
                    opacity={icon.opacity}
                    rotation={(icon.placement.rotation * 180) / Math.PI}
                  />
                ))}
              </g>
            )}
            <g aria-label={t('map.liveLayer')} pointerEvents="none">
              {map.territories.map((territory) => {
                const territoryState = state.territories.find(
                  (candidate) => candidate.id === territory.id,
                )
                if (!territoryState) {
                  return null
                }

                const [centerX, centerY] = centroid(territory.points)
                const territoryNobles = state.nobles.filter(
                  (noble) => noble.location === territory.id,
                )

                return (
                  <g key={territory.id}>
                    {territoryState.resources > 0 && (
                      <text
                        x={centerX + 30 * annotationScale}
                        y={centerY - 20 * annotationScale}
                        fill="#59401f"
                        fontSize={11 * annotationScale}
                        fontWeight="700"
                        textAnchor="middle"
                        stroke="#fff8e7"
                        strokeWidth={3 * annotationScale}
                        paintOrder="stroke"
                      >
                        {`×${territoryState.resources}`}
                      </text>
                    )}
                    {territoryState.infrastructures.map((infrastructure, index) => (
                      <InfrastructureMarker
                        key={`${territory.id}-${infrastructure.type}-${index}`}
                        infrastructure={infrastructure}
                        x={centerX + (index * 24 - 9) * annotationScale}
                        y={centerY - 25 * annotationScale}
                        isCapital={
                          infrastructure.type === 'castle' &&
                          state.players.some(
                            (player) => player.capitalTerritory === territory.id,
                          )
                        }
                        scale={annotationScale}
                        ownerColor={
                          territoryState.owner
                            ? (playerColors.get(territoryState.owner) ?? null)
                            : null
                        }
                      />
                    ))}
                    {territoryState.army && (
                      <g key={`${territory.id}-army`}>
                        <title>
                          {t('map.armyMarker', {
                            owner: territoryState.army.owner,
                            size: territoryState.army.size,
                          })}
                        </title>
                        <circle
                          cx={centerX - 9 * annotationScale}
                          cy={centerY + 26 * annotationScale}
                          r={9 * annotationScale}
                          fill={playerColors.get(territoryState.army.owner) ?? '#475569'}
                          stroke="#fff8e7"
                          strokeWidth={2 * annotationScale}
                        />
                        <text
                          x={centerX - 9 * annotationScale}
                          y={centerY + 29 * annotationScale}
                          fill="#fff8e7"
                          fontSize={9 * annotationScale}
                          fontWeight="800"
                          textAnchor="middle"
                        >
                          {territoryState.army.size}
                        </text>
                      </g>
                    )}
                    {territoryNobles.map((noble, index) => (
                      <NobleMarker
                        key={noble.id}
                        noble={noble}
                        x={centerX + (28 + index * 16) * annotationScale}
                        y={centerY + 20 * annotationScale}
                        color={playerColors.get(noble.owner) ?? '#475569'}
                        scale={annotationScale}
                      />
                    ))}
                  </g>
                )
              })}
            </g>

            <g aria-label={t('map.territoryLabels')} pointerEvents="none">
              {map.territories.map((territory) => {
                const [centerX, centerY] = centroid(territory.points)
                const chefRegion = regionsActive
                  ? regionBySeed.get(territory.id)
                  : undefined
                const chefFill = chefRegion
                  ? (regionStyleByID.get(chefRegion.id)?.fill ?? '#30291f')
                  : '#30291f'
                const labelProps = {
                  fontSize: 13 * annotationScale,
                  fontWeight: '800' as const,
                  letterSpacing: 0.5 * annotationScale,
                  textAnchor: 'middle' as const,
                  stroke: '#f5ecd9',
                  strokeWidth: 3 * annotationScale,
                  paintOrder: 'stroke' as const,
                }
                return chefRegion ? (
                  <g key={territory.id}>
                    <text
                      {...labelProps}
                      x={centerX}
                      y={centerY - 5 * annotationScale}
                      fill={chefFill}
                      fontSize={12 * annotationScale}
                      data-chef-lieu={territory.id}
                    >
                      {territory.name}
                    </text>
                    <text
                      {...labelProps}
                      x={centerX}
                      y={centerY + 11 * annotationScale}
                      fill="#30291f"
                    >
                      {territory.id}
                    </text>
                  </g>
                ) : (
                  <text
                    key={territory.id}
                    x={centerX}
                    y={centerY + 4 * annotationScale}
                    {...labelProps}
                    fill="#30291f"
                  >
                    {territory.id}
                  </text>
                )
              })}
            </g>

            {showRegions && (map.regions?.length ?? 0) > 0 && (
              <g aria-label={t('map.regionBadges')} pointerEvents="none">
                {(map.regions ?? [])
                  .filter((region) => interiorRegionIDs.has(region.id))
                  .map((region) => {
                    const seedTerritory = map.territories.find(
                      (territory) => territory.id === region.seed,
                    )
                    if (!seedTerritory) return null
                    const [centerX, centerY] = centroid(seedTerritory.points)
                    const color = regionStyleByID.get(region.id)?.fill ?? '#315a75'
                    const label = t('map.regionLabel', {
                      name: seedTerritory.name,
                      seed: region.seed,
                    })
                    const scale = annotationScale
                    const fontSize = 9.5 * scale
                    const width = label.length * fontSize * 0.62 + 10 * scale
                    const height = 15 * scale
                    return (
                      <g
                        key={`region-badge-${region.id}`}
                        data-region-label={region.id}
                        transform={`translate(${centerX} ${centerY - 56 * scale})`}
                      >
                        <title>{label}</title>
                        <rect
                          x={-width / 2}
                          y={-height / 2}
                          width={width}
                          height={height}
                          rx={height / 2}
                          fill="#fffaf0"
                          fillOpacity="0.94"
                          stroke={color}
                          strokeWidth={1.8 * scale}
                          vectorEffect="non-scaling-stroke"
                        />
                        <text
                          y={0.5 * scale}
                          fill={color}
                          fontSize={fontSize}
                          fontWeight="800"
                          textAnchor="middle"
                          dominantBaseline="central"
                        >
                          {label}
                        </text>
                      </g>
                    )
                  })}
              </g>
            )}

            {showRegions && (state.activeRegionEffects?.length ?? 0) > 0 && (
              <g aria-label={t('map.regionEffects')} pointerEvents="none">
                {state.activeRegionEffects?.map((effect, index) => {
                  const region = map.regions?.find(
                    (candidate) => candidate.seed === effect.regionSeed,
                  )
                  const seedTerritory = map.territories.find(
                    (territory) => territory.id === effect.regionSeed,
                  )
                  if (!region || !seedTerritory) return null
                  const [centerX, centerY] = centroid(seedTerritory.points)
                  const offset =
                    state.activeRegionEffects
                      ?.slice(0, index)
                      .filter((candidate) => candidate.regionSeed === effect.regionSeed)
                      .length ?? 0
                  const isCalamity = CALAMITY_KINDS.includes(effect.kind)
                  const effectColor = isCalamity ? '#b91c1c' : '#15803d'
                  const cardLabel = formatCardLabel(effect.kind, t)
                  return (
                    <g
                      key={`${effect.regionSeed}-${effect.kind}-${index}`}
                      data-region-effect-kind={effect.kind}
                      transform={`translate(${centerX + 24 * annotationScale} ${centerY - 25 * annotationScale + offset * 18 * annotationScale})`}
                    >
                      <title>
                        {t('map.regionEffectMarker', {
                          card: cardLabel,
                          region: effect.regionSeed,
                        })}
                      </title>
                      <circle
                        r={10 * annotationScale}
                        fill="#fffaf0"
                        stroke={effectColor}
                        strokeWidth={3 * annotationScale}
                        vectorEffect="non-scaling-stroke"
                      />
                      <text
                        y={4 * annotationScale}
                        fill={effectColor}
                        fontSize={8 * annotationScale}
                        fontWeight="900"
                        textAnchor="middle"
                      >
                        {formatCardCode(effect.kind, t)}
                      </text>
                    </g>
                  )
                })}
              </g>
            )}

            {state.season !== 'winter' && showIntentions && (
              <IntentionsOverlay
                intentions={intentions}
                annotationScale={annotationScale}
                passableBorderDash={passableBorderDash}
                intentionsColor={intentionsColor}
              />
            )}

            {showIntentions &&
              state.season === 'winter' &&
              winterIntentions.length > 0 && (
                <g
                  aria-label={t('map.winterOverlay')}
                  data-winter-orders-overlay="true"
                  pointerEvents="none"
                >
                  {map.territories.map((territory) => {
                    const [centerX, centerY] = centroid(territory.points)
                    const localIntentions = winterIntentions.filter(
                      (intention) => intention.territory === territory.id,
                    )
                    const builds = localIntentions.filter(
                      (intention) => intention.valid && intention.kind === 'build',
                    )
                    const badges = localIntentions.filter(
                      (intention) =>
                        intention.valid &&
                        intention.kind !== 'build' &&
                        intention.kind !== 'transfer',
                    )
                    const errors = localIntentions.filter((intention) => !intention.valid)

                    return (
                      <g key={`winter-${territory.id}`}>
                        {builds.map((intention, index) => {
                          const buildX = centerX + (-10 + index * 40) * annotationScale
                          const buildY = centerY - 38 * annotationScale
                          return (
                            <g key={`build-${intention.line}`}>
                              <InfrastructureMarker
                                infrastructure={{
                                  type: intention.infrastructure ?? 'mill',
                                  level: intention.level ?? 1,
                                }}
                                x={buildX}
                                y={buildY}
                                isCapital={false}
                                scale={annotationScale}
                                ownerColor={intention.color ?? DRAFT_INTENTION_COLOR}
                                variant="winterGhost"
                              />
                              {intention.warning && (
                                <WinterMarkerTriangle
                                  x={buildX + 26 * annotationScale}
                                  y={buildY + 11 * annotationScale}
                                  line={intention.line}
                                  title={winterMarkerTitle(t, intention)}
                                  scale={annotationScale}
                                  color={WINTER_WARNING_COLOR}
                                  dataTag="warning"
                                />
                              )}
                            </g>
                          )
                        })}
                        {badges.map((intention, index) => {
                          const label =
                            intention.kind === 'recruit_troop'
                              ? '+1'
                              : intention.kind === 'recruit_noble'
                                ? 'N'
                                : intention.kind === 'liberate'
                                  ? 'L'
                                  : intention.kind === 'hostage'
                                    ? 'O'
                                    : intention.kind === 'dungeon'
                                      ? 'P'
                                      : 'E'
                          const detail =
                            intention.kind === 'recruit_noble'
                              ? 'R'
                              : intention.kind === 'capital'
                                ? 'C'
                                : intention.noble
                          const badgeY = centerY + (-18 + index * 20) * annotationScale
                          return (
                            <g key={`${intention.kind}-${intention.line}`}>
                              <WinterBadge
                                x={centerX + 14 * annotationScale}
                                y={badgeY}
                                label={label}
                                detail={detail}
                                title={intention.label}
                                color={intention.color ?? DRAFT_INTENTION_COLOR}
                                scale={annotationScale}
                              />
                              {intention.warning && (
                                <WinterMarkerTriangle
                                  x={centerX - 8 * annotationScale}
                                  y={badgeY}
                                  line={intention.line}
                                  title={winterMarkerTitle(t, intention)}
                                  scale={annotationScale}
                                  color={WINTER_WARNING_COLOR}
                                  dataTag="warning"
                                />
                              )}
                            </g>
                          )
                        })}
                        {errors.map((intention, index) => {
                          const reason = winterReasonText(t, intention)
                          return (
                            <WinterMarkerTriangle
                              key={`error-${intention.line}-${index}`}
                              x={centerX - 22 * annotationScale}
                              y={centerY - (42 + index * 18) * annotationScale}
                              line={intention.line}
                              title={t('error.line', {
                                line: intention.line,
                                message: reason,
                              })}
                              scale={annotationScale}
                              color={WINTER_ERROR_COLOR}
                              dataTag="error"
                            />
                          )
                        })}
                      </g>
                    )
                  })}
                  {winterIntentions
                    .filter(
                      (intention) =>
                        intention.valid &&
                        intention.kind === 'transfer' &&
                        intention.sourceTerritory &&
                        intention.targetTerritory,
                    )
                    .map((intention) => {
                      const source = map.territories.find(
                        (territory) => territory.id === intention.sourceTerritory,
                      )
                      const target = map.territories.find(
                        (territory) => territory.id === intention.targetTerritory,
                      )
                      if (!source || !target) return null
                      const [midX, midY] = [
                        (centroid(source.points)[0] + centroid(target.points)[0]) / 2,
                        (centroid(source.points)[1] + centroid(target.points)[1]) / 2,
                      ]
                      return (
                        <g
                          key={`transfer-${intention.line}-${intention.sourceTerritory}-${intention.targetTerritory}`}
                        >
                          <WinterTransferArrow
                            from={centroid(source.points)}
                            to={centroid(target.points)}
                            amount={intention.amount}
                            title={intention.label}
                            color={intention.color ?? DRAFT_INTENTION_COLOR}
                            scale={annotationScale}
                          />
                          {intention.warning && (
                            <WinterMarkerTriangle
                              x={midX + 18 * annotationScale}
                              y={midY - 5 * annotationScale}
                              line={intention.line}
                              title={winterMarkerTitle(t, intention)}
                              scale={annotationScale}
                              color={WINTER_WARNING_COLOR}
                              dataTag="warning"
                            />
                          )}
                        </g>
                      )
                    })}
                  {winterIntentions
                    .filter((intention) => !intention.valid && !intention.territory)
                    .map((intention, index) => {
                      const reason = winterReasonText(t, intention)
                      return (
                        <WinterMarkerTriangle
                          key={`unplaced-error-${intention.line}-${index}`}
                          x={(18 + (index % 8) * 20) * annotationScale}
                          y={(22 + Math.floor(index / 8) * 18) * annotationScale}
                          line={intention.line}
                          title={t('error.line', {
                            line: intention.line,
                            message: reason,
                          })}
                          scale={annotationScale}
                          color={WINTER_ERROR_COLOR}
                          dataTag="error"
                        />
                      )
                    })}
                </g>
              )}
          </g>
        </svg>

        <MapControls onZoom={handleControlZoom} />

        {(onToggleIntentions ||
          onToggleOwnership ||
          onToggleRegions ||
          onToggleCalamities ||
          onToggleCards) && (
          <div className="absolute right-3 top-3 z-10">
            <button
              type="button"
              aria-expanded={legendOpen}
              aria-label={t(legendOpen ? 'legend.hide' : 'legend.show')}
              title={t(legendOpen ? 'legend.hide' : 'legend.show')}
              className="flex size-9 items-center justify-center rounded-lg border border-[#b7a786] bg-[#fffaf0] text-[#594b3c] shadow-md transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
              onClick={() => setLegendOpen((open) => !open)}
            >
              <svg
                aria-hidden="true"
                viewBox="0 0 24 24"
                className="size-4"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M4 4h6v6h-6z" />
                <path d="M14 6l6 0" />
                <path d="M14 10l6 0" />
                <path d="M4 14h16" />
                <path d="M4 18h16" />
              </svg>
            </button>
            {legendOpen && (
              <div className="absolute right-0 top-11 w-72 max-w-[calc(100vw-1.5rem)]">
                <MapLegend
                  showIntentions={showIntentions}
                  onToggleIntentions={onToggleIntentions}
                  showOwnership={showOwnership}
                  onToggleOwnership={onToggleOwnership}
                  showRegions={showRegions}
                  onToggleRegions={onToggleRegions}
                  showCalamities={showCalamities}
                  onToggleCalamities={onToggleCalamities}
                  showCards={showCards}
                  onToggleCards={onToggleCards}
                />
              </div>
            )}
          </div>
        )}
      </div>
    </TooltipProvider>
  )
}
