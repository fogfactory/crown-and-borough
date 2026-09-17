import type { ReactNode } from 'react'

import { PanelSheet } from '@/components/ui/panel-sheet'
import { useLanguage } from '@/i18n/LanguageContext'
import { cn } from '@/lib/utils'

interface GameLayoutProps {
  /** The interactive map (already includes its own overlays). */
  map: ReactNode
  /** Sidebar content: scoreboard, command post card, lobby, ... */
  children: ReactNode
  /** Increment to bring the mobile sheet to its half snap. */
  focusSignal?: number
  /** Extra classes for the main element (e.g. max width). */
  mainClassName?: string
}

/**
 * Shared game screen shell: full-height sticky map with a scrolling sidebar on
 * laptops, full-viewport map with a bottom sheet on phones.
 */
export function GameLayout({
  map,
  children,
  focusSignal,
  mainClassName,
}: GameLayoutProps) {
  const { t } = useLanguage()

  return (
    <main
      className={cn(
        'mx-auto flex w-full min-h-0 flex-1 flex-col p-3 sm:p-4 lg:flex-row lg:items-start lg:gap-4 lg:p-6',
        mainClassName,
      )}
    >
      <section
        aria-label={t('map.territories')}
        className="relative min-h-0 flex-1 overflow-hidden rounded-2xl border border-[#b7a786] bg-[#e6d8bb] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)] lg:sticky lg:top-[4.75rem] lg:h-[calc(100dvh-5.75rem)] lg:flex-none"
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
    </main>
  )
}
