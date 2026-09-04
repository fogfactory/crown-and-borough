import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { RulesPanel } from '@/components/RulesPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('RulesPanel', () => {
  it('loads and renders the French Markdown rules document', async () => {
    const fetchMock = vi.fn(() =>
      Promise.resolve({
        ok: true,
        text: async () => '# Règles du jeu\n\nUne règle **importante**.\n',
      } as Response),
    )
    vi.stubGlobal('fetch', fetchMock)

    render(
      <LanguageProvider initialLanguage="fr">
        <RulesPanel />
      </LanguageProvider>,
    )

    expect(screen.getByText('Chargement des règles...')).toBeInTheDocument()
    expect(
      await screen.findByRole('heading', { name: 'Règles du jeu' }),
    ).toBeInTheDocument()
    expect(screen.getByText('importante')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/rules?lang=fr', {
      signal: expect.anything(),
    })
  })

  it('shows a loading error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 503,
          text: async () => 'unavailable',
        } as Response),
      ),
    )

    render(
      <LanguageProvider initialLanguage="fr">
        <RulesPanel />
      </LanguageProvider>,
    )

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Impossible de charger les règles (503)',
      )
    })
  })

  it('leaves scrolling to the page in the full-page variant', async () => {
    const fetchMock = vi.fn(() =>
      Promise.resolve({
        ok: true,
        text: async () => '# Règles du jeu\n\nUn texte suffisamment long.\n',
      } as Response),
    )
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(
      <LanguageProvider initialLanguage="fr">
        <RulesPanel variant="page" />
      </LanguageProvider>,
    )

    await screen.findByRole('heading', { name: 'Règles du jeu' })
    expect(container.querySelector('[class*="max-h-"]')).not.toBeInTheDocument()
    expect(container.querySelector('[class*="overflow-y-auto"]')).not.toBeInTheDocument()
  })

  it('scrolls to the requested rules section', async () => {
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView
    const scrollIntoView = vi.fn()
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: scrollIntoView,
    })
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({
          ok: true,
          text: async () =>
            '# Règles\n\n## 4. Aide-mémoire des ordres\n\n## 5. Ordres d’hiver\n',
        } as Response),
      ),
    )

    const { rerender } = render(
      <LanguageProvider initialLanguage="fr">
        <RulesPanel />
      </LanguageProvider>,
    )
    await screen.findByText('4. Aide-mémoire des ordres')

    rerender(
      <LanguageProvider initialLanguage="fr">
        <RulesPanel targetSection="winter-orders" navigationKey={1} />
      </LanguageProvider>,
    )

    await waitFor(() => {
      expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' })
    })

    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: originalScrollIntoView,
    })
  })

  it('loads the English document and preserves English section anchors', async () => {
    const fetchMock = vi.fn(() =>
      Promise.resolve({
        ok: true,
        text: async () =>
          '# Game Rules\n\n## 4. Order Cheat Sheet\n\n## 5. Winter Orders\n',
      } as Response),
    )
    vi.stubGlobal('fetch', fetchMock)

    render(
      <LanguageProvider initialLanguage="en">
        <RulesPanel />
      </LanguageProvider>,
    )

    expect(await screen.findByRole('heading', { name: 'Game Rules' })).toBeInTheDocument()
    expect(screen.getByText('4. Order Cheat Sheet')).toHaveAttribute(
      'id',
      'rules-action-orders',
    )
    expect(screen.getByText('5. Winter Orders')).toHaveAttribute(
      'id',
      'rules-winter-orders',
    )
    expect(fetchMock).toHaveBeenCalledWith('/api/rules?lang=en', {
      signal: expect.anything(),
    })
  })
})
