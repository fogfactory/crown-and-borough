import { IconFocus2, IconMinus, IconPlus } from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'
import { ZOOM_BUTTON_FACTOR } from '@/lib/map-gestures'

export function MapControls({ onZoom }: { onZoom: (zoomFactor: number) => void }) {
  const { t } = useLanguage()

  return (
    <div
      role="group"
      aria-label={t('map.zoomControls')}
      className="absolute bottom-28 right-3 z-10 flex flex-col gap-1.5 lg:bottom-3"
    >
      <button
        type="button"
        aria-label={t('map.zoomIn')}
        title={t('map.zoomIn')}
        className="flex size-9 items-center justify-center rounded-lg border border-line bg-paper-50 text-ink-700 shadow-md transition hover:bg-paper-100 hover:text-ink-900 focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none"
        onClick={() => onZoom(ZOOM_BUTTON_FACTOR)}
      >
        <IconPlus aria-hidden="true" className="size-4" />
      </button>
      <button
        type="button"
        aria-label={t('map.zoomOut')}
        title={t('map.zoomOut')}
        className="flex size-9 items-center justify-center rounded-lg border border-line bg-paper-50 text-ink-700 shadow-md transition hover:bg-paper-100 hover:text-ink-900 focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none"
        onClick={() => onZoom(1 / ZOOM_BUTTON_FACTOR)}
      >
        <IconMinus aria-hidden="true" className="size-4" />
      </button>
      <button
        type="button"
        aria-label={t('map.recenter')}
        title={t('map.recenter')}
        className="flex size-9 items-center justify-center rounded-lg border border-line bg-paper-50 text-ink-700 shadow-md transition hover:bg-paper-100 hover:text-ink-900 focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none"
        onClick={() => onZoom(0)}
      >
        <IconFocus2 aria-hidden="true" className="size-4" />
      </button>
    </div>
  )
}
