import type { KeyboardEvent as ReactKeyboardEvent, ReactNode } from 'react'
import { IconBook } from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'

export const PANEL_ORDER = ['command', 'report', 'rules'] as const
export type Panel = (typeof PANEL_ORDER)[number]

/**
 * Roster of the three side-panel tabs (command post, turn report, rules),
 * shared by the hotseat and online screens. Only the report tab label can
 * carry extra content (turn badge, "new" marker) via `reportLabelExtra`.
 */
export function CommandReportRulesTabs({
  activePanel,
  onPanelChange,
  reportLabelExtra,
}: {
  activePanel: Panel
  onPanelChange: (panel: Panel) => void
  reportLabelExtra?: ReactNode
}) {
  const { t } = useLanguage()

  const handlePanelKeyDown = (event: ReactKeyboardEvent<HTMLButtonElement>) => {
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
    onPanelChange(nextPanel)
    event.currentTarget.parentElement
      ?.querySelector<HTMLButtonElement>(`[data-panel-tab="${nextPanel}"]`)
      ?.focus()
  }

  const labels: Record<Panel, ReactNode> = {
    command: t('app.commandPost'),
    report: (
      <>
        {t('app.turnReport')}
        {reportLabelExtra}
      </>
    ),
    rules: (
      <span className="inline-flex items-center gap-1.5">
        <IconBook aria-hidden="true" className="size-3.5" />
        {t('app.rules')}
      </span>
    ),
  }

  return (
    <div
      role="tablist"
      aria-label={t('app.panelViews')}
      className="mt-2 grid grid-cols-3 gap-1 rounded-lg bg-[#f3ead9] p-1"
    >
      {PANEL_ORDER.map((panel) => (
        <button
          key={panel}
          type="button"
          role="tab"
          aria-selected={activePanel === panel}
          aria-controls={`${panel}-panel`}
          tabIndex={activePanel === panel ? 0 : -1}
          data-panel-tab={panel}
          className={`rounded-md px-2 py-1.5 text-xs font-semibold transition ${activePanel === panel ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
          onClick={() => onPanelChange(panel)}
          onKeyDown={handlePanelKeyDown}
        >
          {labels[panel]}
        </button>
      ))}
    </div>
  )
}
