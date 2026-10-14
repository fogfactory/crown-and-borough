import { fireEvent, render, screen } from '@testing-library/react'
import { useState, type ReactNode } from 'react'
import { describe, expect, it } from 'vitest'

import { GameDrawer, type Panel } from '@/components/GameDrawer'
import { LanguageProvider } from '@/i18n/LanguageContext'

function Harness({
  reportLabelExtra,
  footer,
  focusSignal,
}: {
  reportLabelExtra?: ReactNode
  footer?: ReactNode
  focusSignal?: number
}) {
  const [panel, setPanel] = useState<Panel>('orders')
  return (
    <LanguageProvider initialLanguage="en">
      <GameDrawer
        activePanel={panel}
        onPanelChange={setPanel}
        season="autumn"
        reportLabelExtra={reportLabelExtra}
        footer={footer}
        focusSignal={focusSignal}
        panels={{
          orders: <p>orders content</p>,
          report: <p>report content</p>,
          realm: <p>realm content</p>,
          rules: <p>rules content</p>,
        }}
      />
    </LanguageProvider>
  )
}

describe('GameDrawer', () => {
  it('renders the four bookmarks and only the active panel', () => {
    render(<Harness />)

    for (const [panel, name] of [
      ['orders', 'Orders'],
      ['report', 'Report'],
      ['realm', 'Realm'],
      ['rules', 'Rules'],
    ]) {
      const tab = screen.getByRole('tab', { name })
      expect(tab).toHaveAttribute('data-panel-tab', panel)
      expect(tab).toHaveAttribute('aria-controls', `${panel}-panel`)
    }
    expect(screen.getByRole('tab', { name: 'Orders' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(screen.getByText('orders content')).toBeVisible()
    expect(screen.getByText('realm content')).not.toBeVisible()
    // The bookmarks stand for the title: no heading repeats the active one.
    expect(screen.queryByRole('heading')).toBeNull()
  })

  it('marks the active bookmark with the season tint', () => {
    render(<Harness />)

    const tint = screen
      .getByRole('tab', { name: 'Orders' })
      .querySelector('[data-season-tint]')
    expect(tint).toHaveAttribute('data-season-tint', 'autumn')
    expect(tint?.className).toContain('bg-season-autumn')
  })

  it('switches panels on click and with arrow keys', () => {
    render(<Harness />)

    fireEvent.click(screen.getByRole('tab', { name: 'Realm' }))
    expect(screen.getByText('realm content')).toBeVisible()

    fireEvent.keyDown(screen.getByRole('tab', { name: 'Realm' }), { key: 'ArrowDown' })
    expect(screen.getByRole('tab', { name: 'Rules' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(screen.getByRole('tab', { name: 'Rules' })).toHaveFocus()
  })

  it('renders the extra report label inside the report bookmark', () => {
    render(<Harness reportLabelExtra={<span>· 2</span>} />)

    expect(screen.getByRole('tab', { name: /report/i })).toHaveTextContent('· 2')
  })

  it('folds and reopens, a bookmark reopening a folded drawer', () => {
    render(<Harness />)
    const drawer = screen.getByTestId('game-drawer')

    fireEvent.click(screen.getByRole('button', { name: 'Fold the drawer' }))
    expect(drawer).toHaveAttribute('data-collapsed', 'true')
    expect(screen.getByRole('button', { name: 'Open the drawer' })).toHaveAttribute(
      'aria-expanded',
      'false',
    )

    fireEvent.click(screen.getByRole('tab', { name: 'Report' }))
    expect(drawer).not.toHaveAttribute('data-collapsed')
    expect(screen.getByText('report content')).toBeVisible()
  })

  it('keeps the footer outside the panels, whatever the bookmark', () => {
    render(<Harness footer={<button type="button">Seal orders</button>} />)

    fireEvent.click(screen.getByRole('tab', { name: 'Rules' }))
    const seal = screen.getByRole('button', { name: 'Seal orders' })
    expect(seal).toBeVisible()
    expect(seal.closest('[role="tabpanel"]')).toBeNull()
  })

  it('snaps the phone sheet with the chevron, the arrows and a drag', () => {
    render(<Harness />)
    const drawer = screen.getByTestId('game-drawer')
    expect(drawer).toHaveAttribute('data-sheet-snap', 'half')

    fireEvent.click(screen.getByRole('button', { name: 'Expand panel' }))
    expect(drawer).toHaveAttribute('data-sheet-snap', 'full')
    fireEvent.keyDown(screen.getByRole('button', { name: 'Collapse panel' }), {
      key: 'ArrowDown',
    })
    expect(drawer).toHaveAttribute('data-sheet-snap', 'peek')

    const handle = drawer.querySelector('div[aria-hidden="true"]') as HTMLElement
    fireEvent.pointerDown(handle, { pointerId: 1, clientY: 300 })
    fireEvent.pointerUp(handle, { pointerId: 1, clientY: 180 })
    expect(drawer).toHaveAttribute('data-sheet-snap', 'half')
  })

  it('brings the phone sheet back to half when the focus signal increments', () => {
    const { rerender } = render(<Harness focusSignal={0} />)
    const drawer = screen.getByTestId('game-drawer')

    fireEvent.click(screen.getByRole('button', { name: 'Expand panel' }))
    expect(drawer).toHaveAttribute('data-sheet-snap', 'full')

    rerender(<Harness focusSignal={1} />)
    expect(drawer).toHaveAttribute('data-sheet-snap', 'half')
  })
})
