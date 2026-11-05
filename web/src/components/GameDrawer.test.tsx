import { fireEvent, render, screen } from '@testing-library/react'
import { useState, type ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { GameDrawer, type Panel } from '@/components/GameDrawer'
import { LanguageProvider } from '@/i18n/LanguageContext'
import { stubPhoneViewport } from '@/test/viewport'

function Harness({
  initial = 'orders',
  reportLabelExtra,
  footer,
}: {
  initial?: Panel
  reportLabelExtra?: ReactNode
  footer?: ReactNode
}) {
  const [panel, setPanel] = useState<Panel>(initial)
  return (
    <LanguageProvider initialLanguage="en">
      <GameDrawer
        activePanel={panel}
        onPanelChange={setPanel}
        season="autumn"
        reportLabelExtra={reportLabelExtra}
        footer={footer}
        panels={{
          orders: <p>orders content</p>,
          report: <p>report content</p>,
          realm: <p>realm content</p>,
          rules: <p>rules content</p>,
        }}
      />
      <output data-testid="active">{panel}</output>
    </LanguageProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('GameDrawer on desktop', () => {
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
    expect(screen.queryByRole('tab', { name: 'Map' })).toBeNull()
    expect(screen.getByRole('tab', { name: 'Orders' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(screen.getByText('orders content')).toBeVisible()
    expect(screen.getByText('realm content')).not.toBeVisible()
    // The bookmarks stand for the title: no heading repeats the active one.
    expect(screen.queryByRole('heading')).toBeNull()
  })

  it('shows the orders when the phone left the map tab active', () => {
    render(<Harness initial="map" />)

    expect(screen.getByRole('tab', { name: 'Orders' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(screen.getByText('orders content')).toBeVisible()
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
})

describe('GameDrawer on a phone', () => {
  it('switches between five full-screen views with a bottom tab bar', () => {
    stubPhoneViewport()
    render(<Harness initial="map" />)

    const tabs = screen.getAllByRole('tab').map((tab) => tab.textContent)
    expect(tabs).toEqual(['Map', 'Orders', 'Report', 'Realm', 'Rules'])
    expect(screen.queryByRole('button', { name: 'Fold the drawer' })).toBeNull()
    // The map view hides the whole panel.
    expect(screen.getByTestId('game-drawer')).not.toBeVisible()

    fireEvent.click(screen.getByRole('tab', { name: 'Report' }))
    expect(screen.getByTestId('active')).toHaveTextContent('report')
    expect(screen.getByTestId('game-drawer')).toBeVisible()
    expect(screen.getByText('report content')).toBeVisible()

    fireEvent.keyDown(screen.getByRole('tab', { name: 'Report' }), { key: 'ArrowLeft' })
    expect(screen.getByRole('tab', { name: 'Orders' })).toHaveFocus()
  })

  it('gives every tab a 44 px touch target and tints the active one', () => {
    stubPhoneViewport()
    render(<Harness initial="map" />)

    for (const tab of screen.getAllByRole('tab')) {
      expect(tab.className).toContain('min-h-14')
      expect(tab.className).toContain('min-w-11')
    }
    expect(
      screen.getByRole('tab', { name: 'Map' }).querySelector('[data-season-tint]'),
    ).not.toBeNull()
  })

  it('shows the seal footer under the orders only', () => {
    stubPhoneViewport()
    render(<Harness footer={<button type="button">Seal orders</button>} />)

    expect(screen.getByRole('button', { name: 'Seal orders' })).toBeVisible()
    fireEvent.click(screen.getByRole('tab', { name: 'Realm' }))
    expect(screen.queryByRole('button', { name: 'Seal orders' })).toBeNull()
  })
})
