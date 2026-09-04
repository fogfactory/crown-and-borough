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
              <span
                className="size-3 shrink-0 rounded-full border border-[#594b3c]/30"
                style={{ backgroundColor: TERRAIN_COLORS[terrain] }}
              />
              <span>{t(TERRAIN_LABEL_KEYS[terrain])}</span>
            </div>
          ))}
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-10 -10 20 20" aria-hidden="true">
              <path
                d="M-9 8V-1L0-10L9-1V8Z"
                fill="#fff8e7"
                stroke="#6b4c28"
                strokeWidth="1.5"
              />
              <rect x="-4" y="1" width="8" height="7" fill="#b7834e" />
            </svg>
            <span>{t('legend.village')}</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-11 -11 22 22" aria-hidden="true">
              <path
                d="M-9 9V-3H-5V-9H-1V-3H3V-9H7V-3H10V9Z"
                fill="#efe6d0"
                stroke="#5f4936"
                strokeWidth="1.5"
              />
            </svg>
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
