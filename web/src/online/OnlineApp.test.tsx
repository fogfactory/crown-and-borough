import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/auth/AuthProvider', () => ({
  useAuth: () => ({
    status: 'signed-in',
    user: { uid: 'player-1' },
    profile: { displayName: 'Alice', email: 'alice@example.test' },
    profileLoading: false,
    profileError: null,
    authError: null,
    signOut: vi.fn(async () => undefined),
  }),
}))

import { LanguageProvider } from '@/i18n/LanguageContext'
import { OnlineRoutes } from '@/online/OnlineApp'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('online information routes', () => {
  it('exposes the rules and FAQ links in the authenticated header', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <MemoryRouter initialEntries={['/faq']}>
          <OnlineRoutes />
        </MemoryRouter>
      </LanguageProvider>,
    )

    expect(screen.getByRole('heading', { name: 'Tactical FAQ' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Rules' })).toHaveAttribute('href', '/rules')
    expect(screen.getByRole('link', { name: 'FAQ' })).toHaveAttribute('href', '/faq')
  })

  it('loads the full rules page without an inner scroll container', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(
          new Response('# Game Rules\n\n## Rules\n\nA reference.\n', { status: 200 }),
        ),
      ),
    )

    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MemoryRouter initialEntries={['/rules']}>
          <OnlineRoutes />
        </MemoryRouter>
      </LanguageProvider>,
    )

    expect(await screen.findByRole('heading', { name: 'Game rules' })).toBeInTheDocument()
    expect(container.querySelector('[class*="max-h-"]')).not.toBeInTheDocument()
    expect(container.querySelector('[class*="overflow-y-auto"]')).not.toBeInTheDocument()
  })
})
