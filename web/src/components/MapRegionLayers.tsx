import { useLanguage } from '@/i18n/LanguageContext'
import { formatCardCode, formatCardLabel } from '@/lib/card-hand'
import { centroid, regionRingPath } from '@/lib/map-svg-geometry'
import type { RegionStyle } from '@/lib/region-color'
import { fitLabelFontSize, type RegionOutlineResult } from '@/lib/region-geometry'
import { REGION_BAND_MARGIN, type RegionBand } from '@/lib/region-layout'
import type { ActiveRegionEffect, CardKind, Point, Region, Territory } from '@/types'
import { DEFAULT_REGION_COLOR, FRAME_RULE, MAP_INK, MAP_PAPER } from '@/lib/map-palette'

/** Width of the liseré hugging each region boundary, in map units. */
const REGION_BORDER_WIDTH = 10
const REGION_BORDER_OPACITY = 0.5
/** Inset of the brass rule inside the frame's outer ink rule, in map units. */
const FRAME_RULE_INSET = 3.5

const CALAMITY_KINDS: CardKind[] = ['plague', 'bad_weather', 'famine']

/** A bishopric's name in a paper cartouche with an ink rule. */
function RegionCartouche({
  label,
  fontSize,
  transform,
  regionId,
}: {
  label: string
  fontSize: number
  transform: string
  regionId: string
}) {
  const letterSpacing = fontSize * 0.12
  const width = label.length * (fontSize * 0.74 + letterSpacing) + fontSize * 1.4
  const height = fontSize * 1.75
  return (
    <g transform={transform} data-region-label={regionId}>
      <title>{label}</title>
      <rect
        x={-width / 2}
        y={-height / 2}
        width={width}
        height={height}
        fill={MAP_PAPER}
        stroke={MAP_INK}
        strokeWidth="1"
        vectorEffect="non-scaling-stroke"
      />
      <text
        fill={MAP_INK}
        fontFamily="var(--font-display)"
        fontSize={fontSize}
        fontWeight="700"
        letterSpacing={letterSpacing}
        textAnchor="middle"
        dominantBaseline="central"
      >
        {label.toUpperCase()}
      </text>
    </g>
  )
}

/**
 * Frame slices around the map in each bishopric's tint, between an outer ink
 * rule and an inner brass rule, each carrying its name in a cartouche.
 */
export function RegionBands({
  bands,
  regionStyleByID,
  regionsById,
  territories,
  mapWidth,
  mapHeight,
}: {
  bands: RegionBand[]
  regionStyleByID: Map<string, RegionStyle>
  regionsById: Map<string, Region>
  territories: Territory[]
  mapWidth: number
  mapHeight: number
}) {
  const { t } = useLanguage()
  if (bands.length === 0) return null

  const margin = REGION_BAND_MARGIN
  return (
    <g
      aria-label={t('map.regionBands')}
      pointerEvents="none"
      clipPath="url(#region-frame-clip)"
    >
      {bands.map(({ regionId, piecePath }, bandIndex) => (
        <path
          key={`region-band-${regionId}-${bandIndex}`}
          data-region-band={regionId}
          d={piecePath}
          fill={regionStyleByID.get(regionId)?.fill ?? DEFAULT_REGION_COLOR}
          stroke={MAP_INK}
          strokeWidth="1"
          vectorEffect="non-scaling-stroke"
        />
      ))}
      <rect
        x={-margin}
        y={-margin}
        width={mapWidth + margin * 2}
        height={mapHeight + margin * 2}
        fill="none"
        stroke={MAP_INK}
        strokeWidth="3"
        vectorEffect="non-scaling-stroke"
      />
      <rect
        x={-margin + FRAME_RULE_INSET}
        y={-margin + FRAME_RULE_INSET}
        width={mapWidth + (margin - FRAME_RULE_INSET) * 2}
        height={mapHeight + (margin - FRAME_RULE_INSET) * 2}
        fill="none"
        stroke={FRAME_RULE}
        strokeWidth="1.2"
        vectorEffect="non-scaling-stroke"
      />
      {bands.map(({ regionId, labelX, labelY, labelAngle, labelWidth }, bandIndex) => {
        const region = regionsById.get(regionId)
        const seedTerritory = region
          ? territories.find((territory) => territory.id === region.seed)
          : undefined
        if (!region || !seedTerritory) return null
        const label = t('map.regionLabel', {
          name: seedTerritory.name,
          seed: region.seed,
        })
        return (
          <RegionCartouche
            key={`region-band-label-${regionId}-${bandIndex}`}
            regionId={regionId}
            label={label}
            fontSize={Math.min(9, fitLabelFontSize(label, labelWidth * 0.62))}
            transform={`translate(${labelX} ${labelY}) rotate(${labelAngle})`}
          />
        )
      })}
    </g>
  )
}

/** Flat liseré in the bishopric's tint along the inside of its boundary. */
export function RegionRings({
  outlines,
  regionStyleByID,
  regionHoleTesters,
  annotationScale,
}: {
  outlines: RegionOutlineResult | null
  regionStyleByID: Map<string, RegionStyle>
  regionHoleTesters: Map<string, (loop: Point[]) => boolean>
  annotationScale: number
}) {
  const { t } = useLanguage()

  return (
    <g aria-label={t('map.regions')} pointerEvents="none">
      {[...(outlines?.outlines.entries() ?? [])].map(([regionId, outline]) => (
        <path
          key={`region-ring-${regionId}`}
          data-region-ring={regionId}
          d={regionRingPath(
            outline.loops,
            REGION_BORDER_WIDTH * annotationScale,
            regionHoleTesters.get(regionId) ?? (() => false),
          )}
          fill={regionStyleByID.get(regionId)?.fill ?? DEFAULT_REGION_COLOR}
          fillOpacity={REGION_BORDER_OPACITY}
          fillRule="evenodd"
          stroke="none"
        />
      ))}
    </g>
  )
}

/** Name cartouches for regions that never reach the frame (no band to carry it). */
export function RegionBadges({
  regions,
  interiorRegionIDs,
  territories,
  annotationScale,
}: {
  regions: Region[]
  interiorRegionIDs: Set<string>
  territories: Territory[]
  annotationScale: number
}) {
  const { t } = useLanguage()

  return (
    <g aria-label={t('map.regionBadges')} pointerEvents="none">
      {regions
        .filter((region) => interiorRegionIDs.has(region.id))
        .map((region) => {
          const seedTerritory = territories.find(
            (territory) => territory.id === region.seed,
          )
          if (!seedTerritory) return null
          const [centerX, centerY] = centroid(seedTerritory.points)
          return (
            <RegionCartouche
              key={`region-badge-${region.id}`}
              regionId={region.id}
              label={t('map.regionLabel', {
                name: seedTerritory.name,
                seed: region.seed,
              })}
              fontSize={7 * annotationScale}
              transform={`translate(${centerX} ${centerY - 56 * annotationScale})`}
            />
          )
        })}
    </g>
  )
}

/** Card-code roundels for the active regional effects, next to the seed. */
export function RegionEffectMarkers({
  effects,
  regions,
  territories,
  annotationScale,
}: {
  effects: ActiveRegionEffect[] | undefined
  regions: Region[] | undefined
  territories: Territory[]
  annotationScale: number
}) {
  const { t } = useLanguage()

  return (
    <g aria-label={t('map.regionEffects')} pointerEvents="none">
      {effects?.map((effect, index) => {
        const region = regions?.find((candidate) => candidate.seed === effect.regionSeed)
        const seedTerritory = territories.find(
          (territory) => territory.id === effect.regionSeed,
        )
        if (!region || !seedTerritory) return null
        const [centerX, centerY] = centroid(seedTerritory.points)
        const offset =
          effects
            ?.slice(0, index)
            .filter((candidate) => candidate.regionSeed === effect.regionSeed).length ?? 0
        const isCalamity = CALAMITY_KINDS.includes(effect.kind)
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
            {/* Ink and paper only: a calamity's roundel is doubled. */}
            <circle
              r={10 * annotationScale}
              fill={MAP_PAPER}
              stroke={MAP_INK}
              strokeWidth={isCalamity ? 3 : 1.5}
              vectorEffect="non-scaling-stroke"
            />
            {isCalamity && (
              <circle
                r={7 * annotationScale}
                fill="none"
                stroke={MAP_INK}
                strokeWidth="1"
                vectorEffect="non-scaling-stroke"
              />
            )}
            <text
              y={4 * annotationScale}
              fill={MAP_INK}
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
  )
}
