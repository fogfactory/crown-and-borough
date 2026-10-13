import {
  BANNER_HEIGHT,
  BANNER_PATH,
  BANNER_WIDTH,
  SHIELD_HEIGHT,
  SHIELD_PATH,
  SHIELD_WIDTH,
  chargeShape,
  crownShape,
  type CrownKind,
  type ResolvedArms,
} from '@/lib/heraldry'
import { CROWN_FILL, MAP_HALO, MAP_INK } from '@/lib/map-palette'

/** Stroke of the ink casing around shields, banners and crowns, in units. */
const CASING = 0.9

export function shieldSymbolId(owner: string): string {
  return `arms-shield-${owner}`
}

export function bannerSymbolId(owner: string): string {
  return `arms-banner-${owner}`
}

/** A field with its charge, clipped to `outline`, cased in ink. */
export function ArmsField({
  arms,
  outline,
  clipId,
  box,
}: {
  arms: ResolvedArms
  outline: string
  clipId: string
  box: [number, number, number, number]
}) {
  const charge = arms.charge
  const shape = charge ? chargeShape(charge.kind, ...box) : null
  return (
    <>
      <clipPath id={clipId}>
        <path d={outline} />
      </clipPath>
      <path d={outline} fill={MAP_HALO} stroke={MAP_HALO} strokeWidth={CASING * 3} />
      <path d={outline} fill={arms.field} />
      {charge && shape && (
        <g clipPath={`url(#${clipId})`} data-charge={charge.kind}>
          {charge.kind === 'bordure' ? (
            <path
              d={outline}
              fill="none"
              stroke={charge.metal}
              strokeWidth={shape.stroke! * 2}
            />
          ) : shape.stroke ? (
            <path
              d={shape.d}
              fill="none"
              stroke={charge.metal}
              strokeWidth={shape.stroke}
            />
          ) : (
            <path d={shape.d} fill={charge.metal} />
          )}
        </g>
      )}
      <path
        d={outline}
        fill="none"
        stroke={MAP_INK}
        strokeWidth={CASING}
        strokeLinejoin="round"
      />
    </>
  )
}

/**
 * One shield and one banner per owner, drawn once and reused by every army
 * and every controlled territory through `<use>`.
 */
export function HeraldryDefs({
  armsByOwner,
}: {
  armsByOwner: Map<string, ResolvedArms>
}) {
  return (
    <>
      {[...armsByOwner].map(([owner, arms]) => (
        <g key={owner}>
          <symbol id={shieldSymbolId(owner)} overflow="visible">
            <ArmsField
              arms={arms}
              outline={SHIELD_PATH}
              clipId={`${shieldSymbolId(owner)}-clip`}
              box={[-SHIELD_WIDTH / 2, -SHIELD_HEIGHT / 2, SHIELD_WIDTH, SHIELD_HEIGHT]}
            />
          </symbol>
          <symbol id={bannerSymbolId(owner)} overflow="visible">
            <ArmsField
              arms={arms}
              outline={BANNER_PATH}
              clipId={`${bannerSymbolId(owner)}-clip`}
              box={[0, 0, BANNER_WIDTH, BANNER_HEIGHT]}
            />
          </symbol>
        </g>
      ))}
    </>
  )
}

/**
 * A rank crown in brass cased with ink, resting on y = 0. A courtesy title
 * (4c) draws it as a bare brass outline.
 */
export function CrownGlyph({
  kind,
  outline,
}: {
  kind: CrownKind
  /** Courtesy title: brass outline only, no fill. */
  outline?: string
}) {
  const { solid, pearls } = crownShape(kind)
  const fill = outline ? 'none' : CROWN_FILL
  const stroke = outline ?? MAP_INK
  return (
    <g data-crown={kind} strokeLinejoin="round">
      <path d={solid} fill={MAP_HALO} stroke={MAP_HALO} strokeWidth={1.6} />
      {pearls.map(([cx, cy, r]) => (
        <circle key={`halo-${cx}`} cx={cx} cy={cy} r={r + 0.6} fill={MAP_HALO} />
      ))}
      <path d={solid} fill={fill} stroke={stroke} strokeWidth={0.5} />
      {pearls.map(([cx, cy, r]) => (
        <circle
          key={cx}
          cx={cx}
          cy={cy}
          r={r}
          fill={fill}
          stroke={stroke}
          strokeWidth={0.4}
        />
      ))}
    </g>
  )
}

/** Height of the banner's mast above its foot, in units. */
export const MAST_HEIGHT = 14

/**
 * A swallowtail banner on an ink mast, its foot at the origin: the flag is
 * hoisted at the mast top, a crown may rest on it (fief capital), and a fief
 * member writes its capital's trigram under the flag.
 */
export function BannerGlyph({
  owner,
  crown,
  attachedTo,
  flagHref = `#${bannerSymbolId(owner)}`,
}: {
  owner: string
  crown?: CrownKind
  attachedTo?: string
  /** Flag symbol to draw, by default the owner's from `HeraldryDefs`. */
  flagHref?: string
}) {
  const hoist = -MAST_HEIGHT
  return (
    <g data-banner-owner={owner}>
      <line
        x1={0}
        y1={0.5}
        x2={0}
        y2={hoist - 0.6}
        stroke={MAP_HALO}
        strokeWidth={2.6}
        strokeLinecap="round"
      />
      <line
        x1={0}
        y1={0.5}
        x2={0}
        y2={hoist - 0.6}
        stroke={MAP_INK}
        strokeWidth={1.1}
        strokeLinecap="round"
      />
      <use href={flagHref} x={0.5} y={hoist} />
      {crown && (
        <g transform={`translate(${BANNER_WIDTH / 2} ${hoist - 0.8})`}>
          <CrownGlyph kind={crown} />
        </g>
      )}
      {attachedTo && (
        <text
          x={0.5 + BANNER_WIDTH / 2}
          y={hoist + BANNER_HEIGHT + 6}
          fill={MAP_INK}
          fontFamily="var(--font-display)"
          fontSize={6}
          fontWeight="700"
          letterSpacing={0.3}
          textAnchor="middle"
          stroke={MAP_HALO}
          strokeWidth={2}
          paintOrder="stroke"
          data-fief-attachment={attachedTo}
        >
          {attachedTo}
        </text>
      )}
    </g>
  )
}
