import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react'

import { TERRAIN_COLORS, TERRAIN_LABEL_KEYS } from '@/components/MapLegend'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { Intention } from '@/lib/intent-overlay'
import { hasSupplySource } from '@/lib/supply'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type {
  Infrastructure,
  MapData,
  Noble,
  Point,
  StateData,
  SupplyLine,
} from '@/types'

const MIN_ZOOM = 0.5
const MAX_ZOOM = 4
const DRAG_THRESHOLD = 4
const OUTER_BORDER_WIDTH = 2
const PASSABLE_BORDER_WIDTH = 2
const IMPASSABLE_BORDER_WIDTH = 4
const REFERENCE_MAP_PLAYERS = 4
const REFERENCE_MAP_WIDTH = 1000
const REFERENCE_MAP_HEIGHT = 700
const REFERENCE_MAP_TERRITORIES =
  8 * REFERENCE_MAP_PLAYERS + 4 * (REFERENCE_MAP_PLAYERS + 1)
const REFERENCE_MEAN_TERRITORY_AREA =
  (REFERENCE_MAP_WIDTH * REFERENCE_MAP_HEIGHT) / REFERENCE_MAP_TERRITORIES

const PLAYER_PALETTE = ['#a84632', '#2d5f9e', '#7052a1', '#34775c', '#ad7a25']
const INTENT_OUTLINE_COLOR = '#17120f'
export const DRAFT_INTENTION_COLOR = '#d4a39b'

const INFRASTRUCTURE_LABEL_KEYS: Record<Infrastructure['type'], MessageKey> = {
  mill: 'infrastructure.mill',
  supply_depot: 'infrastructure.supply_depot',
  castle: 'infrastructure.castle',
  village: 'infrastructure.village',
}

interface ViewState {
  x: number
  y: number
  k: number
}

interface DragState {
  pointerId: number
  mode: 'pan' | 'select'
  territoryId: string | null
  start: Point
  last: Point
  dragged: boolean
}

interface InfrastructureMarkerProps {
  infrastructure: Infrastructure
  x: number
  y: number
  isCapital: boolean
  scale: number
}

function pointsToPath(points: Point[]): string {
  if (points.length === 0) {
    return ''
  }

  const [first, ...rest] = points
  return `M ${first[0]},${first[1]} ${rest.map(([x, y]) => `L ${x},${y}`).join(' ')} Z`
}

function pointKey([x, y]: Point): string {
  return `${x},${y}`
}

function edgeKey(from: Point, to: Point): string {
  const fromKey = pointKey(from)
  const toKey = pointKey(to)

  return fromKey < toKey ? `${fromKey}|${toKey}` : `${toKey}|${fromKey}`
}

function polygonEdges(points: Point[]): Array<[Point, Point]> {
  if (points.length < 2) {
    return []
  }

  return points.map((point, index): [Point, Point] => [
    point,
    points[(index + 1) % points.length],
  ])
}

function segmentsToPath(segments: Array<[Point, Point]>): string {
  return segments
    .map(([[fromX, fromY], [toX, toY]]) => `M ${fromX},${fromY} L ${toX},${toY}`)
    .join(' ')
}

function splitBoundaryPaths(
  points: Point[],
  passableBoundaryKeys: Set<string>,
): { solidPath: string; passablePath: string } {
  const solidSegments: Array<[Point, Point]> = []
  const passableSegments: Array<[Point, Point]> = []

  for (const segment of polygonEdges(points)) {
    const [from, to] = segment
    if (passableBoundaryKeys.has(edgeKey(from, to))) {
      passableSegments.push(segment)
    } else {
      solidSegments.push(segment)
    }
  }

  return {
    solidPath: segmentsToPath(solidSegments),
    passablePath: segmentsToPath(passableSegments),
  }
}

function centroid(points: Point[]): Point {
  if (points.length === 0) {
    return [0, 0]
  }

  const total = points.reduce((sum, [x, y]) => ({ x: sum.x + x, y: sum.y + y }), {
    x: 0,
    y: 0,
  })

  return [total.x / points.length, total.y / points.length]
}

function polygonArea(points: Point[]): number {
  if (points.length < 3) {
    return 0
  }

  let twiceArea = 0
  for (let index = 0; index < points.length; index += 1) {
    const [x, y] = points[index]
    const [nextX, nextY] = points[(index + 1) % points.length]
    twiceArea += x * nextY - nextX * y
  }
  return Math.abs(twiceArea) / 2
}

function meanTerritoryArea(territories: MapData['territories']): number {
  if (territories.length === 0) {
    return 0
  }

  return (
    territories.reduce((total, territory) => total + polygonArea(territory.points), 0) /
    territories.length
  )
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum)
}

function clientToSvgPoint(
  svg: SVGSVGElement,
  clientX: number,
  clientY: number,
  mapWidth: number,
  mapHeight: number,
): Point {
  const bounds = svg.getBoundingClientRect()
  const width = bounds.width || 1
  const height = bounds.height || 1

  return [
    ((clientX - bounds.left) / width) * mapWidth,
    ((clientY - bounds.top) / height) * mapHeight,
  ]
}

function getTerritoryIdFromTarget(target: EventTarget | null): string | null {
  if (!(target instanceof Element)) {
    return null
  }

  return (
    target.closest<SVGPathElement>('[data-territory-id]')?.dataset.territoryId ?? null
  )
}

function InfrastructureMarker({
  infrastructure,
  x,
  y,
  isCapital,
  scale,
}: InfrastructureMarkerProps) {
  const { t } = useLanguage()
  const label = `${t(INFRASTRUCTURE_LABEL_KEYS[infrastructure.type])} · ${t('app.level', { level: infrastructure.level })}${isCapital ? ` · ${t('app.capital')}` : ''}`

  return (
    <g transform={`translate(${x} ${y}) scale(${scale})`} pointerEvents="none">
      <title>{label}</title>
      {infrastructure.type === 'castle' && (
        <>
          <path
            d="M-9 9V-3H-5V-9H-1V-3H3V-9H7V-3H10V9Z"
            fill="#efe6d0"
            stroke="#5f4936"
            strokeWidth="1.5"
          />
          {isCapital && (
            <g data-capital-marker="true" transform="translate(0 -7)">
              <path
                d="M-8 1L-6-7L-1-1L0-8L1-1L6-7L8 1Z"
                fill="#f2c14e"
                stroke="#815f1e"
                strokeWidth="1.25"
              />
              <path d="M-8 1H8" stroke="#815f1e" strokeWidth="1.25" />
            </g>
          )}
        </>
      )}
      {infrastructure.type === 'mill' && (
        <>
          <line x1="0" y1="0" x2="-9" y2="-7" stroke="#efe6d0" strokeWidth="3" />
          <line x1="0" y1="0" x2="9" y2="-7" stroke="#efe6d0" strokeWidth="3" />
          <line x1="0" y1="0" x2="-9" y2="7" stroke="#efe6d0" strokeWidth="3" />
          <line x1="0" y1="0" x2="9" y2="7" stroke="#efe6d0" strokeWidth="3" />
          <circle cx="0" cy="0" r="3" fill="#8b5e3c" stroke="#4e3828" />
        </>
      )}
      {infrastructure.type === 'supply_depot' && (
        <>
          <path d="M-9-4L0-9L9-4V8H-9Z" fill="#dcc08d" stroke="#705a36" />
          <path d="M-9-4H9M0-9V8" stroke="#705a36" />
        </>
      )}
      {infrastructure.type === 'village' && (
        <>
          <path d="M-9 8V-1L0-10L9-1V8Z" fill="#fff8e7" stroke="#6b4c28" />
          <rect x="-4" y="1" width="8" height="7" fill="#b7834e" />
          <path d="M-5-1H0L3-4" fill="none" stroke="#6b4c28" strokeWidth="1.5" />
        </>
      )}
      {infrastructure.level > 1 && (
        <text
          x="11"
          y="-7"
          fill="#4e3828"
          fontSize="9"
          fontWeight="700"
          textAnchor="middle"
        >
          {infrastructure.level}
        </text>
      )}
    </g>
  )
}

function NobleMarker({
  noble,
  x,
  y,
  color,
  scale,
}: {
  noble: Noble
  x: number
  y: number
  color: string
  scale: number
}) {
  const { t } = useLanguage()
  const prisoner = noble.status !== 'free'
  return (
    <g transform={`translate(${x} ${y}) scale(${scale})`} pointerEvents="none">
      <title>{`${noble.name} (${noble.id})${prisoner ? ` · ${t(`orders.nobleStatus.${noble.status}` as MessageKey)}` : ''}`}</title>
      <path
        d="M0-8L8 0L0 8L-8 0Z"
        fill={color}
        stroke={prisoner ? '#8d321e' : '#815f1e'}
        strokeWidth="1.5"
      />
      <circle cx="0" cy="0" r="2" fill="#fff3c4" />
      {prisoner && (
        <circle cx="0" cy="0" r="4.5" fill="none" stroke="#8d321e" strokeWidth="1.5" />
      )}
    </g>
  )
}

function IntentBadge({
  x,
  y,
  symbol,
  turnLabel,
  color,
  scale,
  isDraft,
}: {
  x: number
  y: number
  symbol: string
  turnLabel: string
  color: string
  scale: number
  isDraft: boolean
}) {
  const foreground = isDraft ? '#30291f' : '#fff8e7'

  return (
    <g transform={`translate(${x} ${y})`}>
      <circle
        cx="0"
        cy="0"
        r={8 * scale}
        fill={color}
        stroke="#fff8e7"
        strokeWidth={1.5 * scale}
      />
      <text
        x="0"
        y="0"
        fill={foreground}
        fontSize={9 * scale}
        fontWeight="800"
        textAnchor="middle"
        dominantBaseline="central"
      >
        {symbol}
      </text>
      <text
        x={11 * scale}
        y={-1 * scale}
        fill={isDraft ? '#30291f' : color}
        fontSize={9 * scale}
        fontWeight="700"
        textAnchor="middle"
        stroke="#fff8e7"
        strokeWidth={2.5 * scale}
        paintOrder="stroke"
      >
        {turnLabel}
      </text>
    </g>
  )
}

interface MapViewerProps {
  map: MapData
  state: StateData
  onSelect?: (id: string | null) => void
  supply?: SupplyLine | null
  intentions?: Intention[]
  showIntentions?: boolean
  intentionsColor?: string
  showRegions?: boolean
}

export function MapViewer({
  map,
  state,
  onSelect,
  supply,
  intentions = [],
  showIntentions = false,
  intentionsColor = '#a84632',
  showRegions = false,
}: MapViewerProps) {
  const { t } = useLanguage()
  const svgRef = useRef<SVGSVGElement>(null)
  const dragRef = useRef<DragState | null>(null)
  const [view, setView] = useState<ViewState>({ x: 0, y: 0, k: 1 })
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [isDragging, setIsDragging] = useState(false)
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
  const regionColors = ['#607d8b', '#78909c', '#546e7a', '#90a4ae', '#455a64']
  const regionColorByID = new Map(
    [...new Set(regionByTerritory.values())].sort().map((regionID, index) => [regionID, regionColors[index % regionColors.length]]),
  )

  const isRegionBoundary = (key: string) => {
    const end = key.indexOf(']')
    if (end < 0) return false
    try {
      const [first, second] = JSON.parse(key.slice(0, end + 1)) as [string, string]
      return regionByTerritory.get(first) !== regionByTerritory.get(second)
    } catch {
      return false
    }
  }

  const regionColorForBoundary = (key: string) => {
    const end = key.indexOf(']')
    if (end < 0) return '#607d8b'
    try {
      const [first] = JSON.parse(key.slice(0, end + 1)) as [string, string]
      return regionColorByID.get(regionByTerritory.get(first) ?? '') ?? '#607d8b'
    } catch {
      return '#607d8b'
    }
  }

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
      colorsByPlayer.get(owner) ?? PLAYER_PALETTE[index % PLAYER_PALETTE.length],
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
  const meanArea = meanTerritoryArea(map.territories)
  // Scale annotations from the actual territory footprint, not the player count.
  const annotationScale =
    meanArea > 0 ? Math.sqrt(meanArea / REFERENCE_MEAN_TERRITORY_AREA) : 1
  const passableBorderDash = `${4 * annotationScale} ${3 * annotationScale}`
  const supplyPathDash = `${8 * annotationScale} ${5 * annotationScale}`
  const supplyEndpointDash = `${3 * annotationScale} ${3 * annotationScale}`

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
        mapWidth,
        mapHeight,
      )
      const zoomFactor = event.deltaY < 0 ? 1.15 : 1 / 1.15

      setView((current) => {
        const nextZoom = clamp(current.k * zoomFactor, MIN_ZOOM, MAX_ZOOM)
        const ratio = nextZoom / current.k

        return {
          k: nextZoom,
          x: cursor[0] - (cursor[0] - current.x) * ratio,
          y: cursor[1] - (cursor[1] - current.y) * ratio,
        }
      })
    }

    svg.addEventListener('wheel', handleWheel, { passive: false })
    return () => svg.removeEventListener('wheel', handleWheel)
  }, [mapHeight, mapWidth])

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
      mapWidth,
      mapHeight,
    )
    event.currentTarget.setPointerCapture(event.pointerId)
    event.preventDefault()
    dragRef.current = {
      pointerId: event.pointerId,
      mode: event.pointerType === 'touch' ? 'pan' : 'select',
      territoryId: getTerritoryIdFromTarget(event.target),
      start: point,
      last: point,
      dragged: false,
    }
    setIsDragging(event.pointerType === 'touch')
  }

  const handlePointerMove = (event: ReactPointerEvent<SVGSVGElement>) => {
    const activeDrag = dragRef.current
    if (!activeDrag || activeDrag.pointerId !== event.pointerId) {
      return
    }

    const point = clientToSvgPoint(
      event.currentTarget,
      event.clientX,
      event.clientY,
      mapWidth,
      mapHeight,
    )
    const distance = Math.hypot(
      point[0] - activeDrag.start[0],
      point[1] - activeDrag.start[1],
    )
    if (distance >= DRAG_THRESHOLD) {
      activeDrag.dragged = true
      if (activeDrag.mode === 'select') {
        activeDrag.mode = 'pan'
        setIsDragging(true)
      }
    }

    const deltaX = point[0] - activeDrag.last[0]
    const deltaY = point[1] - activeDrag.last[1]
    if (activeDrag.mode === 'pan' && activeDrag.dragged) {
      setView((current) => {
        return {
          ...current,
          x: current.x + deltaX,
          y: current.y + deltaY,
        }
      })
    }
    activeDrag.last = point
  }

  const handlePointerUp = (event: ReactPointerEvent<SVGSVGElement>) => {
    const activeDrag = dragRef.current
    if (event.button !== 0 || !activeDrag || activeDrag.pointerId !== event.pointerId) {
      return
    }

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    if (activeDrag.mode === 'select' && !activeDrag.dragged) {
      selectTerritory(activeDrag.territoryId)
    }
    dragRef.current = null
    setIsDragging(false)
  }

  const handlePointerCancel = (event: ReactPointerEvent<SVGSVGElement>) => {
    const activeDrag = dragRef.current
    if (!activeDrag || activeDrag.pointerId !== event.pointerId) {
      return
    }

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    dragRef.current = null
    setIsDragging(false)
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
          viewBox={`0 0 ${mapWidth} ${mapHeight}`}
          preserveAspectRatio="none"
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

            {showRegions && (map.regions?.length ?? 0) > 0 && (
              <g aria-label={t('map.regions')} pointerEvents="none">
                {outerBorders.map((border) => (
                  <line key={`region-outer-${border.key}`} x1={border.from[0]} y1={border.from[1]} x2={border.to[0]} y2={border.to[1]} stroke="#607d8b" strokeWidth="3" strokeDasharray="8 5" vectorEffect="non-scaling-stroke" />
                ))}
                {sharedBorders.filter((border) => isRegionBoundary(border.key)).map((border) => (
                  <line key={`region-${border.key}`} x1={border.from[0]} y1={border.from[1]} x2={border.to[0]} y2={border.to[1]} stroke={regionColorForBoundary(border.key)} strokeWidth="3" strokeDasharray="8 5" vectorEffect="non-scaling-stroke" />
                ))}
              </g>
            )}

            {state.season === 'winter' && (
              <g aria-label={t('map.winterOverlay')} pointerEvents="none">
                <rect width={mapWidth} height={mapHeight} fill="#eaf3ff" opacity="0.14" />
              </g>
            )}

            <g aria-label={t('map.control')} pointerEvents="none">
              {map.territories.map((territory) => {
                const territoryState = state.territories.find(
                  (candidate) => candidate.id === territory.id,
                )
                const owner = territoryState?.owner
                if (!owner) {
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
                        stroke={playerColors.get(owner) ?? '#475569'}
                        strokeWidth="8"
                        strokeLinecap="round"
                        clipPath={`url(#territory-clip-${territory.id})`}
                        vectorEffect="non-scaling-stroke"
                      />
                    )}
                    {passablePath && (
                      <path
                        d={passablePath}
                        fill="none"
                        stroke={playerColors.get(owner) ?? '#475569'}
                        strokeWidth="8"
                        strokeDasharray={passableBorderDash}
                        strokeLinecap="round"
                        clipPath={`url(#territory-clip-${territory.id})`}
                        vectorEffect="non-scaling-stroke"
                      />
                    )}
                  </g>
                )
              })}
            </g>

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
                        strokeDasharray={passableBorderDash}
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
              {sharedBorders.map((border) => (
                <line
                  key={border.key}
                  x1={border.from[0]}
                  y1={border.from[1]}
                  x2={border.to[0]}
                  y2={border.to[1]}
                  stroke="#39271b"
                  strokeOpacity="0.85"
                  strokeWidth={
                    border.passable ? PASSABLE_BORDER_WIDTH : IMPASSABLE_BORDER_WIDTH
                  }
                  strokeDasharray={border.passable ? passableBorderDash : undefined}
                  strokeLinecap="round"
                  vectorEffect="non-scaling-stroke"
                />
              ))}
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
                        x={centerX + (index * 18 - 6) * annotationScale}
                        y={centerY - 25 * annotationScale}
                        isCapital={
                          infrastructure.type === 'castle' &&
                          state.players.some(
                            (player) => player.capitalTerritory === territory.id,
                          )
                        }
                        scale={annotationScale}
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
                return (
                  <text
                    key={territory.id}
                    x={centerX}
                    y={centerY + 4 * annotationScale}
                    fill="#30291f"
                    fontSize={13 * annotationScale}
                    fontWeight="800"
                    letterSpacing={0.5 * annotationScale}
                    textAnchor="middle"
                    stroke="#f5ecd9"
                    strokeWidth={3 * annotationScale}
                    paintOrder="stroke"
                  >
                    {territory.id}
                  </text>
                )
              })}
            </g>

            {showIntentions && intentions.length > 0 && (
              <g aria-label={t('map.intentionsOverlay')} pointerEvents="none">
                {intentions.map((intention, index) => {
                  const isDraft = intention.source === 'draft'
                  const intentionColor = isDraft ? DRAFT_INTENTION_COLOR : intentionsColor
                  const markerEndFor = (kind: 'arrow' | 'circle') =>
                    `url(#intent-${kind}${isDraft ? '-draft' : ''})`

                  return (
                    <g key={`${intention.armyTerritory}-${index}`}>
                      <title>
                        {intention.nobleCode ? `${intention.nobleCode} · ` : ''}
                        {intention.label}
                      </title>
                      {intention.segments.map((segment, segmentIndex) => {
                        const common = {
                          x1: segment.from[0],
                          y1: segment.from[1],
                          x2: segment.to[0],
                          y2: segment.to[1],
                        }
                        if (segment.kind === 'loop') {
                          const radius = 12 * annotationScale
                          const direction = segmentIndex % 2 === 0 ? 1 : -1
                          const sweep = direction === 1 ? 0 : 1
                          const path = `M ${segment.from[0] + radius * direction} ${segment.from[1]} A ${radius} ${radius} 0 1 ${sweep} ${segment.from[0]} ${segment.from[1] - radius}`
                          return (
                            <g key={segmentIndex}>
                              <path
                                d={path}
                                data-intent-outline="true"
                                fill="none"
                                stroke={INTENT_OUTLINE_COLOR}
                                strokeWidth={4.5 * annotationScale}
                                strokeLinecap="round"
                                markerEnd="url(#intent-arrow-outline)"
                              />
                              <path
                                d={path}
                                fill="none"
                                stroke={intentionColor}
                                strokeWidth={2.5 * annotationScale}
                                strokeLinecap="round"
                                markerEnd={markerEndFor('arrow')}
                              />
                              <IntentBadge
                                x={segment.from[0] + direction * -16 * annotationScale}
                                y={segment.from[1] - 22 * annotationScale}
                                symbol={intention.symbol}
                                turnLabel={intention.turnLabel}
                                color={intentionColor}
                                scale={annotationScale}
                                isDraft={isDraft}
                              />
                            </g>
                          )
                        }
                        const strokeWidth =
                          segment.kind === 'attack'
                            ? 4 * annotationScale
                            : 2.5 * annotationScale
                        const strokeDasharray =
                          segment.kind === 'support-defensive' ||
                          segment.kind === 'support-offensive'
                            ? passableBorderDash
                            : undefined
                        const markerKind =
                          segment.kind === 'support-defensive' ? 'circle' : 'arrow'
                        const outlineMarkerEnd =
                          markerKind === 'circle'
                            ? 'url(#intent-circle-outline)'
                            : 'url(#intent-arrow-outline)'

                        return (
                          <g key={segmentIndex}>
                            <line
                              {...common}
                              data-intent-outline="true"
                              stroke={INTENT_OUTLINE_COLOR}
                              strokeWidth={strokeWidth + 2 * annotationScale}
                              strokeDasharray={strokeDasharray}
                              strokeLinecap="round"
                              markerEnd={outlineMarkerEnd}
                            />
                            <line
                              {...common}
                              stroke={intentionColor}
                              strokeWidth={strokeWidth}
                              strokeDasharray={strokeDasharray}
                              strokeLinecap="round"
                              markerEnd={markerEndFor(markerKind)}
                            />
                            <IntentBadge
                              x={(segment.from[0] + segment.to[0]) / 2}
                              y={(segment.from[1] + segment.to[1]) / 2}
                              symbol={intention.symbol}
                              turnLabel={intention.turnLabel}
                              color={intentionColor}
                              scale={annotationScale}
                              isDraft={isDraft}
                            />
                          </g>
                        )
                      })}
                      {intention.segments.length === 0 && (
                        <IntentBadge
                          x={intention.from[0] + 20 * annotationScale}
                          y={intention.from[1] - 22 * annotationScale}
                          symbol={intention.symbol}
                          turnLabel={intention.turnLabel}
                          color={intentionColor}
                          scale={annotationScale}
                          isDraft={isDraft}
                        />
                      )}
                    </g>
                  )
                })}
              </g>
            )}
          </g>
        </svg>
      </div>
    </TooltipProvider>
  )
}
