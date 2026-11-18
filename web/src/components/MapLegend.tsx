import type { ReactNode } from 'react'

import { BorderGlyph, TerrainGlyph } from '@/components/MapDecorations'
import { ArmyIcon, ControlShieldIcon } from '@/components/ArmsIcons'
import { NoblePawn } from '@/components/MapHeraldry'
import {
  GameIconGlyph,
  GlyphPath,
  MARKER_GLYPHS,
  RuinClipPath,
  useRuinClipId,
} from '@/components/MapMarkers'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useLanguage } from '@/i18n/LanguageContext'
import { GAME_ICON_GLYPHS } from '@/lib/game-icon-glyphs'
import { type CrownKind, type ResolvedArms } from '@/lib/heraldry'
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

/** Settlement glyph with owner fill, halo and casing — mirrors the map. */
function LegendSettlement({
  type,
  ruined = false,
}: {
  type: 'castle' | 'village'
  ruined?: boolean
}) {
  const ruinClipId = useRuinClipId(ruined)
  return (
    <svg className="size-3.5 shrink-0" viewBox="0 0 512 512" aria-hidden="true">
      <RuinClipPath id={ruinClipId} kind={type} />
      <GlyphPath
        d={MARKER_GLYPHS[type]}
        ruinClipId={ruinClipId}
        fill={MAP_PARCHMENT}
        stroke={MAP_PARCHMENT}
        strokeWidth={20}
        opacity={0.9}
      />
      <GlyphPath
        d={MARKER_GLYPHS[type]}
        ruinClipId={ruinClipId}
        fill={LEGEND_SAMPLE_OWNER}
      />
      <GlyphPath
        d={MARKER_GLYPHS[type]}
        ruinClipId={ruinClipId}
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
function BorderSwatch({ kind }: { kind: 'passable' | 'corner' | ImpassableBorderKind }) {
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
      ) : kind === 'corner' ? (
        <g transform="scale(0.85)">
          <BorderGlyph kind="cliff" color={MAP_INK} />
        </g>
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
  kind: 'passable' | 'corner' | ImpassableBorderKind
  label: MessageKey
}> = [
  { kind: 'passable', label: 'legend.borderPassable' },
  { kind: 'peaks', label: 'legend.borderPeaks' },
  { kind: 'marsh', label: 'legend.borderMarsh' },
  { kind: 'cliff', label: 'legend.borderCliff' },
  { kind: 'corner', label: 'legend.borderCorner' },
]

const LEGEND_ARMS: ResolvedArms = {
  field: PLAYER_RED,
  charge: { kind: 'bend', metal: METAL_COLORS.or },
}

/** Shield sample in the sample arms: plain, greyed, crowned or marked with a fief trigram. */
function LegendControl({
  crown,
  attachedTo,
  grey,
}: {
  crown?: CrownKind
  attachedTo?: string
  grey?: boolean
}) {
  const key = grey ? 'occupied' : (crown ?? attachedTo ?? 'plain')
  return (
    <span data-legend-control={key} className="contents">
      <ControlShieldIcon
        id={`legend-control-${key}`}
        arms={LEGEND_ARMS}
        crown={crown}
        attachedTo={attachedTo}
        grey={grey}
      />
    </span>
  )
}

/** Army sample: a standard-bearer of the sample arms, with its strength. */
function LegendArmy() {
  return (
    <span data-legend-army className="contents">
      <ArmyIcon id="legend-army" arms={LEGEND_ARMS} size={3} />
    </span>
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

/** Collapsible legend section; the browser handles the expand/collapse. */
function LegendSection({
  title,
  defaultOpen = true,
  bare = false,
  children,
}: {
  title: string
  defaultOpen?: boolean
  /** Render only the content: the surrounding panel already has the title. */
  bare?: boolean
  children: ReactNode
}) {
  if (bare) return <div className="space-y-2">{children}</div>
  return (
    <details
      open={defaultOpen}
      className="group break-inside-avoid rounded-md border border-line/60 bg-paper-50"
    >
      <summary className="flex cursor-pointer select-none items-center justify-between px-2 py-1.5 text-2xs font-semibold uppercase tracking-[0.14em] text-ink-700">
        {title}
        <span aria-hidden="true" className="transition-transform group-open:rotate-90">
          ›
        </span>
      </summary>
      <div className="space-y-2 px-2 pb-2">{children}</div>
    </details>
  )
}

export type LegendSectionId = 'display' | 'map' | 'nobles' | 'borders'

export const LEGEND_SECTIONS: LegendSectionId[] = ['display', 'map', 'nobles', 'borders']

/** Icon of each legend's toggle button, drawn in a 24-unit stroke box. */
export const LEGEND_TOGGLE_ICONS: Record<LegendSectionId, ReactNode> = {
  display: (
    <>
      <path d="M4 7h10" />
      <path d="M18 7h2" />
      <circle cx="16" cy="7" r="2" />
      <path d="M4 17h2" />
      <path d="M10 17h10" />
      <circle cx="8" cy="17" r="2" />
    </>
  ),
  map: (
    <>
      <path d="M3 19l6-11 4 7 3-4 5 8z" />
    </>
  ),
  nobles: (
    <>
      <circle cx="12" cy="7" r="3" />
      <path d="M6 20c0-4 3-6 6-6s6 2 6 6z" />
    </>
  ),
  borders: (
    <>
      <path d="M3 12h2M8 12h2M13 12h2M18 12h3" />
      <path d="M12 4v4M12 16v4" />
    </>
  ),
}

interface MapLegendProps {
  /** Render a single section as its own compact panel. */
  section?: LegendSectionId
  showIntentions?: boolean
  onToggleIntentions?: (show: boolean) => void
  showTerritoryColors?: boolean
  onToggleTerritoryColors?: (show: boolean) => void
  showRegions?: boolean
  onToggleRegions?: (show: boolean) => void
  showCalamities?: boolean
  onToggleCalamities?: (show: boolean) => void
  showCards?: boolean
  onToggleCards?: (show: boolean) => void
}

export function MapLegend({
  section,
  showIntentions = true,
  onToggleIntentions,
  showTerritoryColors = false,
  onToggleTerritoryColors,
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
      aria-label={section ? t(`legend.section.${section}`) : t('legend.title')}
      className="w-full border-line bg-paper-50 shadow-panel"
    >
      <CardHeader className="gap-0 pb-2">
        <CardTitle className="text-sm uppercase tracking-[0.16em] text-ink-700">
          {section ? t(`legend.section.${section}`) : t('legend.title')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-xs text-ink-700">
        {!section && <p className="text-2xs text-ink-500">{t('app.mapInstructions')}</p>}
        <div
          className={
            section
              ? 'grid gap-2'
              : 'grid grid-cols-[repeat(auto-fit,minmax(14rem,1fr))] items-start gap-2'
          }
        >
          {(!section || section === 'display') &&
            (onToggleIntentions ||
              onToggleTerritoryColors ||
              onToggleRegions ||
              onToggleCalamities ||
              onToggleCards) && (
              <LegendSection title={t('legend.section.display')} bare={!!section}>
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
                      <span className="text-2xs text-ink-500">
                        {t('legend.intentionsHint')}
                      </span>
                    </span>
                  </label>
                )}
                {onToggleTerritoryColors && (
                  <label className="flex items-center gap-2 rounded-md bg-paper-100 px-2 py-1.5">
                    <input
                      type="checkbox"
                      checked={showTerritoryColors}
                      onChange={(event) => onToggleTerritoryColors(event.target.checked)}
                      className="size-3.5 shrink-0 accent-brass-700"
                    />
                    <span>{t('legend.territoryColors')}</span>
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
              </LegendSection>
            )}
          {(!section || section === 'map') && (
            <LegendSection title={t('legend.section.map')} bare={!!section}>
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
                  <LegendSettlement type="village" ruined />
                  <span>{t('legend.villageRuin')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <LegendSettlement type="castle" />
                  <span>{t('legend.castle')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <LegendSettlement type="castle" ruined />
                  <span>{t('legend.castleRuin')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <LegendControl />
                  <span>{t('legend.control')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <LegendControl grey />
                  <span>{t('legend.controlOccupied')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <LegendControl crown="barony" />
                  <span>{t('legend.fiefCapital')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <LegendControl attachedTo="ROS" />
                  <span>{t('legend.fief')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <LegendArmy />
                  <span>{t('legend.army')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="w-6 shrink-0 text-center font-bold text-ink-900">
                    R
                  </span>
                  <span>{t('legend.stock')}</span>
                </div>
              </div>
            </LegendSection>
          )}
          {(!section || section === 'nobles') && (
            <LegendSection
              title={t('legend.section.nobles')}
              defaultOpen={false}
              bare={!!section}
            >
              <div className="grid grid-cols-2 gap-x-3 gap-y-1.5" data-legend-nobles>
                {PAWN_LEGEND.map(({ key, label, status, attributes }) => (
                  <div
                    key={key}
                    className="flex items-center gap-2"
                    data-legend-pawn={key}
                  >
                    <svg
                      className="h-5 w-4 shrink-0"
                      viewBox="-11 -14 18 23"
                      aria-hidden="true"
                    >
                      <NoblePawn
                        color={PLAYER_RED}
                        status={status}
                        attributes={attributes}
                      />
                    </svg>
                    <span>{t(label)}</span>
                  </div>
                ))}
              </div>
            </LegendSection>
          )}
          {(!section || section === 'borders') && (
            <LegendSection title={t('legend.section.borders')} bare={!!section}>
              <div className="grid grid-cols-2 gap-x-3 gap-y-1.5">
                {BORDER_LEGEND.map(({ kind, label }) => (
                  <div key={kind} className="flex items-center gap-2">
                    <BorderSwatch kind={kind} />
                    <span>{t(label)}</span>
                  </div>
                ))}
              </div>
            </LegendSection>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
