import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { RulesPanel } from '@/components/RulesPanel'

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

    render(<RulesPanel />)

    expect(screen.getByText('Chargement des règles…')).toBeInTheDocument()
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

    render(<RulesPanel />)

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Impossible de charger les règles (503)',
      )
    })
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

    const { rerender } = render(<RulesPanel />)
    await screen.findByText('4. Aide-mémoire des ordres')

    rerender(<RulesPanel targetSection="winter-orders" navigationKey={1} />)

    await waitFor(() => {
      expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' })
    })

    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: originalScrollIntoView,
    })
  })
})
