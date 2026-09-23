import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { GamePanelCard } from '@/components/GamePanelCard'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Panel } from '@/components/CommandReportRulesTabs'

function Harness() {
  const [panel, setPanel] = useState<Panel>('command')
  return (
    <LanguageProvider initialLanguage="en">
      <GamePanelCard
        activePanel={panel}
        onPanelChange={setPanel}
        subtitle={<span>Seat P1</span>}
        reportLabelExtra={<span className="badge">· 3</span>}
        command={<p data-testid="command">command content</p>}
        report={<p data-testid="report">report content</p>}
        rules={<p data-testid="rules">rules content</p>}
      />
    </LanguageProvider>
  )
}

describe('GamePanelCard', () => {
  it('renders the subtitle, shared tabs, and only the active panel content', () => {
    render(<Harness />)

    expect(screen.getByText('Seat P1')).toBeInTheDocument()
    expect(screen.getByTestId('command')).toBeVisible()
    expect(screen.getByTestId('report')).not.toBeVisible()
    expect(screen.getByTestId('rules')).not.toBeVisible()
    expect(screen.getByRole('tab', { name: /turn report/i })).toHaveTextContent('· 3')
  })

  it('switches the visible panel through the tab click', () => {
    render(<Harness />)

    fireEvent.click(screen.getByRole('tab', { name: /turn report/i }))
    expect(screen.getByTestId('report')).toBeVisible()
    expect(screen.getByTestId('command')).not.toBeVisible()
  })

  it('labels each tabpanel for accessibility', () => {
    const onPanelChange = vi.fn()
    render(
      <LanguageProvider initialLanguage="en">
        <GamePanelCard
          activePanel="command"
          onPanelChange={onPanelChange}
          subtitle="s"
          command="c"
          report="r"
          rules="u"
        />
      </LanguageProvider>,
    )

    expect(screen.getByRole('tabpanel', { name: /command post/i })).toBeInTheDocument()
    for (const [id, label] of [
      ['report-panel', 'Turn report'],
      ['rules-panel', 'Game rules'],
    ] as const) {
      const panel = document.getElementById(id)
      expect(panel).toHaveAttribute('aria-label', label)
    }
  })
})
