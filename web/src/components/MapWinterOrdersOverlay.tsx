import {
  DRAFT_INTENTION_COLOR,
  InfrastructureMarker,
  WinterBadge,
  WinterMarkerTriangle,
  WinterTransferArrow,
  winterMarkerTitle,
  winterReasonText,
} from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import { centroid } from '@/lib/map-svg-geometry'
import type { WinterIntention } from '@/lib/winter-overlay'
import type { Territory } from '@/types'

const WINTER_ERROR_COLOR = '#c43b2a'
const WINTER_WARNING_COLOR = '#e07a30'

/**
 * Winter order drafts on the map: ghost builds, recruit/noble badges,
 * transfer arrows, and warning/error triangles (unplaced errors stack in
 * the top-left corner).
 */
export function WinterOrdersOverlay({
  territories,
  winterIntentions,
  annotationScale,
}: {
  territories: Territory[]
  winterIntentions: WinterIntention[]
  annotationScale: number
}) {
  const { t } = useLanguage()

  return (
    <g
      aria-label={t('map.winterOverlay')}
      data-winter-orders-overlay="true"
      pointerEvents="none"
    >
      {territories.map((territory) => {
        const [centerX, centerY] = centroid(territory.points)
        const localIntentions = winterIntentions.filter(
          (intention) => intention.territory === territory.id,
        )
        const builds = localIntentions.filter(
          (intention) => intention.valid && intention.kind === 'build',
        )
        const badges = localIntentions.filter(
          (intention) =>
            intention.valid &&
            intention.kind !== 'build' &&
            intention.kind !== 'transfer',
        )
        const errors = localIntentions.filter((intention) => !intention.valid)

        return (
          <g key={`winter-${territory.id}`}>
            {builds.map((intention, index) => {
              const buildX = centerX + (-10 + index * 40) * annotationScale
              const buildY = centerY - 38 * annotationScale
              return (
                <g key={`build-${intention.line}`}>
                  <InfrastructureMarker
                    infrastructure={{
                      type: intention.infrastructure ?? 'mill',
                      level: intention.level ?? 1,
                    }}
                    x={buildX}
                    y={buildY}
                    isCapital={false}
                    scale={annotationScale}
                    ownerColor={intention.color ?? DRAFT_INTENTION_COLOR}
                    variant="winterGhost"
                  />
                  {intention.warning && (
                    <WinterMarkerTriangle
                      x={buildX + 26 * annotationScale}
                      y={buildY + 11 * annotationScale}
                      line={intention.line}
                      title={winterMarkerTitle(t, intention)}
                      scale={annotationScale}
                      color={WINTER_WARNING_COLOR}
                      dataTag="warning"
                    />
                  )}
                </g>
              )
            })}
            {badges.map((intention, index) => {
              const label =
                intention.kind === 'recruit_troop'
                  ? '+1'
                  : intention.kind === 'recruit_noble'
                    ? 'N'
                    : intention.kind === 'liberate'
                      ? 'L'
                      : intention.kind === 'hostage'
                        ? 'O'
                        : intention.kind === 'dungeon'
                          ? 'P'
                          : 'E'
              const detail =
                intention.kind === 'recruit_noble'
                  ? 'R'
                  : intention.kind === 'capital'
                    ? 'C'
                    : intention.noble
              const badgeY = centerY + (-18 + index * 20) * annotationScale
              return (
                <g key={`${intention.kind}-${intention.line}`}>
                  <WinterBadge
                    x={centerX + 14 * annotationScale}
                    y={badgeY}
                    label={label}
                    detail={detail}
                    title={intention.label}
                    color={intention.color ?? DRAFT_INTENTION_COLOR}
                    scale={annotationScale}
                  />
                  {intention.warning && (
                    <WinterMarkerTriangle
                      x={centerX - 8 * annotationScale}
                      y={badgeY}
                      line={intention.line}
                      title={winterMarkerTitle(t, intention)}
                      scale={annotationScale}
                      color={WINTER_WARNING_COLOR}
                      dataTag="warning"
                    />
                  )}
                </g>
              )
            })}
            {errors.map((intention, index) => {
              const reason = winterReasonText(t, intention)
              return (
                <WinterMarkerTriangle
                  key={`error-${intention.line}-${index}`}
                  x={centerX - 22 * annotationScale}
                  y={centerY - (42 + index * 18) * annotationScale}
                  line={intention.line}
                  title={t('error.line', {
                    line: intention.line,
                    message: reason,
                  })}
                  scale={annotationScale}
                  color={WINTER_ERROR_COLOR}
                  dataTag="error"
                />
              )
            })}
          </g>
        )
      })}
      {winterIntentions
        .filter(
          (intention) =>
            intention.valid &&
            intention.kind === 'transfer' &&
            intention.sourceTerritory &&
            intention.targetTerritory,
        )
        .map((intention) => {
          const source = territories.find(
            (territory) => territory.id === intention.sourceTerritory,
          )
          const target = territories.find(
            (territory) => territory.id === intention.targetTerritory,
          )
          if (!source || !target) return null
          const [midX, midY] = [
            (centroid(source.points)[0] + centroid(target.points)[0]) / 2,
            (centroid(source.points)[1] + centroid(target.points)[1]) / 2,
          ]
          return (
            <g
              key={`transfer-${intention.line}-${intention.sourceTerritory}-${intention.targetTerritory}`}
            >
              <WinterTransferArrow
                from={centroid(source.points)}
                to={centroid(target.points)}
                amount={intention.amount}
                title={intention.label}
                color={intention.color ?? DRAFT_INTENTION_COLOR}
                scale={annotationScale}
              />
              {intention.warning && (
                <WinterMarkerTriangle
                  x={midX + 18 * annotationScale}
                  y={midY - 5 * annotationScale}
                  line={intention.line}
                  title={winterMarkerTitle(t, intention)}
                  scale={annotationScale}
                  color={WINTER_WARNING_COLOR}
                  dataTag="warning"
                />
              )}
            </g>
          )
        })}
      {winterIntentions
        .filter((intention) => !intention.valid && !intention.territory)
        .map((intention, index) => {
          const reason = winterReasonText(t, intention)
          return (
            <WinterMarkerTriangle
              key={`unplaced-error-${intention.line}-${index}`}
              x={(18 + (index % 8) * 20) * annotationScale}
              y={(22 + Math.floor(index / 8) * 18) * annotationScale}
              line={intention.line}
              title={t('error.line', {
                line: intention.line,
                message: reason,
              })}
              scale={annotationScale}
              color={WINTER_ERROR_COLOR}
              dataTag="error"
            />
          )
        })}
    </g>
  )
}
