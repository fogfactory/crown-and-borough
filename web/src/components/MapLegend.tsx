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
        fill="#fff8e7"
        stroke="#fff8e7"
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
            <g stroke="#5a7a34" strokeWidth="0.5" opacity="0.22">
              <line x1="1.4" y1="3.6" x2="2.2" y2="2.8" />
              <line x1="2.6" y1="3.6" x2="3.4" y2="2.8" />
              <line x1="3.8" y1="3.6" x2="4.6" y2="2.8" />
            </g>
          </pattern>
        )}
        {terrain === 'forest' && (
          <pattern
            id="legend-terrain-forest"
            width="12"
            height="12"
            patternUnits="userSpaceOnUse"
          >
            <path d="M0 6.8 L2 0.4 L4 6.8 Z" fill="#14291d" opacity="0.25" />
            <path d="M6 11.2 L8 4.8 L10 11.2 Z" fill="#14291d" opacity="0.18" />
          </pattern>
        )}
        {terrain === 'hill' && (
          <pattern
            id="legend-terrain-hill"
            width="14"
            height="7"
            patternUnits="userSpaceOnUse"
          >
            <path
              d="M0 6 Q 3.5 1.2 7 6 T 14 6"
              fill="none"
              stroke="#6b4a30"
              strokeWidth="1.3"
              opacity="0.25"
            />
          </pattern>
        )}
        {terrain === 'mountain' && (
          <pattern
            id="legend-terrain-mountain"
            width="12"
            height="12"
            patternUnits="userSpaceOnUse"
          >
            <path
              d="M0 4.8 L3 0.8 L6 4.8 M6 9.6 L9 5.8 L12 9.6"
              fill="none"
              stroke="#4d565e"
              strokeWidth="1.3"
              opacity="0.28"
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
              opacity="0.28"
            />
            <line
              x1="3.8"
              y1="4.4"
              x2="6.4"
              y2="4.4"
              stroke="#2e5f5a"
              strokeWidth="0.9"
              opacity="0.28"
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
