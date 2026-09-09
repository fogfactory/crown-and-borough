import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { MapData, StateData, SupplyLine, TransferLine } from '@/types'

const authMocks = vi.hoisted(() => ({
  getIdToken: vi.fn(async () => 'test-token'),
  signOut: vi.fn(async () => undefined),
  user: { uid: 'user-1' },
}))

vi.mock('@/auth/AuthProvider', () => ({
  useAuth: () => ({
    status: 'signed-in',
    user: authMocks.user,
    profile: { displayName: 'Alice', email: 'alice@example.test' },
    profileLoading: false,
    profileError: null,
    authError: null,
    getIdToken: authMocks.getIdToken,
    signOut: authMocks.signOut,
  }),
}))

vi.mock('@/lib/subscription', () => ({
  normalizeGameSummary: () => ({
    id: 'GAME1',
    name: 'Online game',
    seed: 'seed',
    status: 'playing',
    players: [
      {
        id: 'P1',
        name: 'Alice',
        color: '#a84632',
        actorId: 'user-1',
        submitted: false,
      },
      { id: 'P2', name: 'Bob', color: '#2d5f9e', submitted: false },
    ],
    currentPlayer: 'P1',
    turn: 1,
    season: 'spring',
    yearCount: 10,
    revision: 1,
  }),
  normalizeStateData: (value: unknown) => value as StateData,
  useGameSubscription: () => ({
    summary: null,
    view: null,
    loading: false,
    error: null,
  }),
}))

import { LanguageProvider } from '@/i18n/LanguageContext'
import { GamePage } from '@/online/GamePage'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: false,
      points: [
        [0, 0],
        [50, 0],
        [50, 50],
        [0, 50],
      ],
      adjacencies: ['BRU'],
      impassable: [],
    },
    {
      id: 'BRU',
      name: 'Brisecote',
      terrain: 'forest',
      village: false,
      points: [
        [50, 0],
        [100, 0],
        [100, 50],
        [50, 50],
      ],
      adjacencies: ['ROS'],
      impassable: [],
    },
  ],
}

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [
    { id: 'P1', name: 'Alice', color: '#a84632' },
    { id: 'P2', name: 'Bob', color: '#2d5f9e' },
  ],
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 2,
      army: { owner: 'P1', size: 2, chain: null },
      infrastructures: [],
    },
    {
      id: 'BRU',
      owner: 'P2',
      resources: 0,
      army: null,
      infrastructures: [{ type: 'village', level: 1 }],
    },
  ],
  nobles: [
    {
      id: 'N1',
      code: 'JEA',
      name: 'Jean de Rosemont',
      owner: 'P1',
      location: 'ROS',
      status: 'free',
    },
  ],
}

const supplyLine: SupplyLine = {
  kind: 'army',
  territory: 'ROS',
  armyOwner: 'P1',
  armySize: 2,
  terrainProduction: 3,
  localProduction: 3,
  rations: 2,
  totalDemand: 2,
  demand: 0,
  source: null,
  distance: 0,
  path: [],
  reachable: ['ROS'],
  selfSupplied: true,
}

const transferLine: TransferLine = {
  kind: 'transfer',
  source: 'ROS',
  target: 'BRU',
  armyOwner: 'P1',
  path: ['ROS', 'BRU'],
  reachable: true,
  distance: 1,
  reachableTerritories: ['ROS', 'BRU'],
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  authMocks.getIdToken.mockClear()
  authMocks.signOut.mockClear()
})

describe('GamePage transfer preview', () => {
  it('requests the online transfer overlay for a drafted action transfer', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.endsWith('/api/games/GAME1')) return Promise.resolve(jsonResponse({}))
      if (url.endsWith('/api/games/GAME1/map')) return Promise.resolve(jsonResponse(map))
      if (url.endsWith('/api/games/GAME1/state')) {
        return Promise.resolve(jsonResponse({ ...state, revision: 1 }))
      }
      if (url.includes('target=BRU')) return Promise.resolve(jsonResponse(transferLine))
      if (url.includes('/supply?territory=ROS'))
        return Promise.resolve(jsonResponse(supplyLine))
      throw new Error(`unexpected request: ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MemoryRouter initialEntries={['/games/GAME1']}>
          <Routes>
            <Route path="/games/:gameId" element={<GamePage />} />
          </Routes>
        </MemoryRouter>
      </LanguageProvider>,
    )

    await screen.findByText('Online game')
    const territory = await waitFor(() => {
      const element = container.querySelector('[data-territory-id="ROS"]')
      if (!element) throw new Error('territory did not render')
      return element
    })
    fireEvent.keyDown(territory, { key: 'Enter', code: 'Enter' })
    fireEvent.change(await screen.findByLabelText('Chain for JEA'), {
      target: { value: 'ROS T BRU 1' },
    })

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/games/GAME1/supply?territory=ROS&target=BRU',
        expect.objectContaining({
          headers: expect.any(Headers),
          signal: expect.anything(),
        }),
      )
    })
    expect(await screen.findByText('Transfer preview')).toBeInTheDocument()
    expect(
      await screen.findByText('The transfer route is reachable.'),
    ).toBeInTheDocument()
  })
})

describe('GamePage submission rehydration and divergence', () => {
  it('rehydrates submitted chain orders into empty textareas on load', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.endsWith('/api/games/GAME1')) return Promise.resolve(jsonResponse({}))
      if (url.endsWith('/api/games/GAME1/map')) return Promise.resolve(jsonResponse(map))
      if (url.endsWith('/api/games/GAME1/state')) {
        return Promise.resolve(jsonResponse({ ...state, revision: 1 }))
      }
      if (url.endsWith('/api/games/GAME1/my-submission')) {
        return Promise.resolve(
          jsonResponse({
            turn: 1,
            season: 'spring',
            submitted: true,
            chains: [{ noble: 'JEA', text: 'JEA\nROS A BRU' }],
          }),
        )
      }
      throw new Error(`unexpected request: ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <LanguageProvider initialLanguage="en">
        <MemoryRouter initialEntries={['/games/GAME1']}>
          <Routes>
            <Route path="/games/:gameId" element={<GamePage />} />
          </Routes>
        </MemoryRouter>
      </LanguageProvider>,
    )

    await screen.findByText('Online game')
    const textarea = await screen.findByLabelText('Chain for JEA')
    await waitFor(() => {
      expect(textarea).toHaveValue('ROS A BRU')
    })
    expect(screen.queryByText('Local draft differs from server')).not.toBeInTheDocument()

    // Modify local draft -> divergence note appears
    fireEvent.change(textarea, { target: { value: 'ROS H' } })
    expect(await screen.findByText('Local draft differs from server')).toBeInTheDocument()
    const restoreBtn = screen.getByRole('button', { name: 'Restore from server' })
    expect(restoreBtn).toBeInTheDocument()

    // Restore from server -> textarea returns to server version and warning disappears
    fireEvent.click(restoreBtn)
    expect(textarea).toHaveValue('ROS A BRU')
    expect(screen.queryByText('Local draft differs from server')).not.toBeInTheDocument()
  })

  it('rehydrates winter orders and shows divergence when modified', async () => {
    const winterState: StateData = {
      ...state,
      turn: 4,
      season: 'winter',
    }
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.endsWith('/api/games/GAME1')) return Promise.resolve(jsonResponse({}))
      if (url.endsWith('/api/games/GAME1/map')) return Promise.resolve(jsonResponse(map))
      if (url.endsWith('/api/games/GAME1/state')) {
        return Promise.resolve(jsonResponse({ ...winterState, revision: 1 }))
      }
      if (url.endsWith('/api/games/GAME1/my-submission')) {
        return Promise.resolve(
          jsonResponse({
            turn: 4,
            season: 'winter',
            submitted: true,
            chains: [],
            winter: { lines: 'R T ROS' },
          }),
        )
      }
      throw new Error(`unexpected request: ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <LanguageProvider initialLanguage="fr">
        <MemoryRouter initialEntries={['/games/GAME1']}>
          <Routes>
            <Route path="/games/:gameId" element={<GamePage />} />
          </Routes>
        </MemoryRouter>
      </LanguageProvider>,
    )

    await screen.findByText('Online game')
    const textarea = await screen.findByLabelText("Ordres d'hiver de P1")
    await waitFor(() => {
      expect(textarea).toHaveValue('R T ROS')
    })
    expect(screen.queryByText('Brouillon différent du serveur')).not.toBeInTheDocument()

    // Edit winter draft -> divergence note in French appears
    fireEvent.change(textarea, { target: { value: 'R T BRU' } })
    expect(await screen.findByText('Brouillon différent du serveur')).toBeInTheDocument()
    const restoreBtn = screen.getByRole('button', { name: 'Restaurer depuis le serveur' })
    fireEvent.click(restoreBtn)
    expect(textarea).toHaveValue('R T ROS')
    expect(screen.queryByText('Brouillon différent du serveur')).not.toBeInTheDocument()
  })
})
