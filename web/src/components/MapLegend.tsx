import { BorderGlyph, TerrainGlyph } from '@/components/MapDecorations'
import { ArmsField, BannerGlyph, NoblePawn } from '@/components/MapHeraldry'
import { GameIconGlyph } from '@/components/MapMarkers'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useLanguage } from '@/i18n/LanguageContext'
import { GAME_ICON_GLYPHS } from '@/lib/game-icon-glyphs'
import {
  BANNER_HEIGHT,
  BANNER_PATH,
  BANNER_WIDTH,
  SHIELD_HEIGHT,
  SHIELD_PATH,
  SHIELD_WIDTH,
  type CrownKind,
  type ResolvedArms,
} from '@/lib/heraldry'
import type { ImpassableBorderKind } from '@/lib/map-borders'
import type { NobleAttributes } from '@/lib/noble-attributes'
import { HERALDIC_COLORS } from '@/lib/region-color'
import type { MessageKey } from '@/i18n/messages'
import type { NobleStatus, Terrain } from '@/types'
import {
  MAP_HALO,
  MAP_INK,
  MAP_PAPER_FILL,
  MAP_PARCHMENT,
  METAL_COLORS,
  PLAYER_RED,
  RIVER_BLUE,
  SHIELD_NUMBER,
} from '@/lib/map-palette'

export const TERRAIN_LABEL_KEYS: Record<Terrain, MessageKey> = {
  plain: 'terrain.plain',
  forest: 'terrain.forest',
  hill: 'terrain.hill',
  mountain: 'terrain.mountain',
  swamp: 'terrain.swamp',
}

const TERRAIN_ORDER: Terrain[] = ['plain', 'forest', 'hill', 'mountain', 'swamp']

/** Sample owner color shown on legend settlement glyphs. */
const LEGEND_SAMPLE_OWNER = PLAYER_RED
const LEGEND_CASING = MAP_INK

/**
 * Heater-shield silhouette shared by the map ownership badges and this
 * legend swatch.
 */
export const OWNERSHIP_SHIELD_PATH =
  'M -10 -12 L 10 -12 L 10 -2 C 10 5 5.5 10.5 0 13 C -5.5 10.5 -10 5 -10 -2 Z'

// Map marker artwork from game-icons.net (CC BY 3.0, icons by Delapouite):
// https://game-icons.net/1x1/delapouite/castle.html
// https://game-icons.net/1x1/delapouite/village.html
const LEGEND_MARKER_PATHS: Record<'castle' | 'village', string> = {
  castle:
    'M255.95 27.11L180.6 107.614l150.7 1.168-75.35-81.674h-.003zM25 109.895v68.01l19.412 25.99h71.06l19.528-26v-68h-14v15.995h-18v-15.994H89v15.995H71v-15.994H57v15.995H39v-15.994H25zm352 0v68l19.527 26h71.06L487 177.906v-68.01h-14v15.995h-18v-15.994h-14v15.995h-18v-15.994h-14v15.995h-18v-15.994h-14zm-176 15.877V260.89h110V126.63l-110-.857zm55 20.118c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zM41 221.897V484.89h78V221.897H41zm352 0V484.89h78V221.897h-78zM56 241.89c4 0 8 4 8 12v32H48v-32c0-8 4-12 8-12zm400 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12zm-303 37v23h-16v183h87v-55c0-24 16-36 32-36s32 12 32 36v55h87v-183h-16v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14v23h-18v-23h-14zm-49 43c4 0 8 4 8 12v32H96v-32c0-8 4-12 8-12zm72 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm80 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm80 0c8 0 16 4 16 12v32h-32v-32c0-8 8-12 16-12zm72 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12zm-352 64c4 0 8 4 8 12v32H48v-32c0-8 4-12 8-12zm400 0c4 0 8 4 8 12v32h-16v-32c0-8 4-12 8-12z',
  village:
    'M109.902 35.87l-71.14 59.284h142.28l-71.14-59.285zm288 32l-71.14 59.284h142.28l-71.14-59.285zM228.73 84.403l-108.9 90.75h217.8l-108.9-90.75zm-173.828 28.75v62h36.81l73.19-60.992v-1.008h-110zm23 14h16v18h-16v-18zm265 18v10.963l23 19.166v-16.13h16v18h-13.756l.104.087 19.098 15.914h-44.446v14h78v-39h18v39h14v-62h-110zm-194.345 48v20.08l24.095-20.08h-24.095zm28.158 0l105.1 87.582 27.087-22.574v-65.008H176.715zm74.683 14h35.735v34h-35.735v-34zm-76.714 7.74L30.37 335.153H319l-144.314-120.26zm198.046 13.51l-76.857 64.047 32.043 26.704H481.63l-108.9-90.75zm-23.214 108.75l.103.086 19.095 15.914h-72.248v77.467h60.435v-63.466h50v63.467h46v-93.466H349.516zm-278.614 16V476.13h126v-76.976h50v76.977h31.565V353.155H70.902zm30 30h50v50h-50v-50z',
}

/** Settlement glyph with owner fill, halo and casing — mirrors the map. */
function LegendSettlement({ type }: { type: 'castle' | 'village' }) {
  return (
    <svg className="size-3.5 shrink-0" viewBox="0 0 512 512" aria-hidden="true">
      <path
        d={LEGEND_MARKER_PATHS[type]}
        fill={MAP_PARCHMENT}
        stroke={MAP_PARCHMENT}
        strokeWidth={20}
        opacity={0.9}
      />
      <path d={LEGEND_MARKER_PATHS[type]} fill={LEGEND_SAMPLE_OWNER} />
      <path
        d={LEGEND_MARKER_PATHS[type]}
        fill="none"
        stroke={LEGEND_CASING}
        strokeWidth={8}
      />
    </svg>
  )
}

/** Terrain sample: the engraved symbol on the map paper. */
function TerrainSwatch({ terrain }: { terrain: Terrain }) {
  return (
    <svg
      className="size-4 shrink-0"
      viewBox="-6.5 -6.5 13 13"
      aria-hidden="true"
      data-legend-terrain={terrain}
    >
      <rect x="-6.5" y="-6.5" width="13" height="13" fill={MAP_PAPER_FILL} />
      <TerrainGlyph terrain={terrain} />
    </svg>
  )
}

/** Border sample on the map paper: dotted passable line or icon series. */
function BorderSwatch({ kind }: { kind: 'passable' | ImpassableBorderKind }) {
  return (
    <svg
      className="h-4 w-6 shrink-0"
      viewBox="-12 -8 24 16"
      aria-hidden="true"
      data-legend-border={kind}
    >
      <rect x="-12" y="-8" width="24" height="16" fill={MAP_PAPER_FILL} />
      {kind === 'passable' ? (
        <line
          x1="-10"
          y1="0"
          x2="10"
          y2="0"
          stroke={MAP_INK}
          strokeOpacity="0.75"
          strokeWidth="1.7"
          strokeDasharray="0.1 4.5"
          strokeLinecap="round"
        />
      ) : kind === 'peaks' ? (
        <GameIconGlyph
          glyph={GAME_ICON_GLYPHS['peaks']}
          x={-7}
          y={-7}
          size={14}
          fill={MAP_INK}
          stroke={MAP_HALO}
          strokeWidth={20}
          opacity={1}
        />
      ) : (
        <g transform="scale(0.85)">
          <BorderGlyph kind={kind} color={kind === 'river' ? RIVER_BLUE : MAP_INK} />
        </g>
      )}
    </svg>
  )
}

const BORDER_LEGEND: Array<{
  kind: 'passable' | ImpassableBorderKind
  label: MessageKey
}> = [
  { kind: 'passable', label: 'legend.borderPassable' },
  { kind: 'peaks', label: 'legend.borderPeaks' },
  { kind: 'marsh', label: 'legend.borderMarsh' },
  { kind: 'cliff', label: 'legend.borderCliff' },
]

const LEGEND_ARMS: ResolvedArms = {
  field: PLAYER_RED,
  charge: { kind: 'bend', metal: METAL_COLORS.or },
}

/** Banner sample in the sample arms, crowned or marked with a fief trigram. */
function LegendBanner({ crown, attachedTo }: { crown?: CrownKind; attachedTo?: string }) {
  const id = `legend-banner-${crown ?? attachedTo ?? 'plain'}`
  return (
    <svg
      className="h-5 w-6 shrink-0"
      viewBox="-3 -24 20 26"
      aria-hidden="true"
      data-legend-banner={crown ?? attachedTo ?? 'plain'}
    >
      <defs>
        <symbol id={`${id}-flag`} overflow="visible">
          <ArmsField
            arms={LEGEND_ARMS}
            outline={BANNER_PATH}
            clipId={`${id}-clip`}
            box={[0, 0, BANNER_WIDTH, BANNER_HEIGHT]}
          />
        </symbol>
      </defs>
      <BannerGlyph
        owner="legend"
        flagHref={`#${id}-flag`}
        crown={crown}
        attachedTo={attachedTo}
      />
    </svg>
  )
}

/** Army sample: the shield of the sample arms, with its strength. */
function LegendShield() {
  return (
    <svg
      className="h-5 w-6 shrink-0"
      viewBox="-10 -11 20 22"
      aria-hidden="true"
      data-legend-shield
    >
      <ArmsField
        arms={LEGEND_ARMS}
        outline={SHIELD_PATH}
        clipId="legend-shield-clip"
        box={[-SHIELD_WIDTH / 2, -SHIELD_HEIGHT / 2, SHIELD_WIDTH, SHIELD_HEIGHT]}
      />
      <text
        fill={SHIELD_NUMBER}
        fontSize={10}
        fontWeight="700"
        textAnchor="middle"
        dominantBaseline="central"
        stroke={MAP_INK}
        strokeWidth={2.4}
        strokeLinejoin="round"
        paintOrder="stroke"
      >
        3
      </text>
    </svg>
  )
}

const NO_ATTRIBUTES: NobleAttributes = { crosier: false, excommunicated: false }

/** Noble pawn samples: statuses, then title attributes. */
const PAWN_LEGEND: Array<{
  key: string
  label: MessageKey
  status?: NobleStatus
  attributes?: NobleAttributes
}> = [
  { key: 'free', label: 'legend.noble' },
  { key: 'hostage', label: 'legend.pawn.hostage', status: 'hostage' },
  { key: 'dungeon', label: 'legend.pawn.dungeon', status: 'dungeon' },
  {
    key: 'excommunicated',
    label: 'legend.pawn.excommunicated',
    attributes: { ...NO_ATTRIBUTES, excommunicated: true },
  },
  {
    key: 'baron',
    label: 'legend.pawn.baron',
    attributes: { ...NO_ATTRIBUTES, crown: 'barony' },
  },
  {
    key: 'count',
    label: 'legend.pawn.count',
    attributes: { ...NO_ATTRIBUTES, crown: 'county' },
  },
  {
    key: 'marquis',
    label: 'legend.pawn.marquis',
    attributes: { ...NO_ATTRIBUTES, crown: 'marquisate' },
  },
  {
    key: 'duke',
    label: 'legend.pawn.duke',
    attributes: { ...NO_ATTRIBUTES, crown: 'duchy' },
  },
  {
    key: 'king',
    label: 'legend.pawn.king',
    attributes: { ...NO_ATTRIBUTES, crown: 'king' },
  },
  {
    key: 'courtesy',
    label: 'legend.pawn.courtesy',
    attributes: { ...NO_ATTRIBUTES, courtesyCrown: 'county' },
  },
  {
    key: 'bishop',
    label: 'legend.pawn.bishop',
    attributes: { ...NO_ATTRIBUTES, crosier: true },
  },
  {
    key: 'cardinal',
    label: 'legend.pawn.cardinal',
    attributes: { ...NO_ATTRIBUTES, robe: 'cardinal' },
  },
  {
    key: 'pope',
    label: 'legend.pawn.pope',
    attributes: { ...NO_ATTRIBUTES, robe: 'pope' },
  },
]

interface MapLegendProps {
  showIntentions?: boolean
  onToggleIntentions?: (show: boolean) => void
  showOwnership?: boolean
  onToggleOwnership?: (show: boolean) => void
  showRegions?: boolean
  onToggleRegions?: (show: boolean) => void
  showCalamities?: boolean
  onToggleCalamities?: (show: boolean) => void
  showCards?: boolean
  onToggleCards?: (show: boolean) => void
}

export function MapLegend({
  showIntentions = true,
  onToggleIntentions,
  showOwnership = true,
  onToggleOwnership,
  showRegions = false,
  onToggleRegions,
  showCalamities = true,
  onToggleCalamities,
  showCards = true,
  onToggleCards,
}: MapLegendProps) {
  const { t } = useLanguage()

  return (
    <Card
      aria-label={t('legend.title')}
      className="w-full border-line bg-paper-50 shadow-panel"
    >
      <CardHeader className="gap-0 pb-2">
        <CardTitle className="text-sm uppercase tracking-[0.16em] text-ink-700">
          {t('legend.title')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-xs text-ink-700">
        {onToggleIntentions && (
          <label className="flex items-center gap-2 rounded-md bg-paper-100 px-2 py-1.5">
            <input
              type="checkbox"
              checked={showIntentions}
              onChange={(event) => onToggleIntentions(event.target.checked)}
              className="size-3.5 shrink-0 accent-brass-700"
            />
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold">{t('legend.intentions')}</span>
              <span className="text-2xs text-ink-500">{t('legend.intentionsHint')}</span>
            </span>
          </label>
        )}
        {onToggleOwnership && (
          <label className="flex items-center gap-2 rounded-md bg-paper-100 px-2 py-1.5">
            <input
              type="checkbox"
              checked={showOwnership}
              onChange={(event) => onToggleOwnership(event.target.checked)}
              className="size-3.5 shrink-0 accent-brass-700"
            />
            <span>{t('legend.ownership')}</span>
          </label>
        )}
        {onToggleRegions && (
          <label className="flex items-center gap-2 rounded-md bg-winter-100 px-2 py-1.5">
            <input
              type="checkbox"
              checked={showRegions}
              onChange={(event) => onToggleRegions(event.target.checked)}
            />
            <span>{t('legend.regions')}</span>
          </label>
        )}
        {onToggleCalamities && (
          <label className="flex items-center gap-2 rounded-md bg-paper-100 px-2 py-1.5">
            <input
              type="checkbox"
              checked={showCalamities}
              onChange={(event) => onToggleCalamities(event.target.checked)}
            />
            <span>{t('legend.calamities')}</span>
          </label>
        )}
        {onToggleCards && (
          <label className="flex items-center gap-2 rounded-md bg-paper-100 px-2 py-1.5">
            <input
              type="checkbox"
              checked={showCards}
              onChange={(event) => onToggleCards(event.target.checked)}
            />
            <span>{t('legend.cards')}</span>
          </label>
        )}
        {onToggleRegions && (
          <div
            className="flex flex-wrap gap-1.5 rounded-md bg-winter-100 px-2 py-1.5"
            data-region-swatches
          >
            {HERALDIC_COLORS.map((color, index) => (
              <span
                key={`region-color-${index}`}
                data-region-color={index}
                className="size-4 rounded-sm border border-winter-700/40"
                style={{ backgroundColor: color }}
              />
            ))}
          </div>
        )}
        <div className="grid grid-cols-2 gap-x-3 gap-y-1.5">
          {TERRAIN_ORDER.map((terrain) => (
            <div key={terrain} className="flex items-center gap-2">
              <TerrainSwatch terrain={terrain} />
              <span>{t(TERRAIN_LABEL_KEYS[terrain])}</span>
            </div>
          ))}
          <div className="flex items-center gap-2">
            <LegendSettlement type="village" />
            <span>{t('legend.village')}</span>
          </div>
          <div className="flex items-center gap-2">
            <LegendSettlement type="castle" />
            <span>{t('legend.castle')}</span>
          </div>
          <div className="flex items-center gap-2">
            <LegendBanner />
            <span>{t('legend.control')}</span>
          </div>
          <div className="flex items-center gap-2">
            <LegendBanner crown="barony" />
            <span>{t('legend.fiefCapital')}</span>
          </div>
          <div className="flex items-center gap-2">
            <LegendBanner attachedTo="ROS" />
            <span>{t('legend.fief')}</span>
          </div>
          <div className="flex items-center gap-2">
            <LegendShield />
            <span>{t('legend.army')}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-6 shrink-0 text-center font-bold text-ink-900">R</span>
            <span>{t('legend.stock')}</span>
          </div>
        </div>
        <div
          className="grid grid-cols-2 gap-x-3 gap-y-1.5 border-t border-line/60 pt-2"
          data-legend-nobles
        >
          {PAWN_LEGEND.map(({ key, label, status, attributes }) => (
            <div key={key} className="flex items-center gap-2" data-legend-pawn={key}>
              <svg className="h-5 w-4 shrink-0" viewBox="-11 -14 18 23" aria-hidden="true">
                <NoblePawn color={PLAYER_RED} status={status} attributes={attributes} />
              </svg>
              <span>{t(label)}</span>
            </div>
          ))}
        </div>
        <div className="grid grid-cols-2 gap-x-3 gap-y-1.5 border-t border-line/60 pt-2">
          {BORDER_LEGEND.map(({ kind, label }) => (
            <div key={kind} className="flex items-center gap-2">
              <BorderSwatch kind={kind} />
              <span>{t(label)}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
