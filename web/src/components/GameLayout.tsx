import type { ReactNode } from 'react'

import { useLanguage } from '@/i18n/LanguageContext'
import { cn } from '@/lib/utils'

interface GameLayoutProps {
  /** The interactive map (already includes its own overlays). */
  map: ReactNode
  /** The game drawer (`GameDrawer`). */
  children: ReactNode
  /** Extra classes for the layout row. */
  className?: string
}

/**
 * Shared game screen shell: the map takes the whole width and the drawer
 * sits on its right edge (a bottom sheet on phones). The parent provides the
 * height context (h-dvh + overflow-hidden); the map reframes itself when the
 * drawer opens or folds.
 */
export function GameLayout({ map, children, className }: GameLayoutProps) {
  const { t } = useLanguage()

  return (
    <div
      className={cn(
        'flex min-h-0 flex-1 flex-col lg:min-w-0 lg:flex-row lg:items-stretch',
        className,
      )}
    >
      <section
        aria-label={t('map.territories')}
        className="relative min-h-0 min-w-0 flex-1 overflow-hidden bg-paper-300"
      >
        {map}
      </section>
      {children}
    </div>
  )
}
