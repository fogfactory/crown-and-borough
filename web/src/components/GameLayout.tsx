import type { ReactNode } from 'react'

import type { Panel } from '@/components/GameDrawer'
import { useLanguage } from '@/i18n/LanguageContext'
import { useIsDesktop } from '@/lib/use-media-query'
import { cn } from '@/lib/utils'

interface GameLayoutProps {
  /** The interactive map (already includes its own overlays). */
  map: ReactNode
  /** The game drawer (`GameDrawer`). */
  children: ReactNode
  /**
   * Active view, which sizes the map on phones: the whole screen for the
   * map, the top 40 % for the orders, nothing for the other panels.
   */
  view?: Panel
  /** Card laid over the bottom of the map (phones, map view). */
  mapOverlay?: ReactNode
  /** Extra classes for the layout row. */
  className?: string
}

/**
 * Shared game screen shell (charte-graphique.md § Disposition sur ordinateur
 * and § Téléphone). On desktop the map takes the whole width and the drawer
 * sits on its right edge; on phones the map, the panel and the tab bar stack.
 * The parent provides the height context (h-dvh + overflow-hidden); the map
 * reframes itself when its box changes.
 */
export function GameLayout({
  map,
  children,
  view = 'orders',
  mapOverlay,
  className,
}: GameLayoutProps) {
  const { t } = useLanguage()
  const desktop = useIsDesktop()
  const phoneMap = desktop
    ? 'full'
    : view === 'map'
      ? 'full'
      : view === 'orders'
        ? 'split'
        : 'none'

  return (
    <div
      className={cn(
        'flex min-h-0 flex-1 flex-col lg:min-w-0 lg:flex-row lg:items-stretch',
        className,
      )}
    >
      <section
        aria-label={t('map.territories')}
        data-map-size={phoneMap}
        hidden={phoneMap === 'none'}
        className={cn(
          'relative min-h-0 min-w-0 overflow-hidden bg-paper-300',
          phoneMap === 'split' ? 'h-[40%] shrink-0' : 'flex-1',
        )}
      >
        {map}
        {mapOverlay && (
          <div
            data-map-overlay
            className="absolute bottom-2 left-2 right-14 z-20 max-h-[55%] overflow-y-auto rounded-md border border-line bg-paper-50 p-3 shadow-panel"
          >
            {mapOverlay}
          </div>
        )}
      </section>
      {children}
    </div>
  )
}
