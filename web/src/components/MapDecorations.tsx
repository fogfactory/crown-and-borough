import { memo, useMemo } from 'react'

import type { ImpassableBorderKind } from '@/lib/map-borders'
import { terrainSymbolPlacements } from '@/lib/terrain-engraving'
import type { Terrain, Territory } from '@/types'
import { MAP_HALO, MAP_INK, MAP_PAPER_FILL } from '@/lib/map-palette'

/** Stroke of the engraved symbols, in map units before annotation scale. */
const ENGRAVING_STROKE = 0.65

/**
 * Engraved terrain symbols, centred on the origin and lit from the top left
 * like a copperplate map: the right half of each relief carries the shade.
 */
export const TERRAIN_SYMBOL_PATHS: Record<
  Terrain,
  { ink: string; shade?: string; shadeOpacity?: number; fill?: boolean; opacity?: number }
> = {
  forest: {
    ink: 'M0 0.2 L0 3.2 M-2.7 -2.3 A2.7 2.7 0 1 1 2.7 -2.3 A2.7 2.7 0 1 1 -2.7 -2.3 Z',
    shade: 'M0 -5 A2.7 2.7 0 0 1 0 0.4 Z',
    shadeOpacity: 0.35,
    fill: true,
  },
  hill: {
    ink: 'M-5.2 1.8 Q0 -4.6 5.2 1.8 M1.4 -0.9 L2.3 1.6 M3.1 0.1 L3.7 1.6',
  },
  mountain: {
    ink: 'M-5 3.2 L0 -4.6 L5 3.2 Z',
    shade: 'M0 -4.6 L5 3.2 L0.9 3.2 Z',
    shadeOpacity: 0.4,
    fill: true,
  },
  swamp: {
    ink: 'M-5.4 2 L-1.8 2 M0.4 2 L4.4 2 M-3.2 3.8 L1.6 3.8 M-0.6 1.2 L-1.9 -2.6 M-0.6 1.2 L-0.6 -3.6 M-0.6 1.2 L0.7 -2.6',
  },
  plain: {
    ink: 'M-1.5 1.2 Q-1.5 -0.4 -2.6 -1.7 M0 1.2 L0 -2.4 M1.5 1.2 Q1.5 -0.4 2.6 -1.7',
    opacity: 0.7,
  },
}

export function terrainSymbolId(terrain: Terrain): string {
  return `terrain-symbol-${terrain}`
}

/** The ink strokes of one engraved terrain symbol, around the origin. */
export function TerrainGlyph({ terrain }: { terrain: Terrain }) {
  const symbol = TERRAIN_SYMBOL_PATHS[terrain]
  return (
    <g opacity={symbol.opacity}>
      <path
        d={symbol.ink}
        fill={symbol.fill ? MAP_PAPER_FILL : 'none'}
        stroke={MAP_INK}
        strokeWidth={ENGRAVING_STROKE}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {symbol.shade && (
        <path d={symbol.shade} fill={MAP_INK} fillOpacity={symbol.shadeOpacity} />
      )}
    </g>
  )
}

/** One engraved terrain symbol, drawn once in the defs and reused by `<use>`. */
export function TerrainSymbol({ terrain }: { terrain: Terrain }) {
  return (
    <symbol id={terrainSymbolId(terrain)} overflow="visible">
      <TerrainGlyph terrain={terrain} />
    </symbol>
  )
}

/**
 * Engraved terrain: every territory scatters the ink symbol of its terrain
 * over the common paper. Memoized so panning and zooming, which re-render
 * the map, never rebuild its thousands of `<use>` nodes.
 */
export const EngravedTerrainLayer = memo(function EngravedTerrainLayer({
  territories,
  annotationScale,
  label,
}: {
  territories: Territory[]
  annotationScale: number
  label: string
}) {
  const symbols = useMemo(
    () =>
      territories.map((territory) => ({
        territory,
        placements: terrainSymbolPlacements(territory, annotationScale),
      })),
    [territories, annotationScale],
  )

  return (
    <g aria-label={label} pointerEvents="none">
      {symbols.map(({ territory, placements }) => (
        <g key={territory.id} data-terrain-symbols={territory.terrain}>
          {placements.map((placement, index) => (
            <use
              key={index}
              href={`#${terrainSymbolId(territory.terrain)}`}
              transform={`translate(${placement.x.toFixed(1)} ${placement.y.toFixed(1)}) scale(${(
                placement.scale * annotationScale
              ).toFixed(3)})`}
            />
          ))}
        </g>
      ))}
    </g>
  )
})

/**
 * Border icons drawn in a 16-unit box centred on the origin. The frontier
 * runs along x; a cliff faces +y, towards the lower territory.
 */
const BORDER_SYMBOL_PATHS: Record<
  Exclude<ImpassableBorderKind, 'peaks'>,
  { stroke?: string; fill?: string }
> = {
  river: {
    stroke: 'M-7.5 0 Q-5.6 -2.2 -3.75 0 T0 0 T3.75 0 T7.5 0',
  },
  marsh: {
    stroke:
      'M-7 3 L-1.6 3 M1 3 L7 3 M-4.4 5.6 L4.6 5.6 M-4.4 1.6 L-6 -3.6 M-4.4 1.6 L-4.4 -5 M-4.4 1.6 L-2.8 -3.6 M0 1.6 L-1.6 -4.2 M0 1.6 L0 -5.8 M0 1.6 L1.6 -4.2 M4.4 1.6 L2.8 -3.6 M4.4 1.6 L4.4 -5 M4.4 1.6 L6 -3.6',
  },
  cliff: {
    stroke: 'M-7.5 -2.5 L7.5 -2.5',
    fill: 'M-7.2 -2.5 L-6 3.4 L-4.8 -2.5 Z M-3.6 -2.5 L-2.4 4.6 L-1.2 -2.5 Z M0 -2.5 L1.2 3.8 L2.4 -2.5 Z M3.6 -2.5 L4.8 4.8 L6 -2.5 Z',
  },
}

export const BORDER_SYMBOL_BOX = 16

export function borderSymbolId(kind: ImpassableBorderKind): string {
  return `border-symbol-${kind}`
}

/**
 * One impassable-border icon with its paper halo, so it stays legible over
 * the terrain symbols. The river is the only colored series.
 */
export function BorderGlyph({
  kind,
  color,
}: {
  kind: Exclude<ImpassableBorderKind, 'peaks'>
  color: string
}) {
  const { stroke, fill } = BORDER_SYMBOL_PATHS[kind]
  const width = kind === 'river' ? 1.8 : 1.1
  return (
    <g strokeLinecap="round" strokeLinejoin="round">
      {stroke && (
        <path d={stroke} fill="none" stroke={MAP_HALO} strokeWidth={width + 2.4} />
      )}
      {fill && <path d={fill} fill={MAP_HALO} stroke={MAP_HALO} strokeWidth={2.4} />}
      {stroke && <path d={stroke} fill="none" stroke={color} strokeWidth={width} />}
      {fill && <path d={fill} fill={color} />}
    </g>
  )
}

export function BorderSymbol({
  kind,
  color,
}: {
  kind: Exclude<ImpassableBorderKind, 'peaks'>
  color: string
}) {
  return (
    <symbol id={borderSymbolId(kind)} overflow="visible">
      <BorderGlyph kind={kind} color={color} />
    </symbol>
  )
}

export const WINTER_SNOW_TILE = 30

/** Hand-placed scatter so the tiling never reads as a grid. */
export const WINTER_SNOW_FLAKES = [
  { x: 7.5, y: 8.5, radius: 2.1, rotation: 0 },
  { x: 22.5, y: 5.5, radius: 1.6, rotation: 24 },
  { x: 15, y: 21.5, radius: 1.8, rotation: 51 },
]

export const WINTER_SNOW_VARIANT_ROTATIONS = [0, 23, 41]

export function snowPatternVariant(id: string): number {
  let hash = 0
  for (let i = 0; i < id.length; i += 1) {
    hash = (hash * 31 + id.charCodeAt(i)) >>> 0
  }
  return hash % WINTER_SNOW_VARIANT_ROTATIONS.length
}

export function Snowflake({
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
