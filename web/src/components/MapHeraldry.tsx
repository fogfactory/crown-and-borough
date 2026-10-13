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
import {
  CARDINAL_ROBE,
  CROWN_FILL,
  MAP_HALO,
  MAP_INK,
  MARKER_BRASS,
  NEUTRAL_ARMY_COLOR,
  POPE_ROBE,
} from '@/lib/map-palette'
import type { NobleAttributes } from '@/lib/noble-attributes'
import type { NobleStatus } from '@/types'

/** A courtesy crown is a bare `brass-700` outline. */
const COURTESY_OUTLINE = MARKER_BRASS

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
  /** Courtesy title: the outline color, with no fill. */
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

// Noble pawn, 11 × 17 units centred on the origin: base, body and head.
const PAWN_BASE = 'M-5.5 5.6 H5.5 V7.3 Q5.5 8.5 4.3 8.5 H-4.3 Q-5.5 8.5 -5.5 7.3 Z'
const PAWN_BODY = 'M-4.1 5.6 L-2 -2.6 Q0 -3.6 2 -2.6 L4.1 5.6 Z'
const PAWN_HEAD = { cy: -5.6, r: 2.9 }
const CARDINAL_HAT =
  'M-4.7 -7.9 Q0 -9.4 4.7 -7.9 Q0 -6.9 -4.7 -7.9 Z M-2.4 -8.3 Q0 -11.4 2.4 -8.3 Z'
const TIARA = 'M-2.3 -7.9 Q-2.6 -12.4 0 -12.9 Q2.6 -12.4 2.3 -7.9 Z'
const TIARA_BANDS = 'M-2.4 -9.3 H2.4 M-2.3 -10.9 H2.3'
// Held on the left: pawns side by side paint left to right, so a crosier on
// the right would hide under the next pawn.
const CROSIER = 'M-6.7 8.5 V-5.2 Q-6.7 -9.2 -9.1 -8.7 Q-10.7 -8.1 -9.7 -6.4'
const DUNGEON_BARS = 'M-3.6 -9.5 V8.9 M0 -9.5 V8.9 M3.6 -9.5 V8.9 M-5 -9.5 H5 M-5 8.9 H5'

/** Height of the crown's resting line above the pawn, by headwear. */
const CROWN_REST = { none: -7.6, cardinal: -10.4, pope: -12.6 } as const

/**
 * A noble's pawn in its owner's tincture, with its title attributes: the
 * crown of its highest secular title (a bare brass outline for a courtesy
 * title), the bishop's crosier, the cardinal's red robe and hat or the pope's
 * white robe and tiara. A hostage or a prisoner turns ashen, the latter
 * behind ink bars; an excommunicated noble's base is ringed with ink dots.
 */
export function NoblePawn({
  color,
  attributes,
  status = 'free',
}: {
  color: string
  attributes?: NobleAttributes
  status?: NobleStatus
}) {
  const tincture = status === 'free' ? color : NEUTRAL_ARMY_COLOR
  const robe = attributes?.robe
  const bodyFill =
    robe === 'pope' ? POPE_ROBE : robe === 'cardinal' ? CARDINAL_ROBE : tincture
  const crown = attributes?.crown ?? attributes?.courtesyCrown
  const parts = (
    <>
      <path d={PAWN_BASE} />
      <path d={PAWN_BODY} />
      <circle cy={PAWN_HEAD.cy} r={PAWN_HEAD.r} />
    </>
  )
  return (
    <g data-noble-pawn={status} strokeLinejoin="round" strokeLinecap="round">
      <g fill={MAP_HALO} stroke={MAP_HALO} strokeWidth={2.6}>
        {parts}
      </g>
      {attributes?.crosier && (
        <g data-attribute="crosier" fill="none">
          <path d={CROSIER} stroke={MAP_HALO} strokeWidth={3} />
          <path d={CROSIER} stroke={MAP_INK} strokeWidth={2} />
          <path d={CROSIER} stroke={CROWN_FILL} strokeWidth={1} />
        </g>
      )}
      <path
        d={PAWN_BASE}
        data-part="base"
        fill={tincture}
        stroke={MAP_INK}
        strokeWidth={attributes?.excommunicated ? 1.1 : 0.7}
        strokeDasharray={attributes?.excommunicated ? '0.1 1.5' : undefined}
        data-attribute={attributes?.excommunicated ? 'excommunicated' : undefined}
      />
      <path
        d={PAWN_BODY}
        data-part="body"
        fill={bodyFill}
        stroke={MAP_INK}
        strokeWidth={0.7}
        data-attribute={robe ? `${robe}-robe` : undefined}
      />
      <circle
        cy={PAWN_HEAD.cy}
        r={PAWN_HEAD.r}
        fill={tincture}
        stroke={MAP_INK}
        strokeWidth={0.7}
      />
      {robe === 'cardinal' && (
        <path
          d={CARDINAL_HAT}
          fill={CARDINAL_ROBE}
          stroke={MAP_INK}
          strokeWidth={0.6}
          data-attribute="cardinal-hat"
        />
      )}
      {robe === 'pope' && (
        <g data-attribute="tiara">
          <path d={TIARA} fill={POPE_ROBE} stroke={MAP_INK} strokeWidth={0.6} />
          <path d={TIARA_BANDS} stroke={CROWN_FILL} strokeWidth={0.7} />
        </g>
      )}
      {crown && (
        <g
          transform={`translate(0 ${CROWN_REST[robe ?? 'none']}) scale(0.62)`}
          data-attribute={attributes?.crown ? 'crown' : 'courtesy-crown'}
        >
          <CrownGlyph
            kind={crown}
            outline={attributes?.crown ? undefined : COURTESY_OUTLINE}
          />
        </g>
      )}
      {status === 'dungeon' && (
        <g data-attribute="dungeon" fill="none">
          <path d={DUNGEON_BARS} stroke={MAP_HALO} strokeWidth={2} />
          <path d={DUNGEON_BARS} stroke={MAP_INK} strokeWidth={0.9} />
        </g>
      )}
    </g>
  )
}
