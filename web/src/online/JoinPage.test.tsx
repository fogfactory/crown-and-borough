import { fireEvent, render, screen, waitFor } from '@testing-library/react'
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
    // The preview (not a lobby here: no form), then the join.
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(String(fetchMock.mock.calls[0]?.[0])).toBe(
      '/api/games/game-1/join-preview?inviteCode=AB12CD',
    )
    const request = fetchMock.mock.calls[1]?.[1]
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

  it('lets a joiner name their house and rank the maps before joining', async () => {
    const square = (x: number): Array<[number, number]> => [
      [x, 0],
      [x + 10, 0],
      [x + 10, 10],
      [x, 10],
    ]
    const candidateMap = {
      territories: [
        { id: 'AAA', name: 'A', terrain: 'plain', village: false, points: square(0), adjacencies: [], impassable: [] },
      ],
      regions: [{ id: 'R1', seed: 'AAA', territories: ['AAA'] }],
      starts: [],
    }
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      void init
      const url = String(input)
      const json = (body: unknown, status = 200) =>
        new Response(JSON.stringify(body), {
          status,
          headers: { 'Content-Type': 'application/json' },
        })
      if (url.includes('/join-preview')) {
        return json({
          id: 'game-1',
          name: 'Marches',
          players: 3,
          seated: 1,
          started: false,
          member: false,
          seedCandidates: [
            { index: 0, seed: 'aaaa' },
            { index: 1, seed: 'bbbb' },
            { index: 2, seed: 'cccc' },
          ],
        })
      }
      if (url.includes('/seed-candidates/')) return json(candidateMap)
      return json({ id: 'game-1', joined: true }, 201)
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

    expect(await screen.findByText('1/3 players joined')).toBeInTheDocument()
    await waitFor(() => expect(screen.getAllByRole('img')).toHaveLength(3))
    // Candidate maps are fetched with the invitation code: the joiner is no member yet.
    expect(
      fetchMock.mock.calls.some((call) =>
        String(call[0]).endsWith('/seed-candidates/2/map?inviteCode=AB12CD'),
      ),
    ).toBe(true)
    // Nothing was joined before the form is confirmed.
    expect(fetchMock.mock.calls.some((call) => String(call[0]).endsWith('/join'))).toBe(false)

    fireEvent.change(screen.getByLabelText(/Your house name/), {
      target: { value: 'House Stark' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Move map 3 up' }))
    fireEvent.click(screen.getByRole('button', { name: 'Join the game' }))

    await waitFor(() =>
      expect(screen.getByTestId('location')).toHaveTextContent('/games/game-1'),
    )
    const joinCall = fetchMock.mock.calls.find((call) => String(call[0]).endsWith('/join'))
    expect(JSON.parse(String(joinCall?.[1]?.body))).toEqual({
      inviteCode: 'AB12CD',
      houseName: 'House Stark',
      seedRanking: [0, 2, 1],
    })
  })

  it('joins without a vote when the joiner leaves the maps alone', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      void init
      const url = String(input)
      const body = url.includes('/join-preview')
        ? { id: 'game-1', name: 'G', players: 2, seated: 1, started: false, member: false, seedCandidates: [] }
        : { id: 'game-1', joined: true }
      return new Response(JSON.stringify(body), {
        status: 200,
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
    fireEvent.click(await screen.findByRole('button', { name: 'Join the game' }))
    await waitFor(() =>
      expect(screen.getByTestId('location')).toHaveTextContent('/games/game-1'),
    )
    const joinCall = fetchMock.mock.calls.find((call) => String(call[0]).endsWith('/join'))
    expect(JSON.parse(String(joinCall?.[1]?.body))).toEqual({ inviteCode: 'AB12CD' })
  })
})
