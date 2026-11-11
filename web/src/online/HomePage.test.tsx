import { act, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

const getIdToken = vi.fn(async () => 'alice-token')
const signOut = vi.fn(async () => undefined)

vi.mock('@/auth/AuthProvider', () => ({
  useAuth: () => ({
    user: { uid: 'alice-uid' },
    getIdToken,
    signOut,
  }),
}))

const realtime = vi.hoisted(() => ({
  current: {
    games: [] as unknown[],
    enabled: false,
    loading: false,
    error: null as { code: string; message: string } | null,
  },
}))

vi.mock('@/lib/subscription', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/subscription')>()),
  useGameListSubscription: () => realtime.current,
}))

import { LanguageProvider } from '@/i18n/LanguageContext'
import { HomePage } from '@/online/HomePage'

afterEach(() => {
  vi.useRealTimers()
  realtime.current = { games: [], enabled: false, loading: false, error: null }
  vi.unstubAllGlobals()
  getIdToken.mockClear()
})

function gameListResponse(name: string) {
  return new Response(
    JSON.stringify([
      {
        id: 'game-1',
        name,
        status: 'playing',
        turn: 1,
        season: 'spring',
        revision: 1,
        players: [],
      },
    ]),
    { status: 200, headers: { 'Content-Type': 'application/json' } },
  )
}

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
                seed: 'adelaide-de-beaufort',
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
    expect(screen.getByText('Year 1001')).toBeInTheDocument()
    expect(screen.getByText('10 years / 40 turns remaining')).toBeInTheDocument()
    expect(screen.getByText('Seed: adelaide-de-beaufort')).toBeInTheDocument()
    expect(screen.queryByText('another player game')).not.toBeInTheDocument()
    expect(getIdToken).toHaveBeenCalled()
  })

  it('polls the API under a discreet notice when the live listener fails', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    realtime.current = {
      games: [],
      enabled: true,
      loading: false,
      error: { code: 'permission-denied', message: 'permission denied' },
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(gameListResponse('Northern Marches'))
      .mockResolvedValueOnce(gameListResponse('Southern Reach'))
    vi.stubGlobal('fetch', fetchMock)

    renderHome()

    expect(await screen.findByText('Northern Marches')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Live updates are paused.')
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(30_000)
    })
    expect(await screen.findByText('Southern Reach')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('pauses the fallback polling while the tab is hidden and catches up after', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    realtime.current = {
      games: [],
      enabled: true,
      loading: false,
      error: { code: 'unavailable', message: 'unavailable' },
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(gameListResponse('Northern Marches'))
      .mockResolvedValueOnce(gameListResponse('Southern Reach'))
    vi.stubGlobal('fetch', fetchMock)
    const visibility = vi.spyOn(document, 'visibilityState', 'get')

    renderHome()
    expect(await screen.findByText('Northern Marches')).toBeInTheDocument()

    visibility.mockReturnValue('hidden')
    act(() => {
      vi.advanceTimersByTime(60_000)
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)

    visibility.mockReturnValue('visible')
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
      // The refresh in flight absorbs the tick that follows it.
      vi.advanceTimersByTime(30_000)
    })
    expect(await screen.findByText('Southern Reach')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    visibility.mockRestore()
  })

  it('hides a failed API load once the live listener feeds the list', async () => {
    realtime.current = {
      games: [
        {
          id: 'game-1',
          name: 'Northern Marches',
          seed: '',
          status: 'playing',
          winner: null,
          players: [],
          turn: 1,
          season: 'spring',
          yearCount: 10,
          scores: {},
          revision: 1,
        },
      ],
      enabled: true,
      loading: false,
      error: null,
    }
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response('{}', { status: 503 })),
    )

    renderHome()

    expect(await screen.findByText('Northern Marches')).toBeInTheDocument()
    // Let the failed load settle before asserting that it stays quiet.
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 20))
    })
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
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
    expect(screen.getByLabelText('Seed')).toHaveValue('adelaide-de-beaufort')
    fireEvent.change(screen.getByLabelText('Game name'), {
      target: { value: 'Second game' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Create and invite' }))

    expect(await screen.findByText('AB12CD')).toBeInTheDocument()
    const postCall = fetchMock.mock.calls.find((call) => call[1]?.method === 'POST')
    expect(postCall).toBeDefined()
    const body = JSON.parse(String(postCall?.[1]?.body)) as Record<string, unknown>
    expect(body).not.toHaveProperty('player')
    expect(body.seed).toBe('adelaide-de-beaufort')
    expect(body.players).toBe(4)
    expect(body.spectate).toBe(false)
    expect(body.years).toBe(10)
  })

  it('can create a game as a non-playing spectator host', async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'POST') {
        return new Response(
          JSON.stringify({
            id: 'observed-game',
            inviteCode: 'OBS123',
            inviteUrl: 'https://example.test/join?gameId=observed-game&inviteCode=OBS123',
          }),
          { status: 201, headers: { 'Content-Type': 'application/json' } },
        )
      }
      return new Response('[]', { status: 200 })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderHome()
    fireEvent.click(await screen.findByRole('button', { name: 'Create a game' }))
    fireEvent.click(screen.getByRole('checkbox', { name: /Observe without playing/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Create and invite' }))

    expect(await screen.findByText('OBS123')).toBeInTheDocument()
    const postCall = fetchMock.mock.calls.find((call) => call[1]?.method === 'POST')
    const body = JSON.parse(String(postCall?.[1]?.body)) as Record<string, unknown>
    expect(body.spectate).toBe(true)
  })
})
