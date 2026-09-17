import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { Terrain } from '@/types'

export const TERRAIN_LABEL_KEYS: Record<Terrain, MessageKey> = {
  plain: 'terrain.plain',
  forest: 'terrain.forest',
  hill: 'terrain.hill',
  mountain: 'terrain.mountain',
  swamp: 'terrain.swamp',
}

export const TERRAIN_COLORS: Record<Terrain, string> = {
  plain: '#b8d99a',
  forest: '#3f7854',
  hill: '#ad8565',
  mountain: '#89929a',
  swamp: '#66a6a0',
}

const TERRAIN_ORDER: Terrain[] = ['plain', 'forest', 'hill', 'mountain', 'swamp']

/** Sample owner color shown on legend settlement glyphs. */
const LEGEND_SAMPLE_OWNER = '#a84632'
const LEGEND_CASING = '#30291f'

// Same Tabler path data as the map markers (https://tabler.io/icons, MIT).
const LEGEND_MARKER_PATHS: Record<'castle' | 'village', string[]> = {
  castle: [
    'M15 19v-2a3 3 0 0 0 -6 0v2a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1v-14h4v3h3v-3h4v3h3v-3h4v14a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1',
    'M3 11l18 0',
  ],
  village: [
    'M8 9l5 5v7h-5v-4m0 4h-5v-7l5 -5m1 1v-6a1 1 0 0 1 1 -1h10a1 1 0 0 1 1 1v17h-8',
    'M13 7l0 .01',
    'M17 7l0 .01',
    'M17 11l0 .01',
    'M17 15l0 .01',
  ],
}

function LegendPaths({ paths }: { paths: string[] }) {
  return (
    <>
      {paths.map((d) => (
        <path key={d} d={d} strokeLinecap="round" strokeLinejoin="round" />
      ))}
    </>
  )
}

/** Settlement glyph with owner fill, halo and casing — mirrors the map. */
function LegendSettlement({ type }: { type: 'castle' | 'village' }) {
  return (
    <svg className="size-3.5 shrink-0" viewBox="-13 -13 26 26" aria-hidden="true">
      <g transform="translate(-10 -10) scale(0.8333)">
        <g stroke="#fff8e7" strokeWidth={4.5} opacity={0.9} fill="none">
          <LegendPaths paths={LEGEND_MARKER_PATHS[type]} />
        </g>
        <g fill={LEGEND_SAMPLE_OWNER} fillOpacity={0.9} stroke="none">
          <LegendPaths paths={LEGEND_MARKER_PATHS[type]} />
        </g>
        <g stroke={LEGEND_CASING} strokeWidth={1.75} fill="none">
          <LegendPaths paths={LEGEND_MARKER_PATHS[type]} />
        </g>
      </g>
    </svg>
  )
}

/** Terrain swatch with the same texture pattern as the map layer. */
function TerrainSwatch({ terrain }: { terrain: Terrain }) {
  return (
    <svg className="size-3 shrink-0" viewBox="0 0 12 12" aria-hidden="true">
      <rect width="12" height="12" fill={TERRAIN_COLORS[terrain]} />
      <rect fill={`url(#legend-terrain-${terrain})`} width="12" height="12" />
      <defs>
        {terrain === 'plain' && (
          <pattern
            id="legend-terrain-plain"
            width="6"
            height="6"
            patternUnits="userSpaceOnUse"
          >
            <circle cx="1.5" cy="1.5" r="0.8" fill="#5a7a34" opacity="0.4" />
            <circle cx="4.5" cy="4.5" r="0.8" fill="#5a7a34" opacity="0.4" />
          </pattern>
        )}
        {terrain === 'forest' && (
          <pattern
            id="legend-terrain-forest"
            width="6"
            height="6"
            patternUnits="userSpaceOnUse"
          >
            <path d="M0 2.2 L1.2 0.4 L2.4 2.2 Z" fill="#14291d" opacity="0.45" />
            <path d="M3 5.2 L4.2 3.4 L5.4 5.2 Z" fill="#14291d" opacity="0.35" />
          </pattern>
        )}
        {terrain === 'hill' && (
          <pattern
            id="legend-terrain-hill"
            width="7"
            height="5"
            patternUnits="userSpaceOnUse"
          >
            <path
              d="M0 3 Q 1.75 1.4 3.5 3 T 7 3"
              fill="none"
              stroke="#6b4a30"
              strokeWidth="0.9"
              opacity="0.45"
            />
          </pattern>
        )}
        {terrain === 'mountain' && (
          <pattern
            id="legend-terrain-mountain"
            width="6"
            height="6"
            patternUnits="userSpaceOnUse"
          >
            <path
              d="M0 2.6 L1.5 0.8 L3 2.6 M3 5.2 L4.5 3.4 L6 5.2"
              fill="none"
              stroke="#4d565e"
              strokeWidth="0.9"
              opacity="0.5"
            />
          </pattern>
        )}
        {terrain === 'swamp' && (
          <pattern
            id="legend-terrain-swamp"
            width="7"
            height="6"
            patternUnits="userSpaceOnUse"
          >
            <line
              x1="0.6"
              y1="1.8"
              x2="3.2"
              y2="1.8"
              stroke="#2e5f5a"
              strokeWidth="0.9"
              opacity="0.5"
            />
            <line
              x1="3.8"
              y1="4.4"
              x2="6.4"
              y2="4.4"
              stroke="#2e5f5a"
              strokeWidth="0.9"
              opacity="0.5"
            />
          </pattern>
        )}
      </defs>
    </svg>
  )
}

interface MapLegendProps {
  showIntentions?: boolean
  onToggleIntentions?: (show: boolean) => void
}

export function MapLegend({ showIntentions = true, onToggleIntentions }: MapLegendProps) {
  const { t } = useLanguage()

  return (
    <Card
      aria-label={t('legend.title')}
      className="w-full border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]"
    >
      <CardHeader className="gap-0 pb-2">
        <CardTitle className="text-sm uppercase tracking-[0.16em] text-[#594b3c]">
          {t('legend.title')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-xs text-[#594b3c]">
        {onToggleIntentions && (
          <label className="flex items-center gap-2 rounded-md bg-[#f3ead9] px-2 py-1.5">
            <input
              type="checkbox"
              checked={showIntentions}
              onChange={(event) => onToggleIntentions(event.target.checked)}
              className="size-3.5 shrink-0 accent-[#a84632]"
            />
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold">{t('legend.intentions')}</span>
              <span className="text-[10px] text-[#806f57]">
                {t('legend.intentionsHint')}
              </span>
            </span>
          </label>
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
            <span className="size-3 shrink-0 rounded-full border-2 border-[#fff8e7] bg-[#a84632]" />
            <span>{t('legend.army')}</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-10 -10 20 20" aria-hidden="true">
              <path
                d="M0-8L8 0L0 8L-8 0Z"
                fill="#a84632"
                stroke="#815f1e"
                strokeWidth="1.5"
              />
              <circle cx="0" cy="0" r="2" fill="#fff3c4" />
            </svg>
            <span>{t('legend.noble')}</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-10 -10 20 20" aria-hidden="true">
              <path
                d="M0-8L8 0L0 8L-8 0Z"
                fill="#a84632"
                stroke="#8d321e"
                strokeWidth="1.5"
              />
              <circle
                cx="0"
                cy="0"
                r="4.5"
                fill="none"
                stroke="#8d321e"
                strokeWidth="1.5"
              />
            </svg>
            <span>{t('legend.prisoner')}</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="0 0 16 16" aria-hidden="true">
              <rect
                x="3"
                y="3"
                width="10"
                height="10"
                fill="none"
                stroke={LEGEND_CASING}
                strokeWidth="5"
                opacity="0.55"
              />
              <rect
                x="3"
                y="3"
                width="10"
                height="10"
                fill="none"
                stroke="#a84632"
                strokeWidth="3"
              />
            </svg>
            <span>{t('legend.control')}</span>
          </div>
        </div>
        <p className="border-t border-[#b7a786]/60 pt-2 leading-relaxed">
          {t('legend.passable')}
        </p>
      </CardContent>
    </Card>
  )
}
