import {
  useEffect,
  useRef,
  useState,
  type ComponentType,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import {
  IconBook,
  IconChevronDown,
  IconChevronUp,
  IconCrown,
  IconFeather,
  IconLayoutSidebarRightCollapse,
  IconLayoutSidebarRightExpand,
  IconNews,
} from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { nextSheetSnap, snapFromDrag, type SheetSnap } from '@/lib/map-gestures'
import { SEASON_TINT_BG } from '@/lib/season'
import { cn } from '@/lib/utils'
import type { Season } from '@/types'

export const PANEL_ORDER = ['orders', 'report', 'realm', 'rules'] as const
export type Panel = (typeof PANEL_ORDER)[number]

const PANEL_LABEL_KEYS: Record<Panel, MessageKey> = {
  orders: 'drawer.orders',
  report: 'drawer.report',
  realm: 'drawer.realm',
  rules: 'drawer.rules',
}

const PANEL_ICONS: Record<
  Panel,
  ComponentType<{ className?: string; 'aria-hidden'?: boolean }>
> = {
  orders: IconFeather,
  report: IconNews,
  realm: IconCrown,
  rules: IconBook,
}

/** Phone sheet heights; the drawer ignores them from lg up. */
const SHEET_HEIGHTS: Record<SheetSnap, string> = {
  peek: 'h-28',
  half: 'h-[55dvh]',
  full: 'h-[92dvh]',
}

export interface GameDrawerProps {
  activePanel: Panel
  onPanelChange: (panel: Panel) => void
  /** Panel contents, owned by the mode pages. */
  panels: Record<Panel, ReactNode>
  /** Extra node inside the report bookmark (turn badge, "new" marker, …). */
  reportLabelExtra?: ReactNode
  /** Sticky footer, visible whatever the bookmark (the main action). */
  footer?: ReactNode
  /** Current season: its interface tint marks the active bookmark. */
  season?: Season
  /** Increment to bring the phone sheet to its half snap. */
  focusSignal?: number
}

/**
 * Game drawer (charte-graphique.md § Disposition sur ordinateur): from lg up,
 * a foldable column on the right of the map whose bookmarks stick out of its
 * left edge; on phones, a bottom sheet with snap points and a row of tabs.
 * The bookmarks stand for the title, so no heading repeats the active one.
 */
export function GameDrawer({
  activePanel,
  onPanelChange,
  panels,
  reportLabelExtra,
  footer,
  season,
  focusSignal = 0,
}: GameDrawerProps) {
  const { t } = useLanguage()
  const [collapsed, setCollapsed] = useState(false)
  const [snap, setSnap] = useState<SheetSnap>('half')
  const dragStartY = useRef<number | null>(null)

  useEffect(() => {
    if (focusSignal > 0) {
      setSnap('half')
    }
  }, [focusSignal])

  const selectPanel = (panel: Panel) => {
    onPanelChange(panel)
    setCollapsed(false)
  }

  const handleTabKeyDown = (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    const currentIndex = PANEL_ORDER.indexOf(activePanel)
    let nextIndex: number | null = null
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
      nextIndex = (currentIndex + 1) % PANEL_ORDER.length
    } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
      nextIndex = (currentIndex - 1 + PANEL_ORDER.length) % PANEL_ORDER.length
    } else if (event.key === 'Home') {
      nextIndex = 0
    } else if (event.key === 'End') {
      nextIndex = PANEL_ORDER.length - 1
    }
    if (nextIndex === null) return

    event.preventDefault()
    const nextPanel = PANEL_ORDER[nextIndex]
    selectPanel(nextPanel)
    event.currentTarget.parentElement
      ?.querySelector<HTMLButtonElement>(`[data-panel-tab="${nextPanel}"]`)
      ?.focus()
  }

  const handlePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    dragStartY.current = event.clientY
    event.currentTarget.setPointerCapture?.(event.pointerId)
  }

  const handlePointerUp = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (dragStartY.current === null) return
    const deltaY = event.clientY - dragStartY.current
    dragStartY.current = null
    setSnap((current) => snapFromDrag(current, deltaY))
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
    <aside
      id="game-drawer"
      data-testid="game-drawer"
      data-sheet-snap={snap}
      data-collapsed={collapsed || undefined}
      aria-label={t('app.panelViews')}
      className={cn(
        'fixed inset-x-0 bottom-0 z-40 flex flex-col rounded-t-xl border-t border-line bg-paper-50 shadow-sheet transition-[height] duration-150 ease-out motion-reduce:transition-none',
        SHEET_HEIGHTS[snap],
        'lg:relative lg:inset-auto lg:z-20 lg:h-auto lg:shrink-0 lg:rounded-none lg:border-t-0 lg:border-l lg:transition-[width]',
        collapsed ? 'lg:w-0 lg:border-l-0' : 'lg:w-96 xl:w-[26rem]',
      )}
    >
      <div
        className="relative flex shrink-0 touch-none items-center justify-center py-2 lg:hidden"
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
        onPointerCancel={() => {
          dragStartY.current = null
        }}
      >
        <div className="h-1.5 w-12 rounded-full bg-line" aria-hidden="true" />
        <button
          type="button"
          aria-label={t(snap === 'full' ? 'panel.collapse' : 'panel.expand')}
          className="absolute right-3 top-1 flex size-8 items-center justify-center rounded-md text-ink-500 transition hover:bg-paper-100 hover:text-ink-900"
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

      <div className="flex shrink-0 flex-col px-3 pb-2 lg:absolute lg:right-full lg:top-4 lg:gap-1.5 lg:p-0">
        <div
          role="tablist"
          aria-label={t('app.panelViews')}
          className="flex gap-1 lg:flex-col lg:gap-1.5"
        >
          {PANEL_ORDER.map((panel) => {
            const active = activePanel === panel
            const Icon = PANEL_ICONS[panel]
            return (
              <button
                key={panel}
                type="button"
                role="tab"
                aria-selected={active}
                aria-controls={`${panel}-panel`}
                tabIndex={active ? 0 : -1}
                data-panel-tab={panel}
                className={cn(
                  'relative inline-flex flex-1 items-center justify-center gap-1.5 rounded-md border px-2 py-1.5 text-xs font-semibold transition',
                  'lg:flex-none lg:rounded-r-none lg:border-r-0 lg:px-1.5 lg:py-3 lg:shadow-panel lg:[writing-mode:vertical-rl] lg:rotate-180',
                  active
                    ? 'border-brass-700 bg-brass-400 text-ink-900'
                    : 'border-line bg-paper-100 text-ink-700 hover:bg-paper-50 hover:text-ink-900',
                )}
                onClick={() => selectPanel(panel)}
                onKeyDown={handleTabKeyDown}
              >
                {active && season && (
                  <span
                    aria-hidden="true"
                    data-season-tint={season}
                    className={cn(
                      'absolute inset-x-1 bottom-0 h-[3px] rounded-full lg:inset-x-auto lg:inset-y-1 lg:right-0 lg:h-auto lg:w-[3px]',
                      SEASON_TINT_BG[season],
                    )}
                  />
                )}
                <Icon aria-hidden={true} className="size-3.5 shrink-0 lg:rotate-180" />
                <span>{t(PANEL_LABEL_KEYS[panel])}</span>
                {panel === 'report' && reportLabelExtra}
              </button>
            )
          })}
        </div>
        <button
          type="button"
          aria-controls="game-drawer-content"
          aria-expanded={!collapsed}
          aria-label={t(collapsed ? 'drawer.expand' : 'drawer.collapse')}
          title={t(collapsed ? 'drawer.expand' : 'drawer.collapse')}
          className="hidden items-center justify-center rounded-l-md border border-r-0 border-line bg-paper-100 py-2 text-ink-700 shadow-panel transition hover:bg-paper-50 hover:text-ink-900 lg:flex"
          onClick={() => setCollapsed((current) => !current)}
        >
          {collapsed ? (
            <IconLayoutSidebarRightExpand aria-hidden="true" className="size-4" />
          ) : (
            <IconLayoutSidebarRightCollapse aria-hidden="true" className="size-4" />
          )}
        </button>
      </div>

      <div
        id="game-drawer-content"
        className={cn(
          'flex min-h-0 flex-1 flex-col lg:overflow-hidden',
          collapsed && 'lg:invisible',
        )}
      >
        <div className="flex min-h-0 flex-1 flex-col lg:w-96 xl:w-[26rem]">
          <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-4 sm:px-4 lg:pt-4">
            {PANEL_ORDER.map((panel) => (
              <div
                key={panel}
                id={`${panel}-panel`}
                role="tabpanel"
                aria-label={t(PANEL_LABEL_KEYS[panel])}
                hidden={activePanel !== panel}
                className="min-w-0 space-y-4"
              >
                {panels[panel]}
              </div>
            ))}
          </div>
          {footer && (
            <div
              data-drawer-footer
              className="shrink-0 space-y-2 border-t border-line bg-paper-50 px-3 py-3 sm:px-4"
            >
              {footer}
            </div>
          )}
        </div>
      </div>
    </aside>
  )
}
