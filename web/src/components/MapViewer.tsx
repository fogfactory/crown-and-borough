import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { Infrastructure, MapData, Noble, Point, StateData, Terrain } from '@/types'

const MIN_ZOOM = 0.5
const MAX_ZOOM = 4
const DRAG_THRESHOLD = 4

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
  post_relay: 'Relais de poste',
  watchtower: 'Tour de guet',
  supply_depot: 'Dépôt de vivres',
  castle: 'Château',
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
}

interface AdjacencyArc {
  key: string
  from: Point
  to: Point
}

function pointsToPath(points: Point[]): string {
  if (points.length === 0) {
    return ''
  }

  const [first, ...rest] = points
  return `M ${first[0]},${first[1]} ${rest.map(([x, y]) => `L ${x},${y}`).join(' ')} Z`
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

function InfrastructureMarker({ infrastructure, x, y }: InfrastructureMarkerProps) {
  const label = `${INFRASTRUCTURE_LABELS[infrastructure.type]} niveau ${infrastructure.level}`

  return (
    <g transform={`translate(${x} ${y})`} pointerEvents="none">
      <title>{label}</title>
      {infrastructure.type === 'castle' && (
        <path
          d="M-9 9V-3H-5V-9H-1V-3H3V-9H7V-3H10V9Z"
          fill="#efe6d0"
          stroke="#5f4936"
          strokeWidth="1.5"
        />
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
      {infrastructure.type === 'post_relay' && (
        <>
          <path d="M-9 8V-1L0-9L9-1V8Z" fill="#efe6d0" stroke="#5f4936" />
          <path d="M-3 8V1H3V8" fill="#9d7048" stroke="#5f4936" />
        </>
      )}
      {infrastructure.type === 'watchtower' && (
        <>
          <path d="M-7 9L-4-7H4L7 9Z" fill="#d9e0e3" stroke="#53606a" />
          <path d="M-7-7H7M-5-11H5" stroke="#53606a" strokeWidth="2" />
        </>
      )}
      {infrastructure.type === 'supply_depot' && (
        <>
          <path d="M-9-4L0-9L9-4V8H-9Z" fill="#dcc08d" stroke="#705a36" />
          <path d="M-9-4H9M0-9V8" stroke="#705a36" />
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

function LieuDitMarker({ x, y }: { x: number; y: number }) {
  return (
    <g transform={`translate(${x} ${y})`} pointerEvents="none">
      <title>Lieu-dit</title>
      <path d="M-8 0L0-8L8 0V8H-8Z" fill="#fff8e7" stroke="#785b36" />
      <rect x="-2" y="3" width="4" height="5" fill="#b7834e" />
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
  onSelect?: (id: string) => void
}

export function MapViewer({ map, state, onSelect }: MapViewerProps) {
  const svgRef = useRef<SVGSVGElement>(null)
  const dragRef = useRef<DragState | null>(null)
  const [view, setView] = useState<ViewState>({ x: 0, y: 0, k: 1 })
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const { mapWidth, mapHeight, adjacencyArcs } = useMemo(() => {
    let mapWidth = 1
    let mapHeight = 1
    const territoryIndexes = new Map<string, number>()
    const centers = map.territories.map((territory, index) => {
      territoryIndexes.set(territory.id, index)

      for (const [x, y] of territory.points) {
        mapWidth = Math.max(mapWidth, x)
        mapHeight = Math.max(mapHeight, y)
      }

      return centroid(territory.points)
    })
    const adjacencyArcs: AdjacencyArc[] = []
    for (const [territoryIndex, territory] of map.territories.entries()) {
      for (const adjacentID of territory.adjacencies) {
        const adjacentIndex = territoryIndexes.get(adjacentID)
        if (adjacentIndex === undefined || adjacentIndex <= territoryIndex) {
          continue
        }

        adjacencyArcs.push({
          key: `${territory.id}-${adjacentID}`,
          from: centers[territoryIndex],
          to: centers[adjacentIndex],
        })
      }
    }

    return { mapWidth, mapHeight, adjacencyArcs }
  }, [map])

  const owners = new Set<string>()
  state.territories.forEach((territoryState) => {
    if (territoryState.owner) {
      owners.add(territoryState.owner)
    }
    territoryState.troops.forEach((troop) => owners.add(troop.owner))
  })
  state.nobles.forEach((noble) => owners.add(noble.owner))

  const playerColors = new Map(
    Array.from(owners)
      .sort((first, second) => first.localeCompare(second))
      .map((owner, index) => [owner, PLAYER_PALETTE[index % PLAYER_PALETTE.length]]),
  )

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
    const mode = event.button === 0 ? 'pan' : event.button === 1 ? 'select' : null
    if (!mode) {
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
      mode,
      territoryId: getTerritoryIdFromTarget(event.target),
      start: point,
      last: point,
      dragged: false,
    }
    setIsDragging(mode === 'pan')
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
    }

    if (activeDrag.mode === 'pan' && activeDrag.dragged) {
      setView((current) => ({
        ...current,
        x: current.x + point[0] - activeDrag.last[0],
        y: current.y + point[1] - activeDrag.last[1],
      }))
    }
    activeDrag.last = point
  }

  const handlePointerUp = (event: ReactPointerEvent<SVGSVGElement>) => {
    const activeDrag = dragRef.current
    if (!activeDrag || activeDrag.pointerId !== event.pointerId) {
      return
    }

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    if (activeDrag.mode === 'select' && !activeDrag.dragged && activeDrag.territoryId) {
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

  const selectTerritory = (id: string) => {
    setSelectedId(id)
    onSelect?.(id)
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
            <g aria-label="Terrains">
              {map.territories.map((territory) => (
                <Tooltip key={territory.id}>
                  <TooltipTrigger asChild>
                    <path
                      data-territory-id={territory.id}
                      d={pointsToPath(territory.points)}
                      fill={TERRAIN_COLORS[territory.terrain]}
                      stroke="#594b3c"
                      strokeWidth="1.5"
                      tabIndex={0}
                      role="button"
                      aria-label={`${territory.name}, ${TERRAIN_LABELS[territory.terrain]}`}
                      onKeyDown={(event) => handleTerritoryKeyDown(event, territory.id)}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="top" sideOffset={8}>
                    <div className="space-y-0.5">
                      <p className="font-semibold">{territory.name}</p>
                      <p>{TERRAIN_LABELS[territory.terrain]}</p>
                      {(state.asOf[territory.id] ?? state.turn) < state.turn && (
                        <p>Données du tour {state.asOf[territory.id]}</p>
                      )}
                    </div>
                  </TooltipContent>
                </Tooltip>
              ))}
            </g>

            <g aria-label="Contrôle territorial" pointerEvents="none">
              {map.territories.map((territory) => {
                const territoryState = state.territories.find(
                  (candidate) => candidate.id === territory.id,
                )
                const owner = territoryState?.owner
                if (!owner) {
                  return null
                }

                return (
                  <path
                    key={territory.id}
                    d={pointsToPath(territory.points)}
                    fill={playerColors.get(owner) ?? '#475569'}
                    opacity="0.13"
                  />
                )
              })}
            </g>

            <g aria-label="Données anciennes" pointerEvents="none">
              {map.territories.map((territory) => {
                const observedTurn = state.asOf[territory.id] ?? state.turn
                if (observedTurn >= state.turn) {
                  return null
                }

                return (
                  <path
                    key={territory.id}
                    d={pointsToPath(territory.points)}
                    fill="black"
                    opacity="0.3"
                  />
                )
              })}
            </g>

            <g aria-label="Adjacences franchissables" pointerEvents="none">
              {adjacencyArcs.map((arc) => (
                <line
                  key={arc.key}
                  x1={arc.from[0]}
                  y1={arc.from[1]}
                  x2={arc.to[0]}
                  y2={arc.to[1]}
                  stroke="#39271b"
                  strokeOpacity="0.85"
                  strokeWidth="2"
                  strokeDasharray="6 3"
                  strokeLinecap="round"
                  vectorEffect="non-scaling-stroke"
                />
              ))}
            </g>

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
                    {territory.lieuDit && territoryState.resources > 0 && (
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
                    {territory.lieuDit && (
                      <LieuDitMarker x={centerX - 29} y={centerY - 18} />
                    )}
                    {territoryState.infrastructures.map((infrastructure, index) => (
                      <InfrastructureMarker
                        key={`${territory.id}-${infrastructure.type}-${index}`}
                        infrastructure={infrastructure}
                        x={centerX + index * 18 - 6}
                        y={centerY - 25}
                      />
                    ))}
                    {territoryState.troops.map((troop, index) => (
                      <g key={troop.id}>
                        <title>{`${troop.id}, ${troop.owner}`}</title>
                        <circle
                          cx={centerX - 8 + index * 16}
                          cy={centerY + 26}
                          r="7"
                          fill={playerColors.get(troop.owner) ?? '#475569'}
                          stroke="#fff8e7"
                          strokeWidth="2"
                        />
                        <text
                          x={centerX - 8 + index * 16}
                          y={centerY + 29}
                          fill="#fff8e7"
                          fontSize="7"
                          fontWeight="700"
                          textAnchor="middle"
                        >
                          {troop.id.slice(2)}
                        </text>
                      </g>
                    ))}
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

            <g aria-label="Lieux-dits et codes" pointerEvents="none">
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

            <g aria-label="Sélection" pointerEvents="none">
              {map.territories.map((territory) => {
                if (territory.id !== selectedId) {
                  return null
                }

                return (
                  <path
                    key={territory.id}
                    d={pointsToPath(territory.points)}
                    fill="none"
                    stroke="#d28b22"
                    strokeWidth="4"
                    vectorEffect="non-scaling-stroke"
                  />
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
            </div>
            <p className="border-t border-[#b7a786]/60 pt-2 leading-relaxed">
              Trait pointillé = adjacence franchissable
            </p>
            <p className="leading-relaxed">Territoire assombri = données anciennes</p>
          </CardContent>
        </Card>
      </div>
    </TooltipProvider>
  )
}
