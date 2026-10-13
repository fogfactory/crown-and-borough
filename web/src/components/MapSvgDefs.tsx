import {
  BorderSymbol,
  Snowflake,
  TERRAIN_SYMBOL_PATHS,
  TerrainSymbol,
  WINTER_SNOW_FLAKES,
  WINTER_SNOW_TILE,
  WINTER_SNOW_VARIANT_ROTATIONS,
} from '@/components/MapDecorations'
import { pointsToPath } from '@/lib/map-svg-geometry'
import type { Terrain, Territory } from '@/types'
import { INTENT_OUTLINE_COLOR, MAP_INK, RIVER_BLUE, SNOW_STROKE } from '@/lib/map-palette'

/**
 * Shared SVG definitions referenced by the map layers: per-territory clip
 * paths, the supply hatch, engraved terrain and border symbols, snow
 * patterns, the region frame clip and the intention arrow/circle markers.
 */
export function MapSvgDefs({
  territories,
  annotationScale,
  bandMarginX,
  bandMarginY,
  viewWidth,
  viewHeight,
  intentionsColor,
}: {
  territories: Territory[]
  annotationScale: number
  bandMarginX: number
  bandMarginY: number
  viewWidth: number
  viewHeight: number
  intentionsColor: string
}) {
  return (
    <defs>
      {territories.map((territory) => (
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
          stroke={MAP_INK}
          strokeWidth={1.4 * annotationScale}
          strokeOpacity="0.35"
        />
      </pattern>
      {(Object.keys(TERRAIN_SYMBOL_PATHS) as Terrain[]).map((terrain) => (
        <TerrainSymbol key={terrain} terrain={terrain} />
      ))}
      <BorderSymbol kind="marsh" color={MAP_INK} />
      <BorderSymbol kind="cliff" color={MAP_INK} />
      <BorderSymbol kind="river" color={RIVER_BLUE} />
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
            stroke={SNOW_STROKE}
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
        <rect x={-bandMarginX} y={-bandMarginY} width={viewWidth} height={viewHeight} />
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
  )
}
