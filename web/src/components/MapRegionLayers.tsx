import { useLanguage } from '@/i18n/LanguageContext'
import { formatCardCode, formatCardLabel } from '@/lib/card-hand'
import { centroid, regionRingPath } from '@/lib/map-svg-geometry'
import type { RegionStyle } from '@/lib/region-color'
import { fitLabelFontSize, type RegionOutlineResult } from '@/lib/region-geometry'
import type { RegionBand } from '@/lib/region-layout'
import type { ActiveRegionEffect, CardKind, Point, Region, Territory } from '@/types'

/** Width of the gradient liseré hugging each region boundary, in map units. */
const REGION_BORDER_WIDTH = 10
/** Stepped falloff of the regional liseré: inset fractions and opacities. */
const REGION_BORDER_STEPS = [
  { fraction: 1, opacity: 0.18 },
  { fraction: 0.55, opacity: 0.42 },
  { fraction: 0.32, opacity: 0.72 },
  { fraction: 0.18, opacity: 1 },
]

const CALAMITY_KINDS: CardKind[] = ['plague', 'bad_weather', 'famine']

/** Colored frame slices around the map, each carrying its region's name. */
export function RegionBands({
  bands,
  regionStyleByID,
  regionsById,
  territories,
}: {
  bands: RegionBand[]
  regionStyleByID: Map<string, RegionStyle>
  regionsById: Map<string, Region>
  territories: Territory[]
}) {
  const { t } = useLanguage()
  if (bands.length === 0) return null

  return (
    <g
      aria-label={t('map.regionBands')}
      pointerEvents="none"
      clipPath="url(#region-frame-clip)"
    >
      {bands.map(({ regionId, piecePath }) => (
        <path
          key={`region-band-${regionId}-${bands.length}`}
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
      {bands.map(({ regionId, labelX, labelY, labelAngle, labelWidth }) => {
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
      })}
    </g>
  )
}

/** Stepped gradient liseré along the inside of each region boundary. */
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
      {[...(outlines?.outlines.entries() ?? [])].map(([regionId, outline]) => {
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
      })}
    </g>
  )
}

/** Name badges for regions that never reach the frame (no band to carry it). */
export function RegionBadges({
  regions,
  interiorRegionIDs,
  territories,
  regionStyleByID,
  annotationScale,
}: {
  regions: Region[]
  interiorRegionIDs: Set<string>
  territories: Territory[]
  regionStyleByID: Map<string, RegionStyle>
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
  )
}
