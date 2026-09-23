import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/lib/firebase', () => ({
  firebaseConfigured: false,
}))

import App from '@/App'
import type {
  MapData,
  OrdersPreview,
  StateData,
  SupplyLine,
  TransferLine,
  TurnReport,
} from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: true,
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
      name: 'Bruyères',
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
    { id: 'P1', name: 'One', color: '#a84632' },
    { id: 'P2', name: 'Two', color: '#2d5f9e' },
  ],
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 3,
      army: {
        owner: 'P1',
        size: 4,
        chain: {
          noble: 'JEA',
          currentIndex: 1,
          orders: [
            { type: 'attack', position: 'ROS', targets: ['BRU'], liaison: 'single' },
            { type: 'hold', position: 'ROS', liaison: 'loop' },
          ],
        },
      },
      infrastructures: [],
    },
    { id: 'BRU', owner: null, resources: 0, army: null, infrastructures: [] },
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
    {
      id: 'N2',
      code: 'BOB',
      name: 'Robert de Rosemont',
      owner: 'P2',
      location: 'ROS',
      status: 'hostage',
    },
    {
      id: 'N3',
      code: 'KAR',
      name: 'Karin de Bruyères',
      owner: 'P2',
      location: 'ROS',
      status: 'dungeon',
    },
  ],
}

const resolvedReport: TurnReport = {
  header: { year: 1, season: 'spring', turn: 1 },
  players: [],
  receptions: [],
  production: [],
  consumption: [],
  combats: [],
  orders: [],
  moves: [],
  nobles: [],
}

const supplyLine: SupplyLine = {
  kind: 'army',
  territory: 'ROS',
  armyOwner: 'P1',
  armySize: 4,
  terrainProduction: 3,
  localProduction: 3,
  rations: 3,
  totalDemand: 8,
  demand: 5,
  source: 'ROS',
  distance: 0,
  path: ['ROS'],
  reachable: ['ROS'],
  selfSupplied: false,
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

const rulesDocument = '# Règles du jeu\n\nLes ordres sont résolus simultanément.\n'

const GAME_ID = 'hotseat-1'
const GAME_PATH = `/api/games/${GAME_ID}`

type FetchImpl = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>

/**
 * Serves the hotseat game list with a single game, then delegates every other
 * request to impl.
 */
function hotseatFetch(impl: FetchImpl) {
  return vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    if (String(input) === '/api/games?player=P1' && init?.method !== 'POST') {
      return Promise.resolve({
        ok: true,
        json: async () => [{ id: GAME_ID }],
      } as Response)
    }
    return impl(input, init)
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  window.localStorage.clear()
})

describe('App command/report tabs', () => {
  it('opens the full rules and FAQ pages from the hotseat header', async () => {
    const fetchMock = hotseatFetch((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () => (url.includes('/map') ? map : state),
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App initialLanguage="fr" />)
    await screen.findByText('Tour 1 · Printemps')

    fireEvent.click(screen.getByRole('button', { name: /^Règles$/ }))
    expect(
      await screen.findByRole('heading', { name: 'Règles du jeu' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /^FAQ$/ }))
    expect(
      await screen.findByRole('heading', { name: 'FAQ tactique' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Quel est l’effet des moulins/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /^Partie$/ }))
    expect(await screen.findByRole('tablist')).toBeInTheDocument()
  })

  it('requests a server-filtered state when the hotseat player changes', async () => {
    const fetchMock = hotseatFetch((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () => (url.includes('/map') ? map : state),
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App initialLanguage="fr" />)
    await screen.findByText('Tour 1 · Printemps')

    fireEvent.click(screen.getByRole('combobox', { name: 'Joueur actif' }))
    fireEvent.click(await screen.findByRole('option', { name: /P2 · Two/ }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `${GAME_PATH}/state?player=P2`,
        expect.objectContaining({ signal: expect.anything() }),
      )
    })
  })

  it('renders installed chains and winter investments in the hotseat map', async () => {
    const winterState: StateData = {
      ...state,
      turn: 4,
      season: 'winter',
      territories: state.territories.map((territory, index) =>
        index === 0
          ? {
              ...territory,
              infrastructures: [{ type: 'village' as const, level: 1 }],
              army: {
                owner: 'P1',
                size: 4,
                chain: {
                  visibility: 'known',
                  currentIndex: 0,
                  orders: [
                    {
                      type: 'attack',
                      position: 'ROS',
                      targets: ['BRU'],
                      liaison: 'single',
                    },
                  ],
                },
              },
            }
          : territory,
      ),
    }
    const winterPreview: OrdersPreview = {
      errors: [],
      chains: [],
      winter: [
        {
          line: 1,
          status: 'applied',
          type: 'build',
          territory: 'ROS',
          infrastructure: 'castle',
        },
      ],
      winterCost: { spent: 10, available: 3 },
    }
    const fetchMock = hotseatFetch((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/orders/preview')
            ? winterPreview
            : url.includes('/map')
              ? map
              : winterState,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App initialLanguage="en" />)
    const textarea = await screen.findByLabelText('Winter orders for P1')
    fireEvent.change(textarea, { target: { value: 'C C ROS' } })

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `${GAME_PATH}/orders/preview?player=P1&lang=en`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            chains: [],
            winter: [{ lines: 'C C ROS' }],
            special: [],
          }),
        }),
      )
    })
    await waitFor(() => {
      const actionOverlay = container.querySelector('g[aria-label="Intentions overlay"]')
      const winterOverlay = container.querySelector('[data-winter-orders-overlay="true"]')
      expect(actionOverlay).toBeInTheDocument()
      expect(actionOverlay?.textContent).toContain('A')
      expect(winterOverlay).toBeInTheDocument()
      expect(
        winterOverlay?.querySelector('g[data-winter-ghost="true"]'),
      ).toBeInTheDocument()
    })
  })

  it('identifies a selected capital in the command post', async () => {
    const capitalState: StateData = {
      ...state,
      players: [
        { id: 'P1', name: 'One', color: '#a84632', capitalTerritory: 'ROS' },
        state.players[1],
      ],
      territories: [
        {
          ...state.territories[0],
          infrastructures: [{ type: 'castle', level: 1 }],
        },
        state.territories[1],
      ],
    }
    const fetchMock = hotseatFetch((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/map')
            ? map
            : url.includes('/supply')
              ? supplyLine
              : capitalState,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App initialLanguage="fr" />)
    const firstTerritory = await waitFor(() => {
      const territory = container.querySelector('[data-territory-id="ROS"]')
      if (!territory) throw new Error('territory did not render')
      return territory
    })
    fireEvent.keyDown(firstTerritory, { key: 'Enter', code: 'Enter' })

    expect(await screen.findByText('Capitale de One')).toBeInTheDocument()
    expect(screen.getByText('Capitale', { exact: true })).toBeInTheDocument()
  })

  it('keeps selection and drafts while switching between tabs', async () => {
    const fetchMock = hotseatFetch((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/map') ? map : url.includes('/supply') ? supplyLine : state,
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App initialLanguage="fr" />)
    const commandTab = await screen.findByRole('tab', { name: /Poste de commandement/ })
    fireEvent.keyDown(commandTab, { key: 'ArrowRight' })
    expect(screen.getByRole('tab', { name: /Rapport/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    fireEvent.keyDown(screen.getByRole('tab', { name: /Rapport/ }), { key: 'ArrowLeft' })
    expect(commandTab).toHaveAttribute('aria-selected', 'true')

    const firstTerritory = await waitFor(() => {
      const territory = container.querySelector('[data-territory-id="ROS"]')
      if (!territory) throw new Error('territory did not render')
      return territory
    })
    fireEvent.keyDown(firstTerritory, { key: 'Enter', code: 'Enter' })

    expect(await screen.findByText('Nobles présents')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Afficher la légende' }),
    ).toBeInTheDocument()
    expect(screen.getByText('JEA · Jean de Rosemont')).toBeInTheDocument()
    expect(screen.getAllByText(/Robert de Rosemont/)).not.toHaveLength(0)
    expect(screen.getByText('Otage')).toBeInTheDocument()
    const noblesSection = screen.getByText('Nobles présents').closest('div')
    if (!noblesSection) {
      throw new Error('nobles section did not render')
    }
    expect(within(noblesSection).getAllByText('Propriétaire')).toHaveLength(3)
    expect(within(noblesSection).getAllByText('Détenteur')).toHaveLength(2)
    expect(within(noblesSection).getAllByText('One')).toHaveLength(1)
    expect(within(noblesSection).getAllByText('Two')).toHaveLength(2)
    expect(within(noblesSection).getByText('invité par One')).toBeInTheDocument()
    expect(within(noblesSection).getByText('emprisonné par One')).toBeInTheDocument()
    expect(within(noblesSection).queryByText('—')).not.toBeInTheDocument()
    expect(within(noblesSection).getAllByText('Détenteur')).toHaveLength(2)
    expect(screen.getAllByLabelText('Couleur de One')).toHaveLength(2)
    expect(screen.getAllByLabelText('Couleur de Two')).toHaveLength(2)
    expect(screen.getAllByText('ROS A BRU').length).toBeGreaterThanOrEqual(1)
    expect(
      screen.getByText('(H ROS)', { selector: 'span' }).closest('li'),
    ).toHaveAttribute('aria-current', 'step')
    expect(await screen.findByText(/Source :/)).toBeInTheDocument()
    expect(screen.getByText(/ROS · Rosemont/)).toBeInTheDocument()

    const draft = screen.getByLabelText('Chaîne de JEA')
    fireEvent.change(draft, { target: { value: 'ROS A BRU' } })
    fireEvent.click(screen.getByRole('tab', { name: /Rapport/ }))
    expect(screen.getByText('Aucun rapport disponible')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('tab', { name: /Poste de commandement/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Aide-mémoire des ordres' }))
    expect(screen.getByRole('tab', { name: /Règles/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(
      await screen.findByText('Les ordres sont résolus simultanément.'),
    ).toBeInTheDocument()

    fireEvent.click(screen.getByRole('tab', { name: /Poste de commandement/ }))
    expect(screen.getByLabelText('Chaîne de JEA')).toHaveValue('ROS A BRU')
    expect(screen.getByText('Rosemont')).toBeInTheDocument()
  })

  it('requests a transfer overlay for a drafted action transfer', async () => {
    const transferPreview: OrdersPreview = {
      errors: [],
      chains: [
        {
          noble: 'JEA',
          orders: [
            {
              type: 'transfer',
              position: 'ROS',
              targets: ['BRU'],
              amount: 1,
              liaison: 'single',
            },
          ],
        },
      ],
      winter: [],
    }
    const fetchMock = hotseatFetch((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/orders/preview')
            ? transferPreview
            : url.includes('target=')
              ? transferLine
              : url.includes('/map')
                ? map
                : url.includes('/supply')
                  ? supplyLine
                  : state,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App initialLanguage="en" />)
    await screen.findByLabelText('Chain for JEA')

    const territory = await waitFor(() => {
      const element = container.querySelector('[data-territory-id="ROS"]')
      if (!element) throw new Error('territory did not render')
      return element
    })
    fireEvent.keyDown(territory, { key: 'Enter', code: 'Enter' })
    fireEvent.change(screen.getByLabelText('Chain for JEA'), {
      target: { value: 'ROS T BRU 1' },
    })

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `${GAME_PATH}/supply?territory=ROS&target=BRU`,
        expect.objectContaining({ signal: expect.anything() }),
      )
    })
    expect(await screen.findByText('Transfer preview')).toBeInTheDocument()
    expect(
      await screen.findByText('The transfer route is reachable.'),
    ).toBeInTheDocument()
  })

  it('opens the report tab after a resolved submission', async () => {
    const fetchMock = hotseatFetch((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            status: 'resolved',
            submitted: [],
            remaining: [],
            report: resolvedReport,
            state,
          }),
        } as Response)
      }
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/map') ? map : url.includes('/supply') ? supplyLine : state,
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App initialLanguage="fr" />)
    await screen.findByLabelText('Chaîne de JEA')
    fireEvent.click(screen.getByRole('button', { name: 'Soumettre' }))

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /Rapport/ })).toHaveAttribute(
        'aria-selected',
        'true',
      )
    })
    expect(screen.getByText('Rapport du tour 1')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.queryByText('Nouveau')).not.toBeInTheDocument()
    })
  })

  it('renders a resolved private response containing combat and order details', async () => {
    const resolvedState: StateData = { ...state, turn: 4, season: 'winter' }
    const detailedReport: TurnReport = {
      ...resolvedReport,
      players: [
        {
          id: 'P1',
          name: 'One',
          resourcesBefore: 3,
          resourcesAfter: 3,
          controlledBefore: 1,
          controlledAfter: 1,
          armies: [{ id: 'A1', owner: 'P1', territory: 'ROS', size: 2 }],
          nobles: [],
          infrastructures: [],
        },
      ],
      combats: [
        {
          visibility: 'exact',
          territory: 'BRU',
          baseDefense: 1,
          defense: 2,
          castleBonus: 0,
          contenders: [{ army: 'A1', owner: 'P1', force: 3, defender: false }],
          cutSupporters: [],
          winner: 'A1',
          reason: 'attack_wins',
          standoff: false,
        },
      ],
      orders: [
        {
          visibility: 'known',
          army: 'A1',
          chain: 'C1',
          order: 'O1',
          owner: 'P1',
          noble: 'JEA',
          type: 'attack',
          source: 'ROS',
          targets: ['BRU'],
          liaison: 'single',
          outcome: 'success',
          progression: 'consumed',
          indexBefore: 0,
          indexAfter: 1,
        },
      ],
    }
    const fetchMock = hotseatFetch((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            status: 'resolved',
            submitted: [],
            remaining: [],
            report: detailedReport,
            state: resolvedState,
          }),
        } as Response)
      }
      return Promise.resolve({
        ok: true,
        json: async () => (url.includes('/map') ? map : state),
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App initialLanguage="en" />)
    await screen.findByText('Turn 1 · Spring')
    fireEvent.click(screen.getByRole('button', { name: 'Resolve' }))

    expect(await screen.findByText('Turn report 1')).toBeInTheDocument()
    expect(screen.getByText('Forces')).toBeInTheDocument()
    expect(screen.getAllByText('ROS A BRU').length).toBeGreaterThanOrEqual(1)
  })

  it('shows order validation errors above the order rules shortcut', async () => {
    const fetchMock = hotseatFetch((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST') {
        return Promise.resolve({
          ok: false,
          status: 400,
          json: async () => ({
            errors: [{ line: 2, message: 'ordre invalide' }],
          }),
        } as Response)
      }
      return Promise.resolve({
        ok: true,
        json: async () => (url.includes('/map') ? map : state),
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App initialLanguage="fr" />)
    await screen.findByLabelText('Chaîne de JEA')
    fireEvent.click(screen.getByRole('button', { name: 'Soumettre' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Ligne 2 : ordre invalide')

    const rulesShortcut = screen.getByRole('button', {
      name: 'Aide-mémoire des ordres',
    })
    expect(
      Boolean(
        alert.compareDocumentPosition(rulesShortcut) & Node.DOCUMENT_POSITION_FOLLOWING,
      ),
    ).toBe(true)
  })

  it('loads the reachable zone when a controlled source is selected', async () => {
    const sourceState: StateData = {
      ...state,
      territories: [
        state.territories[0],
        {
          ...state.territories[1],
          owner: 'P1',
          infrastructures: [{ type: 'castle', level: 1 }],
        },
      ],
    }
    const sourceZone: SupplyLine = {
      kind: 'source',
      territory: 'BRU',
      armyOwner: 'P1',
      armySize: 0,
      terrainProduction: 2,
      localProduction: 4,
      rations: 0,
      totalDemand: 0,
      demand: 0,
      source: 'BRU',
      distance: 0,
      path: [],
      reachable: ['ROS', 'BRU'],
      selfSupplied: false,
    }
    const fetchMock = hotseatFetch((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/map') ? map : url.includes('/supply') ? sourceZone : sourceState,
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App initialLanguage="fr" />)
    const sourceTerritory = await waitFor(() => {
      const territory = container.querySelector('[data-territory-id="BRU"]')
      if (!territory) throw new Error('source territory did not render')
      return territory
    })
    fireEvent.keyDown(sourceTerritory, { key: 'Enter', code: 'Enter' })

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `${GAME_PATH}/supply?territory=BRU`,
        expect.objectContaining({ signal: expect.anything() }),
      )
    })
    await waitFor(() => {
      expect(
        container.querySelector('g[aria-label="Zone de ravitaillement"]'),
      ).toBeInTheDocument()
    })
    expect(await screen.findByText('2 territoires atteignables.')).toBeInTheDocument()
    expect(
      container.querySelector('g[aria-label="Ligne de ravitaillement"]'),
    ).not.toBeInTheDocument()
  })

  it('starts a new game with the chosen seed and player count', async () => {
    const newMap: MapData = {
      territories: [
        {
          id: 'ROS',
          name: 'Rosemont',
          terrain: 'plain',
          village: true,
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
          name: 'Bruyères',
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
        {
          id: 'CHA',
          name: 'Champvert',
          terrain: 'hill',
          village: false,
          points: [
            [50, 50],
            [100, 50],
            [100, 100],
            [50, 100],
          ],
          adjacencies: ['BRU'],
          impassable: [],
        },
      ],
    }
    const newPlayers = Array.from({ length: 6 }, (_, index) => ({
      id: `P${index + 1}`,
      name: `Joueur ${index + 1}`,
      color: '#a84632',
    }))
    const newState: StateData = {
      turn: 1,
      season: 'spring',
      players: newPlayers,
      territories: [
        {
          id: 'ROS',
          owner: 'P1',
          resources: 10,
          army: null,
          infrastructures: [],
        },
        { id: 'BRU', owner: null, resources: 0, army: null, infrastructures: [] },
        { id: 'CHA', owner: null, resources: 0, army: null, infrastructures: [] },
      ],
      nobles: [],
    }
    const fetchMock = hotseatFetch((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ id: 'hotseat-2' }),
        } as Response)
      }
      if (url.startsWith('/api/games/hotseat-2/')) {
        return Promise.resolve({
          ok: true,
          json: async () => (url.includes('/map') ? newMap : newState),
        } as Response)
      }
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/map') ? map : url.includes('/supply') ? supplyLine : state,
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App initialLanguage="fr" />)
    await screen.findByText('Tour 1 · Printemps')

    const newGameButton = screen.getByRole('button', { name: 'Nouvelle partie' })
    expect(newGameButton).toBeDisabled()

    fireEvent.change(screen.getByLabelText('Graine'), {
      target: { value: 'nouvelle-graine' },
    })
    await waitFor(() => {
      expect(newGameButton).not.toBeDisabled()
    })
    fireEvent.click(screen.getByRole('combobox', { name: 'Joueurs' }))
    fireEvent.click(await screen.findByRole('option', { name: '6' }))
    fireEvent.click(newGameButton)

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/games?player=P1&lang=fr',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            name: 'Hotseat',
            seed: 'nouvelle-graine',
            players: 6,
            years: 10,
          }),
        }),
      )
    })
    await waitFor(() => {
      expect(container.querySelectorAll('[data-territory-id]')).toHaveLength(3)
    })
  })

  it('forces the resolution as the host and reloads the selected player view', async () => {
    const fetchMock = hotseatFetch((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST' && url.includes('/orders')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            status: 'pending',
            submitted: ['P2'],
            remaining: ['P1'],
            state,
          }),
        } as Response)
      }
      if (init?.method === 'POST' && url.includes('/resolve')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            status: 'resolved',
            submitted: ['P2'],
            remaining: [],
            state,
          }),
        } as Response)
      }
      if (url.includes('/reports/0')) {
        return Promise.resolve({ ok: true, json: async () => resolvedReport } as Response)
      }
      if (url.includes('/reports')) {
        return Promise.resolve({ ok: true, json: async () => [{ index: 0 }] } as Response)
      }
      return Promise.resolve({
        ok: true,
        json: async () => (url.includes('/map') ? map : state),
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App initialLanguage="fr" />)
    await screen.findByText('Tour 1 · Printemps')
    fireEvent.click(screen.getByRole('combobox', { name: 'Joueur actif' }))
    fireEvent.click(await screen.findByRole('option', { name: /P2 · Two/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Résoudre' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `${GAME_PATH}/resolve?player=P1&lang=fr`,
        expect.objectContaining({ method: 'POST' }),
      )
    })
    expect(fetchMock).toHaveBeenCalledWith(`${GAME_PATH}/reports/0?player=P2`)
    expect(await screen.findByText('Rapport du tour 1')).toBeInTheDocument()
  })

  it('reopens the remembered game and falls back to the first listed game', async () => {
    const serve = (games: string[]) =>
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input)
        if (url === '/api/games?player=P1') {
          return Promise.resolve({
            ok: true,
            json: async () => games.map((id) => ({ id })),
          } as Response)
        }
        return Promise.resolve({
          ok: true,
          json: async () => (url.includes('/map') ? map : state),
          text: async () => rulesDocument,
        } as Response)
      })

    window.localStorage.setItem('cb.hotseatGame', 'second')
    const remembered = serve(['first', 'second'])
    vi.stubGlobal('fetch', remembered)
    const { unmount } = render(<App initialLanguage="fr" />)
    await screen.findByText('Tour 1 · Printemps')
    expect(remembered).toHaveBeenCalledWith(
      '/api/games/second/map?player=P1',
      expect.anything(),
    )
    unmount()

    window.localStorage.setItem('cb.hotseatGame', 'gone')
    const fallback = serve(['first'])
    vi.stubGlobal('fetch', fallback)
    render(<App initialLanguage="fr" />)
    await screen.findByText('Tour 1 · Printemps')
    expect(fallback).toHaveBeenCalledWith(
      '/api/games/first/map?player=P1',
      expect.anything(),
    )
    expect(window.localStorage.getItem('cb.hotseatGame')).toBe('first')
  })
})
