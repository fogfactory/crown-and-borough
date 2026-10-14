import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { GameLayout } from '@/components/GameLayout'
import { LanguageProvider } from '@/i18n/LanguageContext'
import { stubPhoneViewport } from '@/test/viewport'

const mapSection = (container: HTMLElement) =>
  container.querySelector('section') as HTMLElement

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('GameLayout', () => {
  it('renders the map and the drawer without a nested main landmark', () => {
    const { container } = render(
      <GameLayout map={<p>Map body</p>}>
        <aside>Drawer body</aside>
      </GameLayout>,
    )

    expect(screen.getByText('Map body')).toBeInTheDocument()
    expect(screen.getByText('Drawer body')).toBeInTheDocument()
    expect(container.querySelectorAll('main')).toHaveLength(0)
    expect(screen.queryByText(/public map/i)).toBeNull()
  })

  it('keeps the whole map on desktop, whatever the view', () => {
    const { container } = render(
      <GameLayout map={<p>Map body</p>} view="report">
        <aside>Drawer body</aside>
      </GameLayout>,
    )

    expect(mapSection(container)).toHaveAttribute('data-map-size', 'full')
    expect(mapSection(container)).toBeVisible()
  })

  it('sizes the map to the phone view: whole, split with the orders, or hidden', () => {
    stubPhoneViewport()
    const { container, rerender } = render(
      <GameLayout map={<p>Map body</p>} view="map">
        <aside>Drawer body</aside>
      </GameLayout>,
    )
    expect(mapSection(container)).toHaveAttribute('data-map-size', 'full')

    rerender(
      <GameLayout map={<p>Map body</p>} view="orders">
        <aside>Drawer body</aside>
      </GameLayout>,
    )
    expect(mapSection(container)).toHaveAttribute('data-map-size', 'split')
    expect(mapSection(container).className).toContain('h-[40%]')

    rerender(
      <GameLayout map={<p>Map body</p>} view="realm">
        <aside>Drawer body</aside>
      </GameLayout>,
    )
    expect(mapSection(container)).not.toBeVisible()
  })

  it('lays a closable card over the map', () => {
    const onClose = vi.fn()
    render(
      <LanguageProvider initialLanguage="fr">
        <GameLayout
          map={<p>Map body</p>}
          view="map"
          mapOverlay={<p>Rochevent</p>}
          onCloseOverlay={onClose}
        >
          <aside>Drawer body</aside>
        </GameLayout>
      </LanguageProvider>,
    )

    expect(screen.getByText('Rochevent').closest('[data-map-overlay]')).not.toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Fermer la fiche du territoire' }))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
