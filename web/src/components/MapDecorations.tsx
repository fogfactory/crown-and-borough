import type { ReactNode } from 'react'

import type { Terrain } from '@/types'

/**
 * Cartographic texture per terrain: a repeating symbol in a darker shade of
 * the terrain fill, so terrain stays identifiable beyond color alone.
 */
export interface TerrainPatternSpec {
  terrain: Terrain
  /** Repeating tile size in map units (before annotation scale). */
  size: number
  strokeWidth?: number
  render: (scale: number) => ReactNode
}

export const TERRAIN_PATTERNS: TerrainPatternSpec[] = [
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

export function TerrainPattern({
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
