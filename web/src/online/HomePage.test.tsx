import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

const getIdToken = vi.fn(async () => 'alice-token')

vi.mock('@/auth/AuthProvider', () => ({
  useAuth: () => ({
    user: { uid: 'alice-uid' },
    getIdToken,
    signOut: vi.fn(async () => undefined),
  }),
}))

import { LanguageProvider } from '@/i18n/LanguageContext'
import { HomePage } from '@/online/HomePage'

afterEach(() => {
  vi.unstubAllGlobals()
  getIdToken.mockClear()
})

function renderHome() {
  return render(
    <LanguageProvider initialLanguage="en">
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    </LanguageProvider>,
  )
}

describe('online home', () => {
  it('loads only the authenticated player games', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(
            JSON.stringify([
              {
                id: 'game-1',
                name: 'Northern Marches',
                status: 'playing',
                turn: 1,
                season: 'spring',
                revision: 1,
                players: [
                  { id: 'P1', name: 'Alice', color: '#a84632', submitted: false },
                ],
              },
            ]),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          ),
      ),
    )

    renderHome()

    expect(await screen.findByText('Northern Marches')).toBeInTheDocument()
    expect(screen.queryByText('another player game')).not.toBeInTheDocument()
    expect(getIdToken).toHaveBeenCalled()
  })

  it('creates a game and keeps the invitation visible without sending a player identity', async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'POST') {
        return new Response(
          JSON.stringify({
            id: 'game-2',
            inviteCode: 'AB12CD',
            inviteUrl: 'https://example.test/join?gameId=game-2&inviteCode=AB12CD',
          }),
          { status: 201, headers: { 'Content-Type': 'application/json' } },
        )
      }
      return new Response('[]', {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderHome()
    fireEvent.click(await screen.findByRole('button', { name: 'Create a game' }))
    fireEvent.change(screen.getByLabelText('Game name'), {
      target: { value: 'Second game' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Create and invite' }))

    expect(await screen.findByText('AB12CD')).toBeInTheDocument()
    const postCall = fetchMock.mock.calls.find((call) => call[1]?.method === 'POST')
    expect(postCall).toBeDefined()
    const body = JSON.parse(String(postCall?.[1]?.body)) as Record<string, unknown>
    expect(body).not.toHaveProperty('player')
    expect(body.players).toBe(4)
  })
})
