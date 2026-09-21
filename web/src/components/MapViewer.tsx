import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import { IconFocus2, IconMinus, IconPlus } from '@tabler/icons-react'

import { MapLegend, TERRAIN_COLORS, TERRAIN_LABEL_KEYS } from '@/components/MapLegend'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { Intention } from '@/lib/intent-overlay'
import {
  CALAMITY_ICONS,
  CANCELED_KIND_BY_CARD,
  CARD_ICONS,
  parseSpecialOrderPlacements,
} from '@/lib/game-icons'
import {
  chaoticIconPlacements,
  territoryRadius,
  type IconPlacement,
} from '@/lib/chaotic-icons'
import { NEUTRAL_PLAYER_ID } from '@/types'
import {
  DRAG_THRESHOLD,
  WHEEL_ZOOM_FACTOR,
  ZOOM_BUTTON_FACTOR,
  clientToMapPoint,
  distanceBetween,
  midpointOf,
  pinchView,
  viewportScale,
  zoomAtCenter,
  zoomAtPoint,
  type MapPoint,
  type ViewState,
} from '@/lib/map-gestures'
import {
  REGION_PATTERNS,
  regionStyle,
  type RegionPattern,
  type RegionStyle,
} from '@/lib/region-color'
import { formatCardCode, formatCardLabel } from '@/lib/card-hand'
import { hasSupplySource } from '@/lib/supply'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type {
  Infrastructure,
  CardKind,
  MapData,
  Noble,
  PlayerId,
  Point,
  StateData,
  SupplyLine,
  Terrain,
  Territory,
} from '@/types'

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

const PLAYER_PALETTE = ['#a84632', '#2d5f9e', '#7052a1', '#0e7490', '#ad7a25']
const INTENT_OUTLINE_COLOR = '#17120f'
export const DRAFT_INTENTION_COLOR = '#d4a39b'
const CALAMITY_KINDS: CardKind[] = ['plague', 'bad_weather', 'famine']

// Map marker artwork from game-icons.net (CC BY 3.0, icons by Delapouite):
// https://game-icons.net/1x1/delapouite/castle.html
// https://game-icons.net/1x1/delapouite/village.html
// https://game-icons.net/1x1/delapouite/windmill.html
// https://game-icons.net/1x1/delapouite/barn.html
// Filled 512x512 silhouettes, scaled down to the marker footprint.
const MARKER_GLYPHS: Record<Infrastructure['type'], string> = {
  castle:
    'M255.95 27.11L180.6 107.614l150.7 1.168-75.35-81.674h-.003zM25 109.895v68.01l19.412 25.99h71.06l19.528-26v-68h-14v15.995h-18v-15.994H89v15.995H71v-15.994H57v15.995H39v-15.994H25zm352 0v68l19.527 26h71.06L487 177.906v-68.01h-14v15.995h-18v-15.994h-14v15.995h-18v-15.994h-14v15.995h-18v-15.994h-14zm-176 15.877V260.89h110V126.63l-110-.857zm55 20.118c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zM41 221.897V484.89h78V221.897H41zm352 0V484.89h78V221.897h-78zM56 241.89c4 0 8 4 8 12v32H48v-32c0-8 4-12 8-12zm400 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12zm-303 37v23h-16v183h87v-55c0-24 16-36 32-36s32 12 32 36v55h87v-183h-16v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14zm-49 43c4 0 8 4 8 12v32H96v-32c0-8 4-12 8-12zm72 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm80 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm80 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm72 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12zm-352 64c4 0 8 4 8 12v32H48v-32c0-8 4-12 8-12zm400 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12z',
  village:
    'M109.902 35.87l-71.14 59.284h142.28l-71.14-59.285zm288 32l-71.14 59.284h142.28l-71.14-59.285zM228.73 84.403l-108.9 90.75h217.8l-108.9-90.75zm-173.828 28.75v62h36.81l73.19-60.992v-1.008h-110zm23 14h16v18h-16v-18zm265 18v10.963l23 19.166v-16.13h16v18h-13.756l.104.087 19.098 15.914h-44.446v14h78v-39h18v39h14v-62h-110zm-194.345 48v20.08l24.095-20.08h-24.095zm28.158 0l105.1 87.582 27.087-22.574v-65.008H176.715zm74.683 14h35.735v34h-35.735v-34zm-76.714 7.74L30.37 335.153H319l-144.314-120.26zm198.046 13.51l-76.857 64.047 32.043 26.704H481.63l-108.9-90.75zm-23.214 108.75l.103.086 19.095 15.914h-72.248v77.467h60.435v-63.466h50v63.467h46v-93.466H349.516zm-278.614 16V476.13h126v-76.976h50v76.977h31.565V353.155H70.902zm30 30h50v50h-50v-50z',
  mill: 'M161.188 22L102.25 41.656 230.063 169.47l.843-.845 2.406-2.406L161.188 22zm246.906 18L280.28 167.813l.814.812 2.406 2.406 144.25-72.124L408.094 40zM256 40.938l-53.97 26.968 45.657 91.344c2.727-.648 5.52-.97 8.313-.97 3.306 0 6.614.467 9.813 1.376l75.875-75.875L256 40.938zm-88 89.093V184h53.906c.006-.02-.005-.043 0-.063L168 130.03zm176 28.657L293.375 184H344v-25.313zm-88 15.5c-4.975 0-9.94 1.908-13.78 5.75-7.686 7.685-7.686 19.91 0 27.594 7.683 7.686 19.877 7.686 27.56 0 7.686-7.683 7.686-19.908 0-27.593-3.84-3.842-8.805-5.75-13.78-5.75zM199.312 201l-2.875 13.594 25.094-12.563c-.08-.345-.146-.682-.218-1.03h-22zm91.375 0c-.176.856-.353 1.72-.593 2.563l29.312 29.312-6.72-31.875H290.69zM228.5 216.47L84.25 288.562l19.656 58.937L231.72 219.687l-.814-.843-2.406-2.375zm53.438 1.56l-.844.814-2.375 2.406L350.81 365.5l58.938-19.656L281.937 218.03zm-35.75 9.814l-66.532 66.53L139.094 487H216v-63h80v63h76.906l-22.03-104.688-1.595.532-6.56 2.22-3.126-6.22-75.28-150.625c-5.956 1.416-12.227 1.302-18.127-.376z',
  supply_depot:
    'M256 23.38L89.844 89.845l-64.9 162.254 14.85 5.943c20.312-50.766 40.62-101.535 60.93-152.304l1.432-3.58L256 40.616l153.844 61.54 1.43 3.58 60.93 152.305 14.853-5.942-64.9-162.254C366.77 67.69 311.386 45.534 256 23.38zm0 36.624l-139.996 55.998L72.8 224h.2v263h78V329h-39v-18h297v176h30V224h.2c-14.402-36-28.802-72-43.204-107.998L256 60.004zM151 135h210v114H151V135zm23.563 18L199 201.873V153h-24.438zM313 153v48.873L337.438 153H313zm-144 29.127V231h24.438L169 182.127zm174 0L318.562 231H343v-48.873zm-98.73 18.69c-1.207-.02-2.31.02-3.288.128-2.823.31-10.76 3.708-16.86 7.3-2.796 1.645-5.23 3.22-7.122 4.484V231h78v-16.97c-4.193-1.675-10.334-4.02-17.578-6.368-11.206-3.63-24.71-6.71-33.152-6.846zM160 263h192v18H160v-18zm15.16 66L208 389.205 240.84 329h-65.68zm144 0L352 389.205 384.84 329h-65.68zM169 355.295v105.41L197.748 408 169 355.295zm78 0L218.252 408 247 460.705v-105.41zm66 0v105.41L341.748 408 313 355.295zm78 0L362.252 408 391 460.705v-105.41zm-183 71.5L175.16 487h65.68L208 426.795zm144 0L319.16 487h65.68L352 426.795z',
}

/** Capital crown stays a light stroke glyph (Tabler Icons, MIT). */
const CROWN_PATH = 'M12 6l4 6l5 -4l-2 10h-14l-2 -10l5 4l4 -6'

/** Neutral fill/stroke for infrastructure without a controlling player. */
const NEUTRAL_MARKER_FILL = '#efe6d0'
const MARKER_CASING_COLOR = '#30291f'
/** Rebel armies answer to no crown: they render in a neutral gray. */
const NEUTRAL_ARMY_COLOR = '#6b7280'

/**
 * Cartographic texture per terrain: a repeating symbol in a darker shade of
 * the terrain fill, so terrain stays identifiable beyond color alone.
 */
interface TerrainPatternSpec {
  terrain: Terrain
  /** Repeating tile size in map units (before annotation scale). */
  size: number
  strokeWidth?: number
  render: (scale: number) => ReactNode
}

const TERRAIN_PATTERNS: TerrainPatternSpec[] = [
  {
    terrain: 'plain',
    size: 14,
    strokeWidth: 1,
    // Tiny field-furrow slashes (///), generously spaced.
    render: (scale) => (
      <>
        <line x1={4.8 * scale} y1={7.6 * scale} x2={6 * scale} y2={6.4 * scale} />
        <line x1={6.4 * scale} y1={7.6 * scale} x2={7.6 * scale} y2={6.4 * scale} />
        <line x1={8 * scale} y1={7.6 * scale} x2={9.2 * scale} y2={6.4 * scale} />
      </>
    ),
  },
  {
    terrain: 'forest',
    size: 20,
    render: (scale) => (
      <>
        <path d={`M0 ${6.8 * scale} L${4 * scale} 0 L${8 * scale} ${6.8 * scale} Z`} />
        <path
          d={`M${10 * scale} ${16.8 * scale} L${14 * scale} ${10 * scale} L${18 * scale} ${16.8 * scale} Z`}
        />
      </>
    ),
  },
  {
    terrain: 'hill',
    size: 20,
    strokeWidth: 2.2,
    render: (scale) => (
      <path
        d={`M0 ${6 * scale} Q ${5 * scale} ${1.2 * scale} ${10 * scale} ${6 * scale} T ${20 * scale} ${6 * scale}`}
        fill="none"
      />
    ),
  },
  {
    terrain: 'mountain',
    size: 18,
    strokeWidth: 2.4,
    render: (scale) => (
      <path
        d={`M0 ${7.2 * scale} L${4.4 * scale} ${1.6 * scale} L${8.8 * scale} ${7.2 * scale} M${9 * scale} ${16.4 * scale} L${13.4 * scale} ${10.8 * scale} L${17.8 * scale} ${16.4 * scale}`}
        fill="none"
      />
    ),
  },
  {
    terrain: 'swamp',
    size: 10,
    render: (scale) => (
      <>
        <line x1={0.8 * scale} y1={2.6 * scale} x2={4.4 * scale} y2={2.6 * scale} />
        <line x1={5.6 * scale} y1={7 * scale} x2={9.2 * scale} y2={7 * scale} />
      </>
    ),
  },
]

const TERRAIN_PATTERN_STROKES: Record<Terrain, string> = {
  plain: '#5a7a34',
  forest: '#14291d',
  hill: '#6b4a30',
  mountain: '#4d565e',
  swamp: '#2e5f5a',
}

function TerrainPattern({
  terrain,
  scale,
}: {
  terrain: TerrainPatternSpec
  scale: number
}) {
  return (
    <pattern
      id={`terrain-${terrain.terrain}`}
      width={terrain.size * scale}
      height={terrain.size * scale}
      patternUnits="userSpaceOnUse"
    >
      <g
        stroke={TERRAIN_PATTERN_STROKES[terrain.terrain]}
        strokeWidth={(terrain.strokeWidth ?? 1.3) * scale}
        strokeOpacity={0.22}
        fill={terrain.terrain === 'forest' ? '#14291d' : 'none'}
        fillOpacity={0.18}
      >
        {terrain.render(scale)}
      </g>
    </pattern>
  )
}

const WINTER_SNOW_TILE = 30

/** Hand-placed scatter so the tiling never reads as a grid. */
const WINTER_SNOW_FLAKES = [
  { x: 7.5, y: 8.5, radius: 2.1, rotation: 0 },
  { x: 22.5, y: 5.5, radius: 1.6, rotation: 24 },
  { x: 15, y: 21.5, radius: 1.8, rotation: 51 },
]

const WINTER_SNOW_VARIANT_ROTATIONS = [0, 23, 41]

function snowPatternVariant(id: string): number {
  let hash = 0
  for (let i = 0; i < id.length; i += 1) {
    hash = (hash * 31 + id.charCodeAt(i)) >>> 0
  }
  return hash % WINTER_SNOW_VARIANT_ROTATIONS.length
}

function Snowflake({
  flake,
  scale,
}: {
  flake: (typeof WINTER_SNOW_FLAKES)[number]
  scale: number
}) {
  const dx = 0.866 * flake.radius * scale
  const dy = 0.5 * flake.radius * scale
  return (
    <g
      transform={`translate(${flake.x * scale} ${flake.y * scale}) rotate(${flake.rotation})`}
    >
      <line x1={0} y1={-flake.radius * scale} x2={0} y2={flake.radius * scale} />
      <line x1={-dx} y1={-dy} x2={dx} y2={dy} />
      <line x1={dx} y1={-dy} x2={-dx} y2={dy} />
    </g>
  )
}

const INFRASTRUCTURE_LABEL_KEYS: Record<Infrastructure['type'], MessageKey> = {
  mill: 'infrastructure.mill',
  supply_depot: 'infrastructure.supply_depot',
  castle: 'infrastructure.castle',
  village: 'infrastructure.village',
}

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

interface InfrastructureMarkerProps {
  infrastructure: Infrastructure
  x: number
  y: number
  isCapital: boolean
  scale: number
  ownerColor?: string | null
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

function clientToSvgPoint(
  svg: SVGSVGElement,
  clientX: number,
  clientY: number,
  mapWidth: number,
  mapHeight: number,
): Point {
  const bounds = svg.getBoundingClientRect()
  return clientToMapPoint(clientX, clientY, mapWidth, mapHeight, bounds)
}

function getTerritoryIdFromTarget(target: EventTarget | null): string | null {
  if (!(target instanceof Element)) {
    return null
  }

  return (
    target.closest<SVGPathElement>('[data-territory-id]')?.dataset.territoryId ?? null
  )
}

function TablerMarkerPaths({ paths, fill = 'none' }: { paths: string[]; fill?: string }) {
  return (
    <>
      {paths.map((d) => (
        <path key={d} d={d} fill={fill} strokeLinecap="round" strokeLinejoin="round" />
      ))}
    </>
  )
}

function InfrastructureMarker({
  infrastructure,
  x,
  y,
  isCapital,
  scale,
  ownerColor = null,
}: InfrastructureMarkerProps) {
  const { t } = useLanguage()
  const label = `${t(INFRASTRUCTURE_LABEL_KEYS[infrastructure.type])} · ${t('app.level', { level: infrastructure.level })}${isCapital ? ` · ${t('app.capital')}` : ''}`
  const glyph = MARKER_GLYPHS[infrastructure.type]
  const fill = ownerColor ?? NEUTRAL_MARKER_FILL

  return (
    <g transform={`translate(${x} ${y}) scale(${scale})`} pointerEvents="none">
      <title>{label}</title>
      <g transform="translate(-13 -13) scale(0.05078125)">
        {/* Light halo keeps the glyph readable on any terrain fill. */}
        <path d={glyph} fill="#fff8e7" stroke="#fff8e7" strokeWidth={20} opacity={0.9} />
        {/* Owner color fills the building, dark casing defines its shape. */}
        <path d={glyph} fill={fill} />
        <path d={glyph} fill="none" stroke={MARKER_CASING_COLOR} strokeWidth={8} />
      </g>
      {isCapital && (
        <g
          data-capital-marker="true"
          transform="translate(0 -20) scale(0.5)"
          pointerEvents="none"
        >
          <g stroke="#fff8e7" strokeWidth={4} opacity={0.85}>
            <TablerMarkerPaths paths={[CROWN_PATH]} />
          </g>
          <g stroke="#815f1e" strokeWidth={2}>
            <TablerMarkerPaths paths={[CROWN_PATH]} />
          </g>
        </g>
      )}
      {infrastructure.level > 1 && (
        <text
          x="14"
          y="-9"
          fill="#4e3828"
          fontSize="10"
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

function RegionPatternDefinition({ pattern }: { pattern: RegionPattern }) {
  const stroke = '#fffaf0'
  const strokeWidth = 1.5

  return (
    <pattern
      id={`region-pattern-${pattern}`}
      width={pattern === 'diamonds' ? 12 : 8}
      height={pattern === 'diamonds' ? 12 : 8}
      patternUnits="userSpaceOnUse"
    >
      {pattern === 'diagonal' && (
        <path d="M-2 2L2-2M0 8L8 0M6 10L10 6" stroke={stroke} strokeWidth={strokeWidth} />
      )}
      {pattern === 'vertical' && (
        <path d="M2 0V8M6 0V8" stroke={stroke} strokeWidth={strokeWidth} />
      )}
      {pattern === 'horizontal' && (
        <path d="M0 2H8M0 6H8" stroke={stroke} strokeWidth={strokeWidth} />
      )}
      {pattern === 'cross' && (
        <path d="M2 0V8M6 0V8M0 2H8M0 6H8" stroke={stroke} strokeWidth={strokeWidth} />
      )}
      {pattern === 'dots' && <circle cx="2" cy="2" r="1.2" fill={stroke} />}
      {pattern === 'diamonds' && (
        <path
          d="M6 0L12 6L6 12L0 6Z"
          fill="none"
          stroke={stroke}
          strokeWidth={strokeWidth}
        />
      )}
    </pattern>
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

function MapControls({ onZoom }: { onZoom: (zoomFactor: number) => void }) {
  const { t } = useLanguage()

  return (
    <div
      role="group"
      aria-label={t('map.zoomControls')}
      className="absolute bottom-28 right-3 z-10 flex flex-col gap-1.5 lg:bottom-3"
    >
      <button
        type="button"
        aria-label={t('map.zoomIn')}
        title={t('map.zoomIn')}
        className="flex size-9 items-center justify-center rounded-lg border border-[#b7a786] bg-[#fffaf0] text-[#594b3c] shadow-md transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
        onClick={() => onZoom(ZOOM_BUTTON_FACTOR)}
      >
        <IconPlus aria-hidden="true" className="size-4" />
      </button>
      <button
        type="button"
        aria-label={t('map.zoomOut')}
        title={t('map.zoomOut')}
        className="flex size-9 items-center justify-center rounded-lg border border-[#b7a786] bg-[#fffaf0] text-[#594b3c] shadow-md transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
        onClick={() => onZoom(1 / ZOOM_BUTTON_FACTOR)}
      >
        <IconMinus aria-hidden="true" className="size-4" />
      </button>
      <button
        type="button"
        aria-label={t('map.recenter')}
        title={t('map.recenter')}
        className="flex size-9 items-center justify-center rounded-lg border border-[#b7a786] bg-[#fffaf0] text-[#594b3c] shadow-md transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
        onClick={() => onZoom(0)}
      >
        <IconFocus2 aria-hidden="true" className="size-4" />
      </button>
    </div>
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
  onToggleIntentions?: (show: boolean) => void
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
  showIntentions = false,
  intentionsColor = '#a84632',
  onToggleIntentions,
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
      return regionStyleByID.get(regionByTerritory.get(first) ?? '')?.fill ?? '#315a75'
    } catch {
      return '#607d8b'
    }
  }

  const regionColorForTerritory = (territoryID: string) =>
    regionStyleByID.get(regionByTerritory.get(territoryID) ?? '')?.fill ?? '#315a75'

  const regionStyleForTerritory = (territoryID: string): RegionStyle | null =>
    regionStyleByID.get(regionByTerritory.get(territoryID) ?? '') ?? null

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
        : colorsByPlayer.get(owner) ?? PLAYER_PALETTE[index % PLAYER_PALETTE.length],
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

  /**
   * Regions where a drafted bonus card cancels the active calamity: BT against
   * bad weather, RA against famine. The calamity icons disappear from those
   * regions because the play clears them at the next resolution.
   */
  const canceledCalamityRegions = useMemo(() => {
    const canceled = new Set<string>()
    for (const { text } of specialOrders) {
      for (const placement of parseSpecialOrderPlacements(text)) {
        const canceledKind =
          placement.kind === 'fair_weather' || placement.kind === 'abundant_harvest'
            ? CANCELED_KIND_BY_CARD[placement.kind]
            : null
        if (canceledKind) {
          canceled.add(`${canceledKind}-${placement.target}`)
        }
      }
    }
    return canceled
  }, [specialOrders])

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
      src: string
      className: string
      opacity: number
      placement: IconPlacement
    }> = []
    for (const effect of state.activeRegionEffects ?? []) {
      if (canceledCalamityRegions.has(`${effect.kind}-${effect.regionSeed}`)) {
        continue
      }
      const style =
        CALAMITY_ICONS[effect.kind as keyof typeof CALAMITY_ICONS]
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
            src: style.src,
            className: style.className,
            opacity: style.opacity,
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
  ])

  const cardIcons = useMemo(() => {
    if (!showCards) {
      return { scatterItems: [], revoltItems: [] }
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
      src: string
      className: string
      opacity: number
      placement: IconPlacement
    }> = []
    const revoltItems: Array<{
      key: string
      territory: Territory
      playerColor: string
    }> = []
    const scatteredRegions = new Set<string>()
    for (const { player, text } of specialOrders) {
      for (const placement of parseSpecialOrderPlacements(text)) {
        if (placement.kind === 'revolt') {
          const territory = territoriesById.get(placement.target)
          if (!territory) {
            continue
          }
          revoltItems.push({
            key: `revolt-${placement.target}-${revoltItems.length}`,
            territory,
            playerColor: colorsByPlayer.get(player) ?? '#475569',
          })
          continue
        }
        // A bonus card that cancels an active calamity removes its icons
        // instead of scattering its own.
        if (
          canceledCalamityRegions.has(
            `${CANCELED_KIND_BY_CARD[placement.kind]}-${placement.target}`,
          ) &&
          activeByRegion.has(
            `${CANCELED_KIND_BY_CARD[placement.kind]}-${placement.target}`,
          )
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
              src: style.src,
              className: style.className,
              opacity: style.opacity,
              placement: placement2,
            })
          }
        }
      }
    }
    return { scatterItems, revoltItems }
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
        mapWidth,
        mapHeight,
      )
      const zoomFactor = event.deltaY < 0 ? WHEEL_ZOOM_FACTOR : 1 / WHEEL_ZOOM_FACTOR

      setView((current) => zoomAtPoint(current, cursor, zoomFactor))
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
    const bounds = event.currentTarget.getBoundingClientRect()
    const mapUnitsPerPx =
      1 / viewportScale(mapWidth, mapHeight, bounds.width || 1, bounds.height || 1)
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
      mapWidth,
      mapHeight,
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
          viewBox={`0 0 ${mapWidth} ${mapHeight}`}
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
              {REGION_PATTERNS.map((pattern) => (
                <RegionPatternDefinition key={pattern} pattern={pattern} />
              ))}
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
                {map.territories.map((territory) => {
                  const style = regionStyleForTerritory(territory.id)
                  if (!style) return null
                  return (
                    <g key={`region-fill-${territory.id}`}>
                      <path
                        data-region-fill={regionByTerritory.get(territory.id)}
                        d={pointsToPath(territory.points)}
                        fill={style.fill}
                        fillOpacity="0.30"
                        stroke="none"
                      />
                      {style.pattern && (
                        <path
                          data-region-pattern={style.pattern}
                          d={pointsToPath(territory.points)}
                          fill={`url(#region-pattern-${style.pattern})`}
                          fillOpacity="0.72"
                          stroke="none"
                        />
                      )}
                    </g>
                  )
                })}
                {outerBorders.map((border) => (
                  <g key={`region-outer-${border.key}`}>
                    <line
                      x1={border.from[0]}
                      y1={border.from[1]}
                      x2={border.to[0]}
                      y2={border.to[1]}
                      stroke="#fffaf0"
                      strokeWidth="8"
                      strokeLinecap="round"
                      vectorEffect="non-scaling-stroke"
                    />
                    <line
                      data-region-boundary="true"
                      x1={border.from[0]}
                      y1={border.from[1]}
                      x2={border.to[0]}
                      y2={border.to[1]}
                      stroke="#294c63"
                      strokeWidth="4"
                      strokeDasharray="12 7"
                      strokeLinecap="round"
                      vectorEffect="non-scaling-stroke"
                    />
                  </g>
                ))}
                {sharedBorders
                  .filter((border) => isRegionBoundary(border.key))
                  .map((border) => (
                    <g key={`region-${border.key}`}>
                      <line
                        x1={border.from[0]}
                        y1={border.from[1]}
                        x2={border.to[0]}
                        y2={border.to[1]}
                        stroke="#fffaf0"
                        strokeWidth="8"
                        strokeLinecap="round"
                        vectorEffect="non-scaling-stroke"
                      />
                      <line
                        data-region-boundary="true"
                        x1={border.from[0]}
                        y1={border.from[1]}
                        x2={border.to[0]}
                        y2={border.to[1]}
                        stroke={regionColorForBoundary(border.key)}
                        strokeWidth="4"
                        strokeDasharray="12 7"
                        strokeLinecap="round"
                        vectorEffect="non-scaling-stroke"
                      />
                    </g>
                  ))}
              </g>
            )}

            {state.season === 'winter' && (
              <g aria-label={t('map.winterOverlay')} pointerEvents="none">
                <rect width={mapWidth} height={mapHeight} fill="#eaf3ff" opacity="0.2" />
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
                      <>
                        <path
                          d={solidPath}
                          fill="none"
                          stroke={MARKER_CASING_COLOR}
                          strokeOpacity="0.55"
                          strokeWidth="11"
                          strokeLinecap="round"
                          clipPath={`url(#territory-clip-${territory.id})`}
                          vectorEffect="non-scaling-stroke"
                        />
                        <path
                          d={solidPath}
                          fill="none"
                          stroke={playerColors.get(owner) ?? '#475569'}
                          strokeWidth="8"
                          strokeLinecap="round"
                          clipPath={`url(#territory-clip-${territory.id})`}
                          vectorEffect="non-scaling-stroke"
                        />
                      </>
                    )}
                    {passablePath && (
                      <>
                        <path
                          d={passablePath}
                          fill="none"
                          stroke={MARKER_CASING_COLOR}
                          strokeOpacity="0.55"
                          strokeWidth="11"
                          strokeLinecap="round"
                          strokeDasharray={passableBorderDash}
                          clipPath={`url(#territory-clip-${territory.id})`}
                          vectorEffect="non-scaling-stroke"
                        />
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
                      </>
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

            {calamityIcons.length > 0 && (
              <g aria-label={t('map.calamityOverlay')} pointerEvents="none">
                {calamityIcons.map((icon) => (
                  <image
                    key={icon.key}
                    href={icon.src}
                    x={icon.placement.x - icon.placement.size / 2}
                    y={icon.placement.y - icon.placement.size / 2}
                    width={icon.placement.size}
                    height={icon.placement.size}
                    className={icon.className}
                    opacity={icon.opacity}
                    transform={`rotate(${(icon.placement.rotation * 180) / Math.PI} ${icon.placement.x} ${icon.placement.y})`}
                  />
                ))}
              </g>
            )}
            {(cardIcons.scatterItems.length > 0 || cardIcons.revoltItems.length > 0) && (
              <g aria-label={t('map.cardOverlay')} pointerEvents="none">
                {cardIcons.scatterItems.map((icon) => (
                  <image
                    key={icon.key}
                    href={icon.src}
                    x={icon.placement.x - icon.placement.size / 2}
                    y={icon.placement.y - icon.placement.size / 2}
                    width={icon.placement.size}
                    height={icon.placement.size}
                    className={icon.className}
                    opacity={icon.opacity}
                    transform={`rotate(${(icon.placement.rotation * 180) / Math.PI} ${icon.placement.x} ${icon.placement.y})`}
                  />
                ))}
                {cardIcons.revoltItems.map((icon) => {
                  const [centerX, centerY] = centroid(icon.territory.points)
                  const size = territoryRadius(icon.territory.points) * 0.42
                  const left = centerX - size / 2
                  const top = centerY - size / 2
                  const maskId = `revolt-mask-${icon.key}`
                  return (
                    <g key={icon.key}>
                      <defs>
                        <mask
                          id={maskId}
                          maskUnits="userSpaceOnUse"
                          x={left - size * 0.1}
                          y={top - size * 0.1}
                          width={size * 1.2}
                          height={size * 1.2}
                        >
                          <image
                            href={CARD_ICONS.revolt.src}
                            x={left}
                            y={top}
                            width={size}
                            height={size}
                            style={{ filter: 'brightness(0) invert(1)' }}
                          />
                        </mask>
                      </defs>
                      <circle
                        cx={centerX}
                        cy={centerY}
                        r={size * 0.62}
                        fill="#6b7280"
                        opacity={0.75}
                      />
                      <rect
                        x={left}
                        y={top}
                        width={size}
                        height={size}
                        fill={icon.playerColor}
                        opacity={0.95}
                        mask={`url(#${maskId})`}
                      />
                    </g>
                  )
                })}
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

            {showRegions && (map.regions?.length ?? 0) > 0 && (
              <g aria-label={t('map.regionSeeds')} pointerEvents="none">
                {map.regions?.map((region) => {
                  const seedTerritory = map.territories.find(
                    (territory) => territory.id === region.seed,
                  )
                  if (!seedTerritory) return null
                  const [centerX, centerY] = centroid(seedTerritory.points)
                  const markerY = centerY - 25 * annotationScale
                  const color = regionColorForTerritory(region.seed)
                  return (
                    <g
                      key={`region-seed-${region.seed}`}
                      data-region-seed={region.seed}
                      transform={`translate(${centerX} ${markerY})`}
                    >
                      <title>
                        {t('map.regionSeedMarker', {
                          seed: region.seed,
                          name: seedTerritory.name,
                        })}
                      </title>
                      <circle
                        r={18 * annotationScale}
                        fill="#fffaf0"
                        fillOpacity="0.96"
                        stroke="#1f3a4d"
                        strokeWidth={3 * annotationScale}
                        vectorEffect="non-scaling-stroke"
                      />
                      <path
                        d={`M 0 ${-11 * annotationScale} L ${11 * annotationScale} 0 L 0 ${11 * annotationScale} L ${-11 * annotationScale} 0 Z`}
                        fill={color}
                        stroke="#fffaf0"
                        strokeWidth={2 * annotationScale}
                        vectorEffect="non-scaling-stroke"
                      />
                      <text
                        y={31 * annotationScale}
                        fill="#1f3a4d"
                        fontSize={11 * annotationScale}
                        fontWeight="900"
                        textAnchor="middle"
                        stroke="#fffaf0"
                        strokeWidth={4 * annotationScale}
                        paintOrder="stroke"
                      >
                        {region.seed}
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

            {showIntentions && intentions.length > 0 && (
              <g aria-label={t('map.intentionsOverlay')} pointerEvents="none">
                {intentions.map((intention, index) => {
                  const isDraft = intention.source === 'draft'
                  const intentionColor =
                    intention.color ?? (isDraft ? DRAFT_INTENTION_COLOR : intentionsColor)
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

        <MapControls onZoom={handleControlZoom} />

        {(onToggleIntentions || onToggleRegions || onToggleCalamities || onToggleCards) && (
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
