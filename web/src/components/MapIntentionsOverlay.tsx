import { useLanguage } from '@/i18n/LanguageContext'
import type { Intention } from '@/lib/intent-overlay'
import { NoblePawnSilhouette } from '@/components/MapHeraldry'
import { IntentBadge } from '@/components/MapMarkers'
import { DRAFT_INTENTION_OPACITY, INTENT_OUTLINE_COLOR, MAP_PARCHMENT } from '@/lib/map-palette'

/** Marks the leg a dispersion's nobles take: the pawn silhouette in a disc of the order's colour. */
function NobleCarrierMark({
  x,
  y,
  nobles,
  color,
  scale,
}: {
  x: number
  y: number
  nobles: string[]
  color: string
  scale: number
}) {
  const named = nobles.filter((noble) => noble !== '*')
  return (
    <g transform={`translate(${x} ${y})`} data-intent-nobles={nobles.join(',')}>
      <title>{named.length > 0 ? named.join(', ') : '*'}</title>
      <circle
        r={8 * scale}
        fill={color}
        stroke={MAP_PARCHMENT}
        strokeWidth={1.5 * scale}
      />
      <g transform={`scale(${0.6 * scale})`}>
        <NoblePawnSilhouette fill={MAP_PARCHMENT} />
      </g>
      {named.length > 0 && (
        <text
          x={11 * scale}
          y={-1 * scale}
          fill={color}
          fontSize={9 * scale}
          fontWeight="700"
          textAnchor="start"
          stroke={MAP_PARCHMENT}
          strokeWidth={2.5 * scale}
          paintOrder="stroke"
        >
          {named.join(' ')}
        </text>
      )}
    </g>
  )
}

export function IntentionsOverlay({
  intentions,
  annotationScale,
  passableBorderDash,
  intentionsColor,
}: {
  intentions: Intention[]
  annotationScale: number
  passableBorderDash: string
  intentionsColor: string
}) {
  const { t } = useLanguage()
  if (intentions.length === 0) return null

  return (
    <g aria-label={t('map.intentionsOverlay')} pointerEvents="none">
      {intentions.map((intention, index) => {
        const isDraft = intention.source === 'draft'
        const intentionColor = intention.color ?? intentionsColor
        const markerEndFor = (kind: 'arrow' | 'circle') => `url(#intent-${kind})`
        // A draft keeps the same arrow, faded and dashed.
        const draftDash = isDraft ? passableBorderDash : undefined
        const carrierMark = (segment: Intention['segments'][number], x: number, y: number) =>
          segment.nobles && (
            <NobleCarrierMark
              x={x}
              y={y}
              nobles={segment.nobles}
              color={intentionColor}
              scale={annotationScale}
            />
          )

        return (
          <g
            key={`${intention.armyTerritory}-${index}`}
            opacity={isDraft ? DRAFT_INTENTION_OPACITY : 1}
            data-intent-draft={isDraft || undefined}
          >
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
                      strokeDasharray={draftDash}
                      strokeLinecap="round"
                      markerEnd="url(#intent-arrow-outline)"
                    />
                    <path
                      d={path}
                      fill="none"
                      stroke={intentionColor}
                      strokeWidth={2.5 * annotationScale}
                      strokeDasharray={draftDash}
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
                    />
                    {carrierMark(
                      segment,
                      segment.from[0] + direction * 30 * annotationScale,
                      segment.from[1] - 8 * annotationScale,
                    )}
                  </g>
                )
              }
              const strokeWidth =
                segment.kind === 'attack' ? 4 * annotationScale : 2.5 * annotationScale
              const strokeDasharray =
                segment.kind === 'support-defensive' ||
                segment.kind === 'support-offensive'
                  ? passableBorderDash
                  : draftDash
              const markerKind = segment.kind === 'support-defensive' ? 'circle' : 'arrow'
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
                  />
                  {carrierMark(
                    segment,
                    segment.from[0] + (segment.to[0] - segment.from[0]) * 0.8,
                    segment.from[1] + (segment.to[1] - segment.from[1]) * 0.8,
                  )}
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
              />
            )}
          </g>
        )
      })}
    </g>
  )
}
