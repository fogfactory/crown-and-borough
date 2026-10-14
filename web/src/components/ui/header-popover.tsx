import type { ReactNode } from 'react'
import { Popover } from 'radix-ui'

import { cn } from '@/lib/utils'

interface HeaderPopoverProps {
  /** Accessible name and tooltip for the trigger. */
  label: string
  icon: ReactNode
  /** Short live hint rendered inside the trigger (e.g. "2/4"). */
  hint?: string
  children: ReactNode
  /** Classes for the popover panel (width, etc). */
  contentClassName?: string
}

/**
 * Header button opening a floating panel. Used for secondary information
 * (scores, lobby) that must not compete with the game itself.
 */
export function HeaderPopover({
  label,
  icon,
  hint,
  children,
  contentClassName,
}: HeaderPopoverProps) {
  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <button
          type="button"
          aria-label={label}
          title={label}
          className="inline-flex h-8 shrink-0 items-center justify-center gap-1.5 rounded-lg max-lg:h-11 max-lg:min-w-11 border border-line bg-paper-50 px-2 text-ink-700 transition hover:bg-paper-100 hover:text-ink-900 focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none aria-expanded:bg-paper-100 aria-expanded:text-ink-900"
        >
          {icon}
          {hint && <span className="text-xs font-semibold tabular-nums">{hint}</span>}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={8}
          className={cn(
            'z-50 max-h-[min(70dvh,32rem)] w-72 max-w-[calc(100vw-1.5rem)] overflow-y-auto rounded-xl border border-line bg-paper-50 p-3 shadow-panel-strong',
            contentClassName,
          )}
        >
          {children}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
