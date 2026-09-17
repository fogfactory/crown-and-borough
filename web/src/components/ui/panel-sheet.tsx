import {
  useEffect,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import { IconChevronDown, IconChevronUp } from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'
import { nextSheetSnap, snapFromDrag, type SheetSnap } from '@/lib/map-gestures'
import { cn } from '@/lib/utils'

const SHEET_HEIGHTS: Record<SheetSnap, string> = {
  peek: 'h-20',
  half: 'h-[55dvh]',
  full: 'h-[92dvh]',
}

interface PanelSheetProps {
  children: ReactNode
  className?: string
  /** Increment to bring the sheet to its half snap (e.g. territory selected). */
  focusSignal?: number
}

/**
 * Side panel container: fixed bottom sheet with snap points on phones, plain
 * scrolling sidebar on desktop (lg and up).
 */
export function PanelSheet({ children, className, focusSignal = 0 }: PanelSheetProps) {
  const { t } = useLanguage()
  const [snap, setSnap] = useState<SheetSnap>('half')
  const dragStartY = useRef<number | null>(null)

  useEffect(() => {
    if (focusSignal > 0) {
      setSnap('half')
    }
  }, [focusSignal])

  const handlePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    dragStartY.current = event.clientY
    event.currentTarget.setPointerCapture?.(event.pointerId)
  }

  const handlePointerUp = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (dragStartY.current === null) {
      return
    }
    const deltaY = event.clientY - dragStartY.current
    dragStartY.current = null
    setSnap((current) => snapFromDrag(current, deltaY))
  }

  const handlePointerCancel = () => {
    dragStartY.current = null
  }

  const handleChevronKeyDown = (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      setSnap('full')
    } else if (event.key === 'ArrowDown') {
      event.preventDefault()
      setSnap('peek')
    }
  }

  return (
    <div
      data-sheet-snap={snap}
      data-testid="panel-sheet"
      className={cn(
        'fixed inset-x-0 bottom-0 z-40 flex flex-col rounded-t-2xl border-t border-[#b7a786] bg-[#efe7d8] shadow-[0_-12px_40px_-20px_rgba(67,46,24,0.6)] transition-[height] duration-200 ease-out lg:static lg:z-auto lg:h-full lg:min-h-0 lg:rounded-none lg:border-0 lg:bg-transparent lg:shadow-none',
        SHEET_HEIGHTS[snap],
        className,
      )}
    >
      <div
        className="relative flex shrink-0 touch-none items-center justify-center py-2 lg:hidden"
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerCancel}
      >
        <div className="h-1.5 w-12 rounded-full bg-[#b7a786]" aria-hidden="true" />
        <button
          type="button"
          aria-label={t(snap === 'full' ? 'panel.collapse' : 'panel.expand')}
          className="absolute right-3 top-1.5 flex size-7 items-center justify-center rounded-md text-[#806f57] transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
          onClick={() => setSnap(nextSheetSnap)}
          onKeyDown={handleChevronKeyDown}
        >
          {snap === 'full' ? (
            <IconChevronDown aria-hidden="true" className="size-4" />
          ) : (
            <IconChevronUp aria-hidden="true" className="size-4" />
          )}
        </button>
      </div>
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-3 pb-6 sm:px-4 lg:min-h-0 lg:p-0 lg:pb-0">
        {children}
      </div>
    </div>
  )
}
