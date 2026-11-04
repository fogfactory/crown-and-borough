import {
  BANNER_HEIGHT,
  BANNER_PATH,
  BANNER_WIDTH,
  CONTROL_SHIELD_SCALE,
  FIGURE_HEAD_TOP,
  SPEAR_TOP,
  SPEAR_X,
  armyFigures,
  greyArms,
  hashSeed,
  SHIELD_BOX,
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
  SHIELD_NUMBER,
  SOLDIER_FACE,
  SOLDIER_SHADOW,
  SOLDIER_STEEL,
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

/** The shield of a territory occupied against its controller: greyed. */
export function greyShieldSymbolId(owner: string): string {
  return `arms-shield-grey-${owner}`
}

export function soldierSymbolId(owner: string): string {
  return `arms-soldier-${owner}`
}

const SHADOW_SYMBOL_ID = 'arms-soldier-shadow'

// Soldier figurine, 5 x 11 units, feet on the origin: surcoat in the arms of
// its owner, steel helmet, face, and a spear held in the right hand (left of
// the image, the figurine facing the viewer).
const SURCOAT = 'M-2.4 0 L-2.65 -6.1 H2.65 L2.4 0 Z'
const HELMET = 'M-2.05 -7.75 Q-2.05 -10.95 0 -10.95 Q2.05 -10.95 2.05 -7.75 Z'

/** A field with its charge, clipped to `outline`, cased in ink. */
export function ArmsField({
  arms,
  outline,
  clipId,
  box,
  casing = CASING,
}: {
  arms: ResolvedArms
  outline: string
  clipId: string
  box: [number, number, number, number]
  /** Stroke of the ink casing, in units. */
  casing?: number
}) {
  const charge = arms.charge
  const shape = charge ? chargeShape(charge.kind, ...box) : null
  return (
    <>
      <clipPath id={clipId}>
        <path d={outline} />
      </clipPath>
      <path d={outline} fill={MAP_HALO} stroke={MAP_HALO} strokeWidth={casing * 3} />
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
        strokeWidth={casing}
        strokeLinejoin="round"
      />
    </>
  )
}

/**
 * One shield per owner, drawn once and reused by every controlled territory
 * through `<use>`; the soldier and banner of an owner with an army, and the
 * greyed shield of an owner whose territory is occupied, only when needed.
 */
export function HeraldryDefs({
  armsByOwner,
  armyOwners,
  occupiedOwners,
}: {
  armsByOwner: Map<string, ResolvedArms>
  armyOwners: Set<string>
  occupiedOwners: Set<string>
}) {
  return (
    <>
      {armyOwners.size > 0 && (
        <symbol id={SHADOW_SYMBOL_ID} overflow="visible">
          <g fill={SOLDIER_SHADOW}>
            <path d={SURCOAT} />
            <circle cy={-7.5} r={1.6} />
            <path d={HELMET} />
          </g>
        </symbol>
      )}
      {[...armsByOwner].map(([owner, arms]) => (
        <g key={owner}>
          <symbol id={shieldSymbolId(owner)} overflow="visible">
            <ArmsField
              arms={arms}
              outline={SHIELD_PATH}
              clipId={`${shieldSymbolId(owner)}-clip`}
              box={SHIELD_BOX}
            />
          </symbol>
          {occupiedOwners.has(owner) && (
            <symbol id={greyShieldSymbolId(owner)} overflow="visible">
              <ArmsField
                arms={greyArms(arms)}
                outline={SHIELD_PATH}
                clipId={`${greyShieldSymbolId(owner)}-clip`}
                box={SHIELD_BOX}
              />
            </symbol>
          )}
          {armyOwners.has(owner) && (
            <>
              <symbol id={soldierSymbolId(owner)} overflow="visible">
                <Soldier arms={arms} clipId={`${soldierSymbolId(owner)}-clip`} />
              </symbol>
              <symbol id={bannerSymbolId(owner)} overflow="visible">
                <ArmsField
                  arms={arms}
                  outline={BANNER_PATH}
                  clipId={`${bannerSymbolId(owner)}-clip`}
                  box={[0, 0, BANNER_WIDTH, BANNER_HEIGHT]}
                />
              </symbol>
            </>
          )}
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

/** One figurine of a pack, feet on the origin. */
export function Soldier({ arms, clipId }: { arms: ResolvedArms; clipId: string }) {
  return (
    <g>
      <line
        x1={SPEAR_X}
        y1={0.8}
        x2={SPEAR_X}
        y2={SPEAR_TOP}
        stroke={MAP_INK}
        strokeWidth={0.55}
        strokeLinecap="round"
      />
      <ArmsField
        arms={arms}
        outline={SURCOAT}
        clipId={clipId}
        box={[-2.65, -6.1, 5.3, 6.1]}
        casing={0.4}
      />
      <path
        d={`M-2.6 -5.4 L${SPEAR_X + 0.15} -3.8`}
        stroke={MAP_INK}
        strokeWidth={0.6}
        strokeLinecap="round"
      />
      <circle
        cx={SPEAR_X}
        cy={-3.8}
        r={0.75}
        fill={SOLDIER_FACE}
        stroke={MAP_INK}
        strokeWidth={0.3}
      />
      <circle cy={-7.5} r={1.6} fill={SOLDIER_FACE} stroke={MAP_INK} strokeWidth={0.35} />
      <path d={HELMET} fill={SOLDIER_STEEL} stroke={MAP_INK} strokeWidth={0.4} />
      <path d="M0 -7.75 V-6.6" stroke={MAP_INK} strokeWidth={0.4} />
    </g>
  )
}

/** Foot-to-top room kept between the highest head and the banner, in units. */
const BANNER_CLEARANCE = 10

/**
 * An army: a pack of up to six upright figurines in the owner's arms, shifted
 * irregularly (a function of `seed`), grey shadows behind a big army, and a
 * numbered banner whose pole is the spear of the leftmost soldier, above
 * every head. `compact` keeps the standard-bearer alone (overview zoom). Feet
 * on the origin, centred on x = 0.
 */
export function ArmyGlyph({
  owner,
  size,
  seed,
  compact = false,
  hrefs,
}: {
  owner: string
  size: number
  /** Names the pack's disorder: the territory id. */
  seed: string
  compact?: boolean
  /** Symbols to draw, by default the owner's from `HeraldryDefs`. */
  hrefs?: { soldier: string; banner: string }
}) {
  const figures = armyFigures(size, hashSeed(seed))
  const soldiers = compact ? [{ x: 0, y: 0 }] : figures.soldiers
  const shadows = compact ? [] : figures.shadows
  const bearer = compact ? soldiers[0] : soldiers[figures.bearer]
  const headTop =
    Math.min(
      ...soldiers.map((soldier) => soldier.y),
      ...shadows.map((shadow) => shadow.y),
    ) + FIGURE_HEAD_TOP
  const poleX = bearer.x + SPEAR_X
  const flagTop = headTop - BANNER_CLEARANCE - BANNER_HEIGHT
  const ordered = [...soldiers].sort((first, second) => first.y - second.y)
  return (
    <g data-army-owner={owner} data-army-size={size}>
      {shadows.map((shadow) => (
        <use
          key={`${shadow.x}-${shadow.y}`}
          href={`#${SHADOW_SYMBOL_ID}`}
          x={shadow.x}
          y={shadow.y}
          opacity={0.4}
          data-army-shadow
        />
      ))}
      {ordered.map((soldier) => (
        <use
          key={`${soldier.x}-${soldier.y}`}
          href={hrefs?.soldier ?? `#${soldierSymbolId(owner)}`}
          x={soldier.x}
          y={soldier.y}
          data-army-soldier
        />
      ))}
      <line
        x1={poleX}
        y1={bearer.y + SPEAR_TOP}
        x2={poleX}
        y2={flagTop - 0.4}
        stroke={MAP_HALO}
        strokeWidth={2.2}
        strokeLinecap="round"
      />
      <line
        x1={poleX}
        y1={bearer.y + SPEAR_TOP}
        x2={poleX}
        y2={flagTop - 0.4}
        stroke={MAP_INK}
        strokeWidth={0.9}
        strokeLinecap="round"
      />
      <use
        href={hrefs?.banner ?? `#${bannerSymbolId(owner)}`}
        x={poleX + 0.4}
        y={flagTop}
      />
      <text
        x={poleX + 0.4 + BANNER_WIDTH * 0.4}
        y={flagTop + BANNER_HEIGHT / 2}
        fill={SHIELD_NUMBER}
        fontSize={6.5}
        fontWeight="700"
        textAnchor="middle"
        dominantBaseline="central"
        stroke={MAP_INK}
        strokeWidth={1.6}
        strokeLinejoin="round"
        paintOrder="stroke"
      >
        {size}
      </text>
    </g>
  )
}

/** Half the control shield's width and its height, in units. */
const CONTROL_HALF_WIDTH = (SHIELD_WIDTH * CONTROL_SHIELD_SCALE) / 2
const CONTROL_HEIGHT = SHIELD_HEIGHT * CONTROL_SHIELD_SCALE

/** From a label's left edge to the control shield's centre: half a shield and a gap. */
export const CONTROL_REACH = CONTROL_HALF_WIDTH + 3

/**
 * The shield of the controller, its point on the origin: greyed while an
 * army of another player occupies the territory, crowned on a fief capital,
 * and writing its fief's capital trigram under its point.
 */
export function ControlShield({
  owner,
  grey = false,
  crown,
  attachedTo,
  shieldHref,
}: {
  owner: string
  grey?: boolean
  crown?: CrownKind
  attachedTo?: string
  /** Shield symbol to draw, by default the owner's from `HeraldryDefs`. */
  shieldHref?: string
}) {
  const href =
    shieldHref ?? `#${grey ? greyShieldSymbolId(owner) : shieldSymbolId(owner)}`
  return (
    <g data-control-owner={owner} data-control-grey={grey ? '' : undefined}>
      <use
        href={href}
        transform={`translate(0 ${-CONTROL_HEIGHT / 2}) scale(${CONTROL_SHIELD_SCALE})`}
      />
      {crown && (
        <g transform={`translate(0 ${-CONTROL_HEIGHT - 0.4})`}>
          <CrownGlyph kind={crown} />
        </g>
      )}
      {attachedTo && (
        <text
          x={0}
          y={7}
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
