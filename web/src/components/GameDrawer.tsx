import {
  useState,
  type ComponentType,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
} from 'react'
import {
  IconBook,
  IconCrown,
  IconFeather,
  IconLayoutSidebarRightCollapse,
  IconLayoutSidebarRightExpand,
  IconMap,
  IconNews,
} from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { useIsDesktop } from '@/lib/use-media-query'
import { SEASON_TINT_BG } from '@/lib/season'
import { cn } from '@/lib/utils'
import type { Season } from '@/types'

/** Bookmarks of the desktop drawer. */
export const PANEL_ORDER = ['orders', 'report', 'realm', 'rules'] as const
/** Tabs of the phone tab bar: the map has its own screen there. */
export const PHONE_TAB_ORDER = ['map', ...PANEL_ORDER] as const
export type DrawerPanel = (typeof PANEL_ORDER)[number]
export type Panel = (typeof PHONE_TAB_ORDER)[number]

const PANEL_LABEL_KEYS: Record<Panel, MessageKey> = {
  map: 'drawer.map',
  orders: 'drawer.orders',
  report: 'drawer.report',
  realm: 'drawer.realm',
  rules: 'drawer.rules',
}

const PANEL_ICONS: Record<
  Panel,
  ComponentType<{ className?: string; 'aria-hidden'?: boolean }>
> = {
  map: IconMap,
  orders: IconFeather,
  report: IconNews,
  realm: IconCrown,
  rules: IconBook,
}

/** The drawer panel shown for a tab: on desktop the map is not a panel. */
export function drawerPanelOf(panel: Panel): DrawerPanel {
  return panel === 'map' ? 'orders' : panel
}

export interface GameDrawerProps {
  activePanel: Panel
  onPanelChange: (panel: Panel) => void
  /** Panel contents, owned by the mode pages. */
  panels: Record<DrawerPanel, ReactNode>
  /** Extra node inside the report bookmark (turn badge, "new" marker, …). */
  reportLabelExtra?: ReactNode
  /** Sticky footer: the main action (always on desktop, under Orders on phones). */
  footer?: ReactNode
  /** Current season: its interface tint marks the active bookmark. */
  season?: Season
}

/**
 * Game drawer (charte-graphique.md § Disposition sur ordinateur and
 * § Téléphone). From lg up, a foldable column on the right of the map whose
 * bookmarks stick out of its left edge. On phones, each view takes the whole
 * screen and a tab bar at the bottom switches between the map and the panels.
 * The bookmarks and tabs stand for the title, so no heading repeats them.
 */
export function GameDrawer(props: GameDrawerProps) {
  const desktop = useIsDesktop()
  return desktop ? <DesktopDrawer {...props} /> : <PhoneDrawer {...props} />
}

/** Roving keyboard focus over a row or column of tabs. */
function tabKeyHandler(
  order: readonly Panel[],
  active: Panel,
  select: (panel: Panel) => void,
) {
  return (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    const currentIndex = Math.max(0, order.indexOf(active))
    let nextIndex: number | null = null
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
      nextIndex = (currentIndex + 1) % order.length
    } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
      nextIndex = (currentIndex - 1 + order.length) % order.length
    } else if (event.key === 'Home') {
      nextIndex = 0
    } else if (event.key === 'End') {
      nextIndex = order.length - 1
    }
    if (nextIndex === null) return

    event.preventDefault()
    const nextPanel = order[nextIndex]
    select(nextPanel)
    event.currentTarget.parentElement
      ?.querySelector<HTMLButtonElement>(`[data-panel-tab="${nextPanel}"]`)
      ?.focus()
  }
}

function SeasonTint({ season, vertical }: { season?: Season; vertical: boolean }) {
  if (!season) return null
  return (
    <span
      aria-hidden="true"
      data-season-tint={season}
      className={cn(
        'absolute rounded-full',
        vertical ? 'inset-y-1 right-0 w-[3px]' : 'inset-x-2 top-0 h-[3px]',
        SEASON_TINT_BG[season],
      )}
    />
  )
}

function PanelContents({
  panels,
  visible,
}: {
  panels: Record<DrawerPanel, ReactNode>
  visible: DrawerPanel | null
}) {
  const { t } = useLanguage()
  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-3 pt-3 pb-4 sm:px-4 lg:pt-4">
      {PANEL_ORDER.map((panel) => (
        <div
          key={panel}
          id={`${panel}-panel`}
          role="tabpanel"
          aria-label={t(PANEL_LABEL_KEYS[panel])}
          hidden={visible !== panel}
          className="min-w-0 space-y-4"
        >
          {panels[panel]}
        </div>
      ))}
    </div>
  )
}

function Footer({ children }: { children: ReactNode }) {
  return (
    <div
      data-drawer-footer
      className="shrink-0 space-y-2 border-t border-line bg-paper-50 px-3 py-3 sm:px-4"
    >
      {children}
    </div>
  )
}

function DesktopDrawer({
  activePanel,
  onPanelChange,
  panels,
  reportLabelExtra,
  footer,
  season,
}: GameDrawerProps) {
  const { t } = useLanguage()
  const [collapsed, setCollapsed] = useState(false)
  const active = drawerPanelOf(activePanel)

  const selectPanel = (panel: Panel) => {
    onPanelChange(panel)
    setCollapsed(false)
  }
  const handleTabKeyDown = tabKeyHandler(PANEL_ORDER, active, selectPanel)

  return (
    <aside
      id="game-drawer"
      data-testid="game-drawer"
      data-collapsed={collapsed || undefined}
      aria-label={t('app.panelViews')}
      className={cn(
        'relative z-20 flex shrink-0 flex-col border-l border-line bg-paper-50 shadow-sheet transition-[width] duration-150 ease-out motion-reduce:transition-none',
        collapsed ? 'w-0 border-l-0' : 'w-96 xl:w-[26rem]',
      )}
    >
      <div className="absolute right-full top-4 flex flex-col gap-1.5">
        <div
          role="tablist"
          aria-label={t('app.panelViews')}
          className="flex flex-col gap-1.5"
        >
          {PANEL_ORDER.map((panel) => {
            const selected = active === panel
            const Icon = PANEL_ICONS[panel]
            return (
              <button
                key={panel}
                type="button"
                role="tab"
                aria-selected={selected}
                aria-controls={`${panel}-panel`}
                tabIndex={selected ? 0 : -1}
                data-panel-tab={panel}
                className={cn(
                  'relative inline-flex rotate-180 items-center justify-center gap-1.5 rounded-md rounded-r-none border border-r-0 px-1.5 py-3 text-xs font-semibold shadow-panel transition [writing-mode:vertical-rl]',
                  selected
                    ? 'border-brass-700 bg-brass-400 text-ink-900'
                    : 'border-line bg-paper-100 text-ink-700 hover:bg-paper-50 hover:text-ink-900',
                )}
                onClick={() => selectPanel(panel)}
                onKeyDown={handleTabKeyDown}
              >
                {selected && <SeasonTint season={season} vertical />}
                <Icon aria-hidden={true} className="size-3.5 shrink-0 rotate-180" />
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
          className="flex items-center justify-center rounded-l-md border border-r-0 border-line bg-paper-100 py-2 text-ink-700 shadow-panel transition hover:bg-paper-50 hover:text-ink-900"
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
          'flex min-h-0 flex-1 flex-col overflow-hidden',
          collapsed && 'invisible',
        )}
      >
        <div className="flex min-h-0 w-96 flex-1 flex-col xl:w-[26rem]">
          <PanelContents panels={panels} visible={active} />
          {footer && <Footer>{footer}</Footer>}
        </div>
      </div>
    </aside>
  )
}

function PhoneDrawer({
  activePanel,
  onPanelChange,
  panels,
  reportLabelExtra,
  footer,
  season,
}: GameDrawerProps) {
  const { t } = useLanguage()
  const handleTabKeyDown = tabKeyHandler(PHONE_TAB_ORDER, activePanel, onPanelChange)
  const onMap = activePanel === 'map'

  return (
    <>
      <aside
        id="game-drawer"
        data-testid="game-drawer"
        aria-label={t('app.panelViews')}
        hidden={onMap}
        className="flex min-h-0 flex-1 flex-col border-t border-line bg-paper-50"
      >
        <PanelContents panels={panels} visible={onMap ? null : activePanel} />
        {footer && activePanel === 'orders' && <Footer>{footer}</Footer>}
      </aside>
      <div
        role="tablist"
        aria-label={t('app.panelViews')}
        data-phone-tabs
        className="flex shrink-0 border-t border-line bg-paper-50 pb-[env(safe-area-inset-bottom)]"
      >
        {PHONE_TAB_ORDER.map((panel) => {
          const selected = activePanel === panel
          const Icon = PANEL_ICONS[panel]
          return (
            <button
              key={panel}
              type="button"
              role="tab"
              aria-selected={selected}
              aria-controls={panel === 'map' ? undefined : `${panel}-panel`}
              tabIndex={selected ? 0 : -1}
              data-panel-tab={panel}
              className={cn(
                'relative flex min-h-14 min-w-11 flex-1 flex-col items-center justify-center gap-0.5 px-1 text-2xs font-semibold transition',
                selected
                  ? 'bg-brass-400 text-ink-900'
                  : 'text-ink-700 hover:bg-paper-100',
              )}
              onClick={() => onPanelChange(panel)}
              onKeyDown={handleTabKeyDown}
            >
              {selected && <SeasonTint season={season} vertical={false} />}
              <Icon aria-hidden={true} className="size-5 shrink-0" />
              <span className="inline-flex items-center gap-1">
                {t(PANEL_LABEL_KEYS[panel])}
                {panel === 'report' && reportLabelExtra}
              </span>
            </button>
          )
        })}
      </div>
    </>
  )
}
