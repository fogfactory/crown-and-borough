import type { ReactNode } from 'react'

import { CommandReportRulesTabs, type Panel } from '@/components/CommandReportRulesTabs'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useLanguage } from '@/i18n/LanguageContext'

export interface GamePanelCardProps {
  activePanel: Panel
  onPanelChange: (panel: Panel) => void
  /** Mode-specific line under the card title (selected player, seat, …). */
  subtitle: ReactNode
  /** Extra node inside the report tab button (turn badge, "new" marker, …). */
  reportLabelExtra?: ReactNode
  /** Command post content (territory details, orders, mode-specific extras). */
  command: ReactNode
  report: ReactNode
  rules: ReactNode
}

/**
 * Shared side-panel card for the hotseat and online game screens: title,
 * subtitle, the three roving tabs, and the hidden tab panels. The mode pages
 * own the panel contents.
 */
export function GamePanelCard({
  activePanel,
  onPanelChange,
  subtitle,
  reportLabelExtra,
  command,
  report,
  rules,
}: GamePanelCardProps) {
  const { t } = useLanguage()

  return (
    <Card className="border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
      <CardHeader className="border-b border-[#b7a786]/50 pb-3">
        <CardTitle className="font-serif text-lg text-[#30291f] sm:text-xl">
          {activePanel === 'command'
            ? t('app.commandPost')
            : activePanel === 'report'
              ? t('app.turnReport')
              : t('app.rules')}
        </CardTitle>
        <CardDescription className="text-[#806f57]">{subtitle}</CardDescription>
        <CommandReportRulesTabs
          activePanel={activePanel}
          onPanelChange={onPanelChange}
          reportLabelExtra={reportLabelExtra}
        />
      </CardHeader>
      <CardContent className="min-w-0 space-y-4 pt-4">
        <div
          id="command-panel"
          role="tabpanel"
          aria-label={t('app.commandPost')}
          hidden={activePanel !== 'command'}
          className="space-y-4"
        >
          {command}
        </div>
        <div
          id="report-panel"
          role="tabpanel"
          aria-label={t('app.turnReport')}
          hidden={activePanel !== 'report'}
          className="min-w-0"
        >
          {report}
        </div>
        <div
          id="rules-panel"
          role="tabpanel"
          aria-label={t('app.rules')}
          hidden={activePanel !== 'rules'}
          className="min-w-0"
        >
          {rules}
        </div>
      </CardContent>
    </Card>
  )
}
