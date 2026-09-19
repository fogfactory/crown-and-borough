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
          className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg border border-[#b7a786] bg-[#fffaf0] px-2 text-[#594b3c] transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none aria-expanded:bg-[#f3ead9] aria-expanded:text-[#30291f]"
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
            'z-50 max-h-[min(70dvh,32rem)] w-72 max-w-[calc(100vw-1.5rem)] overflow-y-auto rounded-xl border border-[#b7a786] bg-[#fffaf0] p-3 shadow-[0_18px_50px_-30px_rgba(67,46,24,0.9)] data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95',
            contentClassName,
          )}
        >
          {children}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
