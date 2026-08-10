import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
  Terrain,
} from '@/types'

const MIN_ZOOM = 0.5
const MAX_ZOOM = 4
const DRAG_THRESHOLD = 4
const OUTER_BORDER_WIDTH = 2
const PASSABLE_BORDER_WIDTH = 2
const IMPASSABLE_BORDER_WIDTH = 4
const PASSABLE_BORDER_DASH = '4 3'

const TERRAIN_LABELS: Record<Terrain, string> = {
  plain: 'Plaine',
  forest: 'Forêt',
  hill: 'Colline',
  mountain: 'Montagne',
  swamp: 'Marécage',
}

const TERRAIN_COLORS: Record<Terrain, string> = {
  plain: '#b8d99a',
  forest: '#3f7854',
  hill: '#ad8565',
  mountain: '#89929a',
  swamp: '#66a6a0',
}

const TERRAIN_ORDER: Terrain[] = ['plain', 'forest', 'hill', 'mountain', 'swamp']

const PLAYER_PALETTE = ['#a84632', '#2d5f9e', '#7052a1', '#34775c', '#ad7a25']

const INFRASTRUCTURE_LABELS: Record<Infrastructure['type'], string> = {
  mill: 'Moulin',
  supply_depot: 'Dépôt de vivres',
  castle: 'Château',
  village: 'Village',
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
}: InfrastructureMarkerProps) {
  const label = `${INFRASTRUCTURE_LABELS[infrastructure.type]} niveau ${infrastructure.level}${isCapital ? ' · Capitale' : ''}`

  return (
    <g transform={`translate(${x} ${y})`} pointerEvents="none">
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

function NobleMarker({ noble, x, y }: { noble: Noble; x: number; y: number }) {
  return (
    <g transform={`translate(${x} ${y})`} pointerEvents="none">
      <title>{`${noble.name} (${noble.id})`}</title>
      <path d="M0-8L8 0L0 8L-8 0Z" fill="#f2c14e" stroke="#815f1e" strokeWidth="1.5" />
      <circle cx="0" cy="0" r="2" fill="#fff3c4" />
    </g>
  )
}

interface MapViewerProps {
  map: MapData
  state: StateData
  onSelect?: (id: string | null) => void
  supply?: SupplyLine | null
}

export function MapViewer({ map, state, onSelect, supply }: MapViewerProps) {
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

  const owners = new Set<string>()
  state.territories.forEach((territoryState) => {
    if (territoryState.owner) {
      owners.add(territoryState.owner)
    }
    if (territoryState.army) {
      owners.add(territoryState.army.owner)
    }
  })
  state.nobles.forEach((noble) => owners.add(noble.owner))

  const playerColors = new Map(
    Array.from(owners)
      .sort((first, second) => first.localeCompare(second))
      .map((owner, index) => [owner, PLAYER_PALETTE[index % PLAYER_PALETTE.length]]),
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
          aria-label="Carte des territoires"
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
                width="8"
                height="8"
                patternUnits="userSpaceOnUse"
                patternTransform="rotate(45)"
              >
                <line
                  x1="4"
                  y1="0"
                  x2="4"
                  y2="8"
                  stroke="#808080"
                  strokeWidth="2"
                  strokeOpacity="0.5"
                />
              </pattern>
            </defs>

            <g aria-label="Terrains">
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
                      aria-label={`${territory.name}, ${TERRAIN_LABELS[territory.terrain]}`}
                      onKeyDown={(event) => handleTerritoryKeyDown(event, territory.id)}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="top" sideOffset={8}>
                    <div className="space-y-0.5">
                      <p className="font-semibold">{territory.name}</p>
                      <p>{TERRAIN_LABELS[territory.terrain]}</p>
                    </div>
                  </TooltipContent>
                </Tooltip>
              ))}
            </g>

            {state.season === 'winter' && (
              <g aria-label="Voile hivernal" pointerEvents="none">
                <rect width={mapWidth} height={mapHeight} fill="#eaf3ff" opacity="0.14" />
              </g>
            )}

            <g aria-label="Contrôle territorial" pointerEvents="none">
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
                        strokeDasharray={PASSABLE_BORDER_DASH}
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
              <g aria-label="Zone de ravitaillement" pointerEvents="none">
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

            <g aria-label="Sélection" pointerEvents="none">
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
                        strokeDasharray={PASSABLE_BORDER_DASH}
                        strokeLinecap="round"
                        clipPath={`url(#territory-clip-${territory.id})`}
                        vectorEffect="non-scaling-stroke"
                      />
                    )}
                  </g>
                )
              })}
            </g>

            <g aria-label="Contours extérieurs" pointerEvents="none">
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

            <g aria-label="Frontières" pointerEvents="none">
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
                  strokeDasharray={border.passable ? PASSABLE_BORDER_DASH : undefined}
                  strokeLinecap="round"
                  vectorEffect="non-scaling-stroke"
                />
              ))}
            </g>

            {supplyPathPoints.length > 0 && (
              <g aria-label="Ligne de ravitaillement" pointerEvents="none">
                <polyline
                  points={supplyPathPoints.map(([x, y]) => `${x},${y}`).join(' ')}
                  fill="none"
                  stroke={supplyColor}
                  strokeWidth="5"
                  strokeDasharray="8 5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  vectorEffect="non-scaling-stroke"
                />
                <circle
                  cx={supplyPathPoints[0][0]}
                  cy={supplyPathPoints[0][1]}
                  r="7"
                  fill="#fff8e7"
                  stroke={supplyColor}
                  strokeWidth="3"
                  vectorEffect="non-scaling-stroke"
                />
                {supplyPathPoints.length > 1 && (
                  <circle
                    cx={supplyPathPoints[supplyPathPoints.length - 1][0]}
                    cy={supplyPathPoints[supplyPathPoints.length - 1][1]}
                    r="13"
                    fill="none"
                    stroke={supplyColor}
                    strokeWidth="2"
                    strokeDasharray="3 3"
                    vectorEffect="non-scaling-stroke"
                  />
                )}
              </g>
            )}

            <g aria-label="Couche vivante" pointerEvents="none">
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
                        x={centerX + 30}
                        y={centerY - 20}
                        fill="#59401f"
                        fontSize="11"
                        fontWeight="700"
                        textAnchor="middle"
                        stroke="#fff8e7"
                        strokeWidth="3"
                        paintOrder="stroke"
                      >
                        {`×${territoryState.resources}`}
                      </text>
                    )}
                    {territoryState.infrastructures.map((infrastructure, index) => (
                      <InfrastructureMarker
                        key={`${territory.id}-${infrastructure.type}-${index}`}
                        infrastructure={infrastructure}
                        x={centerX + index * 18 - 6}
                        y={centerY - 25}
                        isCapital={
                          infrastructure.type === 'castle' &&
                          state.players.some(
                            (player) => player.capitalTerritory === territory.id,
                          )
                        }
                      />
                    ))}
                    {territoryState.army && (
                      <g key={`${territory.id}-army`}>
                        <title>{`Armée de ${territoryState.army.owner}, taille ${territoryState.army.size}`}</title>
                        <circle
                          cx={centerX - 9}
                          cy={centerY + 26}
                          r="9"
                          fill={playerColors.get(territoryState.army.owner) ?? '#475569'}
                          stroke="#fff8e7"
                          strokeWidth="2"
                        />
                        <text
                          x={centerX - 9}
                          y={centerY + 29}
                          fill="#fff8e7"
                          fontSize="9"
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
                        x={centerX + 28 + index * 16}
                        y={centerY + 20}
                      />
                    ))}
                  </g>
                )
              })}
            </g>

            <g aria-label="Codes des territoires" pointerEvents="none">
              {map.territories.map((territory) => {
                const [centerX, centerY] = centroid(territory.points)
                return (
                  <text
                    key={territory.id}
                    x={centerX}
                    y={centerY + 4}
                    fill="#30291f"
                    fontSize="13"
                    fontWeight="800"
                    letterSpacing="0.5"
                    textAnchor="middle"
                    stroke="#f5ecd9"
                    strokeWidth="3"
                    paintOrder="stroke"
                  >
                    {territory.code}
                  </text>
                )
              })}
            </g>
          </g>
        </svg>

        <Card className="absolute bottom-4 left-4 w-[min(16rem,calc(100%-2rem))] border-[#b7a786] bg-[#fffaf0]/95 shadow-lg backdrop-blur-sm">
          <CardHeader className="gap-0 pb-2">
            <CardTitle className="text-sm uppercase tracking-[0.16em] text-[#594b3c]">
              Légende
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-xs text-[#594b3c]">
            <div className="grid grid-cols-2 gap-x-3 gap-y-1.5">
              {TERRAIN_ORDER.map((terrain) => (
                <div key={terrain} className="flex items-center gap-2">
                  <span
                    className="size-3 shrink-0 rounded-full border border-[#594b3c]/30"
                    style={{ backgroundColor: TERRAIN_COLORS[terrain] }}
                  />
                  <span>{TERRAIN_LABELS[terrain]}</span>
                </div>
              ))}
              <div className="flex items-center gap-2">
                <svg
                  className="size-3 shrink-0"
                  viewBox="-10 -10 20 20"
                  aria-hidden="true"
                >
                  <path
                    d="M-9 8V-1L0-10L9-1V8Z"
                    fill="#fff8e7"
                    stroke="#6b4c28"
                    strokeWidth="1.5"
                  />
                  <rect x="-4" y="1" width="8" height="7" fill="#b7834e" />
                </svg>
                <span>Village</span>
              </div>
              <div className="flex items-center gap-2">
                <svg
                  className="size-3 shrink-0"
                  viewBox="-11 -11 22 22"
                  aria-hidden="true"
                >
                  <path
                    d="M-9 9V-3H-5V-9H-1V-3H3V-9H7V-3H10V9Z"
                    fill="#efe6d0"
                    stroke="#5f4936"
                    strokeWidth="1.5"
                  />
                </svg>
                <span>Château</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="size-3 shrink-0 rounded-full border-2 border-[#fff8e7] bg-[#a84632]" />
                <span>Armée (pastille numérotée)</span>
              </div>
              <div className="flex items-center gap-2">
                <svg
                  className="size-3 shrink-0"
                  viewBox="-10 -10 20 20"
                  aria-hidden="true"
                >
                  <path
                    d="M0-8L8 0L0 8L-8 0Z"
                    fill="#f2c14e"
                    stroke="#815f1e"
                    strokeWidth="1.5"
                  />
                  <circle cx="0" cy="0" r="2" fill="#fff3c4" />
                </svg>
                <span>Noble</span>
              </div>
              <div className="flex items-center gap-2">
                <svg className="size-3 shrink-0" viewBox="0 0 16 16" aria-hidden="true">
                  <rect
                    x="3"
                    y="3"
                    width="10"
                    height="10"
                    fill="none"
                    stroke="#a84632"
                    strokeWidth="3"
                  />
                </svg>
                <span>Liséré coloré = contrôle territorial</span>
              </div>
            </div>
            <p className="border-t border-[#b7a786]/60 pt-2 leading-relaxed">
              Trait continu épais = frontière infranchissable · Trait pointillé =
              frontière franchissable
            </p>
          </CardContent>
        </Card>
      </div>
    </TooltipProvider>
  )
}
