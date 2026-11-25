import type { ReactNode } from 'react'
import { useLanguage } from '@/i18n/LanguageContext'
import {
  affluentCurve,
  borderSlotPoint,
  middleOf,
  type AffluentPath,
  type DefensiveSupport,
  type Intention,
} from '@/lib/intent-overlay'
import { distanceBetween, midpointOf } from '@/lib/map-gestures'
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

/** The value of a support: a shield for a defensive one, a round disc for an offensive one. */
function SupportValueDisc({
  x,
  y,
  value,
  color,
  scale,
  label,
  turnLabel,
  shape = 'shield',
}: {
  x: number
  y: number
  value: number
  color: string
  scale: number
  label: string
  turnLabel?: string
  shape?: 'shield' | 'disc'
}) {
  return (
    <g transform={`translate(${x} ${y})`} data-intent-support-value={value}>
      <title>{label}</title>
      {shape === 'disc' ? (
        <circle
          r={8 * scale}
          fill={color}
          stroke={MAP_PARCHMENT}
          strokeWidth={1.5 * scale}
        />
      ) : (
        <path
          d={`M ${-8 * scale} ${-9 * scale} H ${8 * scale} V ${1 * scale} C ${8 * scale} ${6 * scale} ${3 * scale} ${9 * scale} 0 ${11 * scale} C ${-3 * scale} ${9 * scale} ${-8 * scale} ${6 * scale} ${-8 * scale} ${1 * scale} Z`}
          fill={color}
          stroke={MAP_PARCHMENT}
          strokeWidth={1.5 * scale}
          strokeLinejoin="round"
        />
      )}
      <text
        y={(shape === 'disc' ? 0 : -1) * scale}
        textAnchor="middle"
        dominantBaseline="central"
        fill={MAP_PARCHMENT}
        fontSize={9 * scale}
        fontWeight="700"
      >
        +{value}
      </text>
      {turnLabel && (
        <text
          y={-16 * scale}
          textAnchor="middle"
          dominantBaseline="central"
          fill={color}
          fontSize={9 * scale}
          fontWeight="700"
          stroke={MAP_PARCHMENT}
          strokeWidth={2.5 * scale}
          paintOrder="stroke"
        >
          {turnLabel}
        </text>
      )}
    </g>
  )
}

/**
 * A defensive support: a row of stakes along each passable border of the held
 * territory, bases inside and points towards the border, in the supporter's
 * colour. Several supports share each border side by side.
 */
function Palisade({
  support,
  from,
  color,
  scale,
  dash,
  label,
}: {
  support: DefensiveSupport
  from: [number, number]
  color: string
  scale: number
  dash?: string
  label: string
}) {
  const depth = 9 * scale
  const width = 3.6 * scale
  const spacing = 6 * scale
  const t0 = support.slot / support.slots
  const t1 = (support.slot + 1) / support.slots
  const anchor = borderSlotPoint(support.link, support.slot, support.slots)
  const stakes: string[] = []
  for (const border of support.borders) {
    const length = distanceBetween(border.from, border.to)
    const count = Math.max(1, Math.round(((t1 - t0) * length) / spacing))
    const ux = (border.to[0] - border.from[0]) / (length || 1)
    const uy = (border.to[1] - border.from[1]) / (length || 1)
    for (let index = 0; index < count; index += 1) {
      const t = t0 + ((t1 - t0) * (index + 0.5)) / count
      // The point of the stake sits just inside the border, its base deeper in.
      const tx =
        border.from[0] +
        (border.to[0] - border.from[0]) * t +
        border.inward[0] * 1.5 * scale
      const ty =
        border.from[1] +
        (border.to[1] - border.from[1]) * t +
        border.inward[1] * 1.5 * scale
      const bx = tx + border.inward[0] * depth
      const by = ty + border.inward[1] * depth
      stakes.push(
        `M ${tx} ${ty} L ${bx + (ux * width) / 2} ${by + (uy * width) / 2} L ${bx - (ux * width) / 2} ${by - (uy * width) / 2} Z`,
      )
    }
  }
  const path = stakes.join(' ')
  return (
    <g data-intent-palisade={support.supported}>
      <path
        d={path}
        data-intent-outline="true"
        fill={INTENT_OUTLINE_COLOR}
        stroke={INTENT_OUTLINE_COLOR}
        strokeWidth={1.6 * scale}
        strokeLinejoin="round"
      />
      <path d={path} fill={color} stroke={color} strokeWidth={0.4 * scale} />
      <line
        data-intent-outline="true"
        x1={from[0]}
        y1={from[1]}
        x2={anchor[0]}
        y2={anchor[1]}
        stroke={INTENT_OUTLINE_COLOR}
        strokeWidth={2.8 * scale}
        strokeDasharray={dash}
        strokeLinecap="round"
      />
      <line
        data-intent-support-link="true"
        x1={from[0]}
        y1={from[1]}
        x2={anchor[0]}
        y2={anchor[1]}
        stroke={color}
        strokeWidth={1.2 * scale}
        strokeDasharray={dash}
        strokeLinecap="round"
      />
      {support.borders.map((border, index) => {
        const [x, y] = borderSlotPoint(border, support.slot, support.slots)
        return (
          <SupportValueDisc
            key={index}
            x={x + border.inward[0] * 17 * scale}
            y={y + border.inward[1] * 17 * scale}
            value={support.value}
            color={color}
            scale={scale}
            label={label}
          />
        )
      })}
    </g>
  )
}

/** The path of an offensive support's tributary: from the army, bowing into the attacked territory. */
function affluentPath({ curves, tail }: AffluentPath): string {
  const [[from, a1, a2, crossing], [, b1, b2, end]] = curves
  return `M ${from[0]} ${from[1]} C ${a1[0]} ${a1[1]} ${a2[0]} ${a2[1]} ${crossing[0]} ${crossing[1]} C ${b1[0]} ${b1[1]} ${b2[0]} ${b2[1]} ${end[0]} ${end[1]} L ${tail[0]} ${tail[1]}`
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
              if (segment.defensive) {
                const badgeAt = midpointOf(segment.from, middleOf(segment.defensive.link))
                return (
                  <g key={segmentIndex}>
                    <Palisade
                      support={segment.defensive}
                      from={segment.from}
                      color={intentionColor}
                      scale={annotationScale}
                      dash={draftDash}
                      label={intention.label}
                    />
                    <IntentBadge
                      x={badgeAt[0]}
                      y={badgeAt[1]}
                      symbol={intention.symbol}
                      turnLabel={intention.turnLabel}
                      color={intentionColor}
                      scale={annotationScale}
                    />
                  </g>
                )
              }
              if (segment.offensive) {
                const support = segment.offensive
                const tributary = affluentCurve(support)
                const path = affluentPath(tributary)
                // A single shield sits at the middle of the curve.
                const disc = tributary.curves[0][3]
                return (
                  <g key={segmentIndex} data-intent-affluent={support.destination}>
                    <path
                      d={path}
                      data-intent-outline="true"
                      fill="none"
                      stroke={INTENT_OUTLINE_COLOR}
                      strokeWidth={3.4 * annotationScale}
                      strokeDasharray={draftDash}
                      strokeLinecap="round"
                    />
                    <path
                      d={path}
                      fill="none"
                      stroke={intentionColor}
                      strokeWidth={1.4 * annotationScale}
                      strokeDasharray={draftDash}
                      strokeLinecap="round"
                      markerEnd="url(#intent-affluent-arrow)"
                    />
                    <SupportValueDisc
                      x={disc[0]}
                      y={disc[1]}
                      value={support.value}
                      color={intentionColor}
                      scale={annotationScale}
                      label={intention.label}
                      turnLabel={intention.turnLabel}
                      shape="disc"
                    />
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
                markerKind === 'circle' ? 'url(#intent-circle-outline)' : undefined

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
