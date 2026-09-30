import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { centroid } from '@/lib/map-svg-geometry'
import { fitLabelFontSize } from '@/lib/region-geometry'
import type { Fief, FiefTitle, PlayerId, Territory } from '@/types'

const FIEF_TITLE_KEYS: Record<FiefTitle, MessageKey> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

/**
 * Name tag near each fief's capital (titres.md, issue #194). The per-
 * territory ownership badges (`OwnershipLayer`) carry the capital's trigram
 * to tell fiefs apart, so this layer no longer needs a boundary outline.
 */
export function MapFiefLayer({
  territories,
  fiefs,
  playerColors,
  annotationScale,
}: {
  territories: Territory[]
  fiefs: Fief[]
  playerColors: Map<PlayerId, string>
  annotationScale: number
}) {
  const { t } = useLanguage()
  if (fiefs.length === 0) return null

  return (
    <g aria-label={t('map.fiefs')} pointerEvents="none">
      {fiefs.map((fief) => {
        const color = playerColors.get(fief.owner) ?? '#a84632'
        const capitalTerritory = territories.find(
          (territory) => territory.id === fief.capital,
        )
        const label = t('map.fiefLabel', {
          title: t(FIEF_TITLE_KEYS[fief.title]),
          capital: capitalTerritory?.name ?? fief.capital,
        })
        const [labelX, labelY] = capitalTerritory
          ? centroid(capitalTerritory.points)
          : [0, 0]
        const fontSize = fitLabelFontSize(label, 120 * annotationScale, 9.5, 6.5)
        const width = label.length * fontSize * 0.62 + 12 * annotationScale
        const height = 16 * annotationScale

        return (
          <g key={fief.capital} data-fief-capital={fief.capital}>
            <g transform={`translate(${labelX} ${labelY - 46 * annotationScale})`}>
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
                strokeWidth={1.6 * annotationScale}
                vectorEffect="non-scaling-stroke"
              />
              <text
                y={0.5 * annotationScale}
                fill={color}
                fontSize={fontSize}
                fontWeight="800"
                textAnchor="middle"
                dominantBaseline="central"
              >
                {label}
              </text>
            </g>
          </g>
        )
      })}
    </g>
  )
}
