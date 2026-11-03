import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

/** Header shared by the pages outside a game (specs/charte-graphique.md, Pages annexes). */
export function PageHeader({
  title,
  intro,
  size = 'page',
  leading,
  actions,
  className,
}: {
  title: ReactNode
  intro?: ReactNode
  size?: 'page' | 'landing'
  leading?: ReactNode
  actions?: ReactNode
  className?: string
}) {
  return (
    <header className={cn('border-b-4 border-double border-line-strong pb-4', className)}>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          {leading}
          <div>
            <p className="label-caps text-2xs font-bold text-brass-700">
              Crown &amp; Borough
            </p>
            <h1
              className={cn(
                'mt-1 font-display font-bold text-ink-900',
                size === 'landing' ? 'text-3xl' : 'text-2xl',
              )}
            >
              {title}
            </h1>
            {intro && <p className="mt-2 max-w-3xl text-sm text-ink-500">{intro}</p>}
          </div>
        </div>
        {actions}
      </div>
    </header>
  )
}
