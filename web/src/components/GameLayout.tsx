import type { ReactNode } from 'react'

import { PanelSheet } from '@/components/ui/panel-sheet'
import { useLanguage } from '@/i18n/LanguageContext'
import { cn } from '@/lib/utils'

interface GameLayoutProps {
  /** The interactive map (already includes its own overlays). */
  map: ReactNode
  /** Sidebar content: command post card, popovers, ... */
  children: ReactNode
  /** Increment to bring the mobile sheet to its half snap. */
  focusSignal?: number
  /** Extra classes for the layout row (e.g. max width). */
  className?: string
}

/**
 * Shared game screen shell: a viewport-fitting row with the map on one side
 * and the panel sheet on the other. The parent provides the height context
 * (h-dvh + overflow-hidden) and the page padding; on phones the map fills the
 * free space and the panels float as a bottom sheet.
 */
export function GameLayout({ map, children, focusSignal, className }: GameLayoutProps) {
  const { t } = useLanguage()

  return (
    <div
      className={cn(
        'flex min-h-0 flex-1 flex-col lg:min-w-0 lg:flex-row lg:items-stretch lg:gap-4',
        className,
      )}
    >
      <section
        aria-label={t('map.territories')}
        className="relative min-h-0 min-w-0 flex-1 overflow-hidden rounded-2xl border border-[#b7a786] bg-[#e6d8bb] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]"
      >
        <div className="pointer-events-none absolute left-4 top-3 z-10 sm:left-5 sm:top-5">
          <p className="text-[10px] font-semibold uppercase tracking-[0.22em] text-[#806f57]">
            {t('app.mapPublic')}
          </p>
          <p className="mt-1 text-xs text-[#594b3c]">{t('app.mapInstructions')}</p>
        </div>
        {map}
      </section>
      <PanelSheet focusSignal={focusSignal} className="lg:w-96 lg:shrink-0 xl:w-[27rem]">
        {children}
      </PanelSheet>
    </div>
  )
}
