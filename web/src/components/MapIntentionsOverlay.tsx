import type { ReactNode } from 'react'
import { useLanguage } from '@/i18n/LanguageContext'
import type { Intention } from '@/lib/intent-overlay'
import { distanceBetween } from '@/lib/map-gestures'
import { NoblePawnSilhouette } from '@/components/MapHeraldry'
import { IntentBadge } from '@/components/MapMarkers'
import {
  DRAFT_INTENTION_OPACITY,
  INTENT_OUTLINE_COLOR,
  MAP_PARCHMENT,
} from '@/lib/map-palette'

/** The disc of the order's colour that holds a mark on a leg. */
function CarrierDisc({
  x,
  y,
  color,
  scale,
  label,
  children,
  ...data
}: {
  x: number
  y: number
  color: string
  scale: number
  label: string
  children: ReactNode
  'data-intent-nobles'?: string
  'data-intent-chain'?: string
}) {
  return (
    <g transform={`translate(${x} ${y})`} {...data}>
      <title>{label}</title>
      <circle
        r={8 * scale}
        fill={color}
        stroke={MAP_PARCHMENT}
        strokeWidth={1.5 * scale}
      />
      {children}
    </g>
  )
}

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
    <CarrierDisc
      x={x}
      y={y}
      color={color}
      scale={scale}
      label={named.length > 0 ? named.join(', ') : '*'}
      data-intent-nobles={nobles.join(',')}
    >
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
    </CarrierDisc>
  )
}

/** Marks the leg the order sheet follows: a folded sheet in a disc of the order's colour. */
function ChainSheetMark({
  x,
  y,
  color,
  scale,
  label,
}: {
  x: number
  y: number
  color: string
  scale: number
  label: string
}) {
  return (
    <CarrierDisc
      x={x}
      y={y}
      color={color}
      scale={scale}
      label={label}
      data-intent-chain="true"
    >
      <g transform={`scale(${scale})`}>
        <path d="M -3.4 -4.8 H 1.4 L 4.2 -2 V 4.8 H -3.4 Z" fill={MAP_PARCHMENT} />
        <path
          d="M 1.4 -4.8 V -2 H 4.2 M -1.8 0.4 H 2.6 M -1.8 2.6 H 2.6"
          fill="none"
          stroke={color}
          strokeWidth={0.9}
          strokeLinejoin="round"
        />
      </g>
    </CarrierDisc>
  )
}

/** A vector of the given length along a leg, from its head back towards its origin. */
function legStep(
  segment: Intention['segments'][number],
  length: number,
): [number, number] {
  const scale = length / (distanceBetween(segment.from, segment.to) || 1)
  return [
    (segment.from[0] - segment.to[0]) * scale,
    (segment.from[1] - segment.to[1]) * scale,
  ]
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
        // The marks of a leg sit side by side, the sheet after the nobles
        // along `step` (a vector as long as the gap between two marks).
        const carrierMark = (
          segment: Intention['segments'][number],
          x: number,
          y: number,
          step: [number, number],
        ) => (
          <>
            {segment.nobles && (
              <NobleCarrierMark
                x={x}
                y={y}
                nobles={segment.nobles}
                color={intentionColor}
                scale={annotationScale}
              />
            )}
            {segment.carriesChain && (
              <ChainSheetMark
                x={segment.nobles ? x + step[0] : x}
                y={segment.nobles ? y + step[1] : y}
                color={intentionColor}
                scale={annotationScale}
                label={t('map.chainCarrier')}
              />
            )}
          </>
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
                      [direction * 22 * annotationScale, 0],
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
                    // Along the leg, towards its origin.
                    legStep(segment, 22 * annotationScale),
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
