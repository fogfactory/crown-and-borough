import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'

import { CommandReportRulesTabs, type Panel } from '@/components/CommandReportRulesTabs'
import { LanguageProvider } from '@/i18n/LanguageContext'

function TabsHarness({ reportLabelExtra }: { reportLabelExtra?: React.ReactNode }) {
  const [panel, setPanel] = useState<Panel>('command')
  return (
    <LanguageProvider initialLanguage="en">
      <CommandReportRulesTabs
        activePanel={panel}
        onPanelChange={setPanel}
        reportLabelExtra={reportLabelExtra}
      />
      <p>{panel}</p>
    </LanguageProvider>
  )
}

describe('CommandReportRulesTabs', () => {
  it('renders the three tabs with the unified data attribute', () => {
    render(<TabsHarness />)

    for (const panel of ['command', 'report', 'rules']) {
      const tab = screen.getByRole('tab', { name: new RegExp(panel, 'i') })
      expect(tab).toHaveAttribute('data-panel-tab', panel)
      expect(tab).toHaveAttribute('aria-controls', `${panel}-panel`)
    }
    expect(screen.getByRole('tab', { name: /command post/i })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('moves the active tab with arrow keys and roving focus', () => {
    render(<TabsHarness />)

    const commandTab = screen.getByRole('tab', { name: /command post/i })
    fireEvent.keyDown(commandTab, { key: 'ArrowRight' })
    expect(screen.getByText('report')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /turn report/i })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('renders the extra report label inside the report tab', () => {
    render(<TabsHarness reportLabelExtra={<span>· 2</span>} />)

    expect(screen.getByRole('tab', { name: /turn report/i })).toHaveTextContent('· 2')
  })
})
