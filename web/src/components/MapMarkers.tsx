import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import type { WinterIntention } from '@/lib/winter-overlay'
import type { GameIconGlyph as GameIconGlyphSpec } from '@/lib/game-icon-glyphs'
import { OWNERSHIP_SHIELD_PATH } from '@/components/MapLegend'
import type { Infrastructure, Noble, Point } from '@/types'

/** Outline color shared by every intention/order annotation on the map. */
export const INTENT_OUTLINE_COLOR = '#17120f'
/** Fill used for a drafted (not yet submitted) order's markers. */
export const DRAFT_INTENTION_COLOR = '#d4a39b'

// Map marker artwork from game-icons.net (CC BY 3.0, icons by Delapouite):
// https://game-icons.net/1x1/delapouite/castle.html
// https://game-icons.net/1x1/delapouite/village.html
// https://game-icons.net/1x1/delapouite/windmill.html
// https://game-icons.net/1x1/delapouite/barn.html
// Filled 512x512 silhouettes, scaled down to the marker footprint.
const MARKER_GLYPHS: Record<Infrastructure['type'], string> = {
  castle:
    'M255.95 27.11L180.6 107.614l150.7 1.168-75.35-81.674h-.003zM25 109.895v68.01l19.412 25.99h71.06l19.528-26v-68h-14v15.995h-18v-15.994H89v15.995H71v-15.994H57v15.995H39v-15.994H25zm352 0v68l19.527 26h71.06L487 177.906v-68.01h-14v15.995h-18v-15.994h-14v15.995h-18v-15.994h-14v15.995h-18v-15.994h-14zm-176 15.877V260.89h110V126.63l-110-.857zm55 20.118c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zM41 221.897V484.89h78V221.897H41zm352 0V484.89h78V221.897h-78zM56 241.89c4 0 8 4 8 12v32H48v-32c0-8 4-12 8-12zm400 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12zm-303 37v23h-16v183h87v-55c0-24 16-36 32-36s32 12 32 36v55h87v-183h-16v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14zm-49 43c4 0 8 4 8 12v32H96v-32c0-8 4-12 8-12zm72 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm80 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm80 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm72 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12zm-352 64c4 0 8 4 8 12v32H48v-32c0-8 4-12 8-12zm400 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12z',
  village:
    'M109.902 35.87l-71.14 59.284h142.28l-71.14-59.285zm288 32l-71.14 59.284h142.28l-71.14-59.285zM228.73 84.403l-108.9 90.75h217.8l-108.9-90.75zm-173.828 28.75v62h36.81l73.19-60.992v-1.008h-110zm23 14h16v18h-16v-18zm265 18v10.963l23 19.166v-16.13h16v18h-13.756l.104.087 19.098 15.914h-44.446v14h78v-39h18v39h14v-62h-110zm-194.345 48v20.08l24.095-20.08h-24.095zm28.158 0l105.1 87.582 27.087-22.574v-65.008H176.715zm74.683 14h35.735v34h-35.735v-34zm-76.714 7.74L30.37 335.153H319l-144.314-120.26zm198.046 13.51l-76.857 64.047 32.043 26.704H481.63l-108.9-90.75zm-23.214 108.75l.103.086 19.095 15.914h-72.248v77.467h60.435v-63.466h50v63.467h46v-93.466H349.516zm-278.614 16V476.13h126v-76.976h50v76.977h31.565V353.155H70.902zm30 30h50v50h-50v-50z',
  mill: 'M161.188 22L102.25 41.656 230.063 169.47l.843-.845 2.406-2.406L161.188 22zm246.906 18L280.28 167.813l.814.812 2.406 2.406 144.25-72.124L408.094 40zM256 40.938l-53.97 26.968 45.657 91.344c2.727-.648 5.52-.97 8.313-.97 3.306 0 6.614.467 9.813 1.376l75.875-75.875L256 40.938zm-88 89.093V184h53.906c.006-.02-.005-.043 0-.063L168 130.03zm176 28.657L293.375 184H344v-25.313zm-88 15.5c-4.975 0-9.94 1.908-13.78 5.75-7.686 7.685-7.686 19.91 0 27.594 7.683 7.686 19.877 7.686 27.56 0 7.686-7.683 7.686-19.908 0-27.593-3.84-3.842-8.805-5.75-13.78-5.75zM199.312 201l-2.875 13.594 25.094-12.563c-.08-.345-.146-.682-.218-1.03h-22zm91.375 0c-.176.856-.353 1.72-.593 2.563l29.312 29.312-6.72-31.875H290.69zM228.5 216.47L84.25 288.562l19.656 58.937L231.72 219.687l-.814-.843-2.406-2.375zm53.438 1.56l-.844.814-2.375 2.406L350.81 365.5l58.938-19.656L281.937 218.03zm-35.75 9.814l-66.532 66.53L139.094 487H216v-63h80v63h76.906l-22.03-104.688-1.595.532-6.56 2.22-3.126-6.22-75.28-150.625c-5.956 1.416-12.227 1.302-18.127-.376z',
  supply_depot:
    'M256 23.38L89.844 89.845l-64.9 162.254 14.85 5.943c20.312-50.766 40.62-101.535 60.93-152.304l1.432-3.58L256 40.616l153.844 61.54 1.43 3.58 60.93 152.305 14.853-5.942-64.9-162.254C366.77 67.69 311.386 45.534 256 23.38zm0 36.624l-139.996 55.998L72.8 224h.2v263h78V329h-39v-18h297v176h30V224h.2c-14.402-36-28.802-72-43.204-107.998L256 60.004zM151 135h210v114H151V135zm23.563 18L199 201.873V153h-24.438zM313 153v48.873L337.438 153H313zm-144 29.127V231h24.438L169 182.127zm174 0L318.562 231H343v-48.873zm-98.73 18.69c-1.207-.02-2.31.02-3.288.128-2.823.31-10.76 3.708-16.86 7.3-2.796 1.645-5.23 3.22-7.122 4.484V231h78v-16.97c-4.193-1.675-10.334-4.02-17.578-6.368-11.206-3.63-24.71-6.71-33.152-6.846zM160 263h192v18H160v-18zm15.16 66L208 389.205 240.84 329h-65.68zm144 0L352 389.205 384.84 329h-65.68zM169 355.295v105.41L197.748 408 169 355.295zm78 0L218.252 408 247 460.705v-105.41zm66 0v105.41L341.748 408 313 355.295zm78 0L362.252 408 391 460.705v-105.41zm-183 71.5L175.16 487h65.68L208 426.795zm144 0L319.16 487h65.68L352 426.795z',
}

/** Capital crown stays a light stroke glyph (Tabler Icons, MIT). */
const CROWN_PATH = 'M12 6l4 6l5 -4l-2 10h-14l-2 -10l5 4l4 -6'

/** Neutral fill/stroke for infrastructure without a controlling player. */
const NEUTRAL_MARKER_FILL = '#efe6d0'
const MARKER_CASING_COLOR = '#30291f'

const INFRASTRUCTURE_LABEL_KEYS: Record<Infrastructure['type'], MessageKey> = {
  mill: 'infrastructure.mill',
  supply_depot: 'infrastructure.supply_depot',
  castle: 'infrastructure.castle',
  village: 'infrastructure.village',
}

interface InfrastructureMarkerProps {
  infrastructure: Infrastructure
  x: number
  y: number
  isCapital: boolean
  scale: number
  ownerColor?: string | null
  opacity?: number
  variant?: 'normal' | 'winterGhost'
}

export function TablerMarkerPaths({
  paths,
  fill = 'none',
}: {
  paths: string[]
  fill?: string
}) {
  return (
    <>
      {paths.map((d) => (
        <path key={d} d={d} fill={fill} strokeLinecap="round" strokeLinejoin="round" />
      ))}
    </>
  )
}

export function InfrastructureMarker({
  infrastructure,
  x,
  y,
  isCapital,
  scale,
  ownerColor = null,
  opacity = 1,
  variant = 'normal',
}: InfrastructureMarkerProps) {
  const { t } = useLanguage()
  const label = `${t(INFRASTRUCTURE_LABEL_KEYS[infrastructure.type])} · ${t('app.level', { level: infrastructure.level })}${isCapital ? ` · ${t('app.capital')}` : ''}`
  const glyph = MARKER_GLYPHS[infrastructure.type]
  const fill = ownerColor ?? NEUTRAL_MARKER_FILL

  return (
    <g
      transform={`translate(${x} ${y}) scale(${scale})`}
      data-winter-ghost={variant === 'winterGhost' ? 'true' : undefined}
      opacity={opacity}
      pointerEvents="none"
    >
      <title>{label}</title>
      {variant === 'winterGhost' ? (
        <>
          {/* Enlarged disc fully contains the 26x26 glyph footprint. */}
          <circle
            cx="0"
            cy="0"
            r="18.5"
            fill={fill}
            stroke={INTENT_OUTLINE_COLOR}
            strokeWidth="2"
          />
          <g transform="translate(-13 -13) scale(0.05078125)">
            <path d={glyph} fill="#fff8e7" />
            <path d={glyph} fill="none" stroke={INTENT_OUTLINE_COLOR} strokeWidth={10} />
          </g>
        </>
      ) : (
        <g transform="translate(-13 -13) scale(0.05078125)">
          {/* Light halo keeps the glyph readable on any terrain fill. */}
          <path
            d={glyph}
            fill="#fff8e7"
            stroke="#fff8e7"
            strokeWidth={20}
            opacity={0.9}
          />
          {/* Owner color fills the building, dark casing defines its shape. */}
          <path d={glyph} fill={fill} />
          <path d={glyph} fill="none" stroke={MARKER_CASING_COLOR} strokeWidth={8} />
        </g>
      )}
      {isCapital && (
        <g
          data-capital-marker="true"
          transform="translate(0 -20) scale(0.5)"
          pointerEvents="none"
        >
          <g stroke="#fff8e7" strokeWidth={4} opacity={0.85}>
            <TablerMarkerPaths paths={[CROWN_PATH]} />
          </g>
          <g stroke="#815f1e" strokeWidth={2}>
            <TablerMarkerPaths paths={[CROWN_PATH]} />
          </g>
        </g>
      )}
      {infrastructure.level > 1 && (
        <text
          x="14"
          y="-9"
          fill="#4e3828"
          fontSize="10"
          fontWeight="700"
          textAnchor="middle"
        >
          {infrastructure.level}
        </text>
      )}
    </g>
  )
}

export function NobleMarker({
  noble,
  x,
  y,
  color,
  scale,
}: {
  noble: Noble
  x: number
  y: number
  color: string
  scale: number
}) {
  const { t } = useLanguage()
  const prisoner = noble.status !== 'free'
  return (
    <g transform={`translate(${x} ${y}) scale(${scale})`} pointerEvents="none">
      <title>{`${noble.name} (${noble.id})${prisoner ? ` · ${t(`orders.nobleStatus.${noble.status}` as MessageKey)}` : ''}`}</title>
      <path
        d="M0-8L8 0L0 8L-8 0Z"
        fill={color}
        stroke={prisoner ? '#8d321e' : '#815f1e'}
        strokeWidth="1.5"
      />
      <circle cx="0" cy="0" r="2" fill="#fff3c4" />
      {prisoner && (
        <circle cx="0" cy="0" r="4.5" fill="none" stroke="#8d321e" strokeWidth="1.5" />
      )}
    </g>
  )
}

export function OwnershipBadge({
  ownerId,
  x,
  y,
  color,
  scale,
  label,
}: {
  ownerId: string
  x: number
  y: number
  color: string
  scale: number
  label: string
}) {
  return (
    <g
      data-ownership-badge={ownerId}
      transform={`translate(${x} ${y}) scale(${scale})`}
      pointerEvents="none"
    >
      <title>{label}</title>
      <path
        d={OWNERSHIP_SHIELD_PATH}
        fill="#fff8e7"
        stroke="#fff8e7"
        strokeWidth={5}
        opacity={0.9}
      />
      <path d={OWNERSHIP_SHIELD_PATH} fill={color} />
      <path
        d={OWNERSHIP_SHIELD_PATH}
        fill="none"
        stroke={MARKER_CASING_COLOR}
        strokeWidth={1.6}
      />
    </g>
  )
}

export function IntentBadge({
  x,
  y,
  symbol,
  turnLabel,
  color,
  scale,
  isDraft,
}: {
  x: number
  y: number
  symbol: string
  turnLabel: string
  color: string
  scale: number
  isDraft: boolean
}) {
  const foreground = isDraft ? '#30291f' : '#fff8e7'

  return (
    <g transform={`translate(${x} ${y})`}>
      <circle
        cx="0"
        cy="0"
        r={8 * scale}
        fill={color}
        stroke="#fff8e7"
        strokeWidth={1.5 * scale}
      />
      <text
        x="0"
        y="0"
        fill={foreground}
        fontSize={9 * scale}
        fontWeight="800"
        textAnchor="middle"
        dominantBaseline="central"
      >
        {symbol}
      </text>
      <text
        x={11 * scale}
        y={-1 * scale}
        fill={isDraft ? '#30291f' : color}
        fontSize={9 * scale}
        fontWeight="700"
        textAnchor="middle"
        stroke="#fff8e7"
        strokeWidth={2.5 * scale}
        paintOrder="stroke"
      >
        {turnLabel}
      </text>
    </g>
  )
}

export function WinterBadge({
  x,
  y,
  label,
  detail,
  title,
  color,
  scale,
}: {
  x: number
  y: number
  label: string
  detail?: string
  title: string
  color: string
  scale: number
}) {
  return (
    <g transform={`translate(${x} ${y})`} pointerEvents="none">
      <title>{title}</title>
      <circle
        cx="0"
        cy="0"
        r={9 * scale}
        fill={color}
        fillOpacity="0.48"
        stroke="#fff8e7"
        strokeWidth={2 * scale}
      />
      <text
        x="0"
        y="0"
        fill="#30291f"
        fontSize={label.length > 2 ? 6.5 * scale : 8 * scale}
        fontWeight="800"
        textAnchor="middle"
        dominantBaseline="central"
      >
        {label}
      </text>
      {detail && (
        <text
          x={13 * scale}
          y={-1 * scale}
          fill="#30291f"
          fontSize={8 * scale}
          fontWeight="800"
          textAnchor="start"
          stroke="#fff8e7"
          strokeWidth={2.5 * scale}
          paintOrder="stroke"
        >
          {detail}
        </text>
      )}
    </g>
  )
}

export function WinterMarkerTriangle({
  x,
  y,
  line,
  title,
  scale,
  color,
  dataTag,
}: {
  x: number
  y: number
  line: number
  title: string
  scale: number
  color: string
  dataTag: 'error' | 'warning'
}) {
  const size = 8 * scale
  return (
    <g transform={`translate(${x} ${y})`} pointerEvents="none">
      <title>{title}</title>
      <path
        data-winter-error={dataTag === 'error' ? 'true' : undefined}
        data-winter-warning={dataTag === 'warning' ? 'true' : undefined}
        d={`M 0 ${-size} L ${size} ${size} L ${-size} ${size} Z`}
        fill={color}
        stroke={INTENT_OUTLINE_COLOR}
        strokeWidth={1.5 * scale}
        strokeLinejoin="round"
      />
      <text
        x="0"
        y={3 * scale}
        fill="#fff8e7"
        fontSize={7 * scale}
        fontWeight="800"
        textAnchor="middle"
      >
        {line}
      </text>
    </g>
  )
}

export function WinterTransferArrow({
  from,
  to,
  color,
  amount,
  title,
  scale,
}: {
  from: Point
  to: Point
  color: string
  amount?: number
  title: string
  scale: number
}) {
  const dx = to[0] - from[0]
  const dy = to[1] - from[1]
  const distance = Math.hypot(dx, dy)
  if (distance === 0) return null

  const ux = dx / distance
  const uy = dy / distance
  const start = [from[0] + ux * 12 * scale, from[1] + uy * 12 * scale] as Point
  const tip = [to[0] - ux * 13 * scale, to[1] - uy * 13 * scale] as Point
  const base = [tip[0] - ux * 9 * scale, tip[1] - uy * 9 * scale] as Point
  const sideX = -uy * 4 * scale
  const sideY = ux * 4 * scale
  const badge = [(from[0] + to[0]) / 2, (from[1] + to[1]) / 2] as Point

  return (
    <g pointerEvents="none">
      <title>{title}</title>
      <line
        x1={start[0]}
        y1={start[1]}
        x2={base[0]}
        y2={base[1]}
        stroke={color}
        strokeOpacity="0.58"
        strokeWidth={2.5 * scale}
        strokeDasharray={`${5 * scale} ${3 * scale}`}
        strokeLinecap="round"
      />
      <path
        d={`M ${tip[0]} ${tip[1]} L ${base[0] + sideX} ${base[1] + sideY} L ${base[0] - sideX} ${base[1] - sideY} Z`}
        fill={color}
        fillOpacity="0.58"
        stroke={INTENT_OUTLINE_COLOR}
        strokeOpacity="0.45"
        strokeWidth={1.2 * scale}
      />
      <WinterBadge
        x={badge[0]}
        y={badge[1] - 5 * scale}
        label="G"
        detail={amount === undefined ? undefined : String(amount)}
        title={title}
        color={color}
        scale={scale}
      />
    </g>
  )
}

export function winterReasonKey(reason: string): MessageKey {
  return (reason.startsWith('error.') ? reason : `reports.reason.${reason}`) as MessageKey
}

export function winterReasonText(t: Translate, intention: WinterIntention): string {
  if (intention.message) return intention.message
  return intention.reason
    ? t(winterReasonKey(intention.reason), intention.reasonValues)
    : intention.label
}

export function winterMarkerTitle(t: Translate, intention: WinterIntention): string {
  return t('error.line', {
    line: intention.line,
    message: winterReasonText(t, intention),
  })
}

/**
 * Inline game-icons.net glyph: one filled path recolored at render time,
 * optionally outlined (the liseré) directly on the path edges.
 */
export function GameIconGlyph({
  glyph,
  x,
  y,
  size,
  fill,
  stroke,
  strokeWidth = 0,
  opacity,
  rotation = 0,
}: {
  glyph: GameIconGlyphSpec
  x: number
  y: number
  size: number
  fill: string
  stroke?: string
  strokeWidth?: number
  opacity: number
  rotation?: number
}) {
  return (
    <svg
      x={x}
      y={y}
      width={size}
      height={size}
      viewBox="0 0 512 512"
      opacity={opacity}
      transform={`rotate(${rotation} ${x + size / 2} ${y + size / 2})`}
      pointerEvents="none"
    >
      <g transform={glyph.transform}>
        <path
          d={glyph.path}
          fill={fill}
          stroke={stroke ?? 'none'}
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  )
}
