import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

const getIdToken = vi.fn(async () => 'bob-token')
const signOut = vi.fn(async () => undefined)

vi.mock('@/auth/AuthProvider', () => ({
  useAuth: () => ({ user: { uid: 'bob-uid' }, getIdToken, signOut }),
}))

import { LanguageProvider } from '@/i18n/LanguageContext'
import { JoinPage } from '@/online/JoinPage'

afterEach(() => {
  vi.unstubAllGlobals()
  getIdToken.mockClear()
  signOut.mockClear()
})

function LocationProbe() {
  const location = useLocation()
  return <output data-testid="location">{location.pathname}</output>
}

describe('online invitation join', () => {
  it('posts only the opaque invitation code and opens the returned game', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      void input
      void init
      return new Response(JSON.stringify({ id: 'game-1', joined: true }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <LanguageProvider initialLanguage="en">
        <MemoryRouter initialEntries={['/join?gameId=game-1&inviteCode=AB12CD']}>
          <JoinPage />
          <LocationProbe />
        </MemoryRouter>
      </LanguageProvider>,
    )

    await waitFor(() =>
      expect(screen.getByTestId('location')).toHaveTextContent('/games/game-1'),
    )
    expect(fetchMock).toHaveBeenCalledOnce()
    const request = fetchMock.mock.calls[0]?.[1]
    expect(JSON.parse(String(request?.body))).toEqual({ inviteCode: 'AB12CD' })
    expect(new Headers(request?.headers).get('Authorization')).toBe('Bearer bob-token')
  })

  it('renders a rejected invitation without retrying it', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({ code: 'invalid_invitation', message: 'invalid' }),
            {
              status: 403,
              headers: { 'Content-Type': 'application/json' },
            },
          ),
      ),
    )

    render(
      <LanguageProvider initialLanguage="en">
        <MemoryRouter initialEntries={['/join?gameId=game-1&inviteCode=BAD999']}>
          <JoinPage />
        </MemoryRouter>
      </LanguageProvider>,
    )

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'invalid or no longer active',
    )
  })
})
