import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import App from '@/App'
import type { MapData, StateData, SupplyLine, TurnReport } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'T1',
      code: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: true,
      points: [
        [0, 0],
        [50, 0],
        [50, 50],
        [0, 50],
      ],
      adjacencies: ['T2'],
      impassable: [],
    },
    {
      id: 'T2',
      code: 'BRU',
      name: 'Bruyères',
      terrain: 'forest',
      village: false,
      points: [
        [50, 0],
        [100, 0],
        [100, 50],
        [50, 50],
      ],
      adjacencies: ['T1'],
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
      id: 'T1',
      owner: 'P1',
      resources: 3,
      army: {
        owner: 'P1',
        size: 2,
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
    { id: 'T2', owner: null, resources: 0, army: null, infrastructures: [] },
  ],
  nobles: [
    {
      id: 'N1',
      code: 'JEA',
      name: 'Jean de Rosemont',
      owner: 'P1',
      location: 'T1',
      status: 'free',
    },
    {
      id: 'N2',
      code: 'BOB',
      name: 'Robert de Rosemont',
      owner: 'P2',
      location: 'T1',
      status: 'hostage',
    },
    {
      id: 'N3',
      code: 'KAR',
      name: 'Karin de Bruyères',
      owner: 'P2',
      location: 'T1',
      status: 'dungeon',
    },
  ],
}

const resolvedReport: TurnReport = {
  header: { year: 1, season: 'spring', turn: 1 },
  players: [],
  receptions: [],
  supply: [],
  famines: [],
  combats: [],
  orders: [],
  moves: [],
  nobles: [],
}

const supplyLine: SupplyLine = {
  kind: 'army',
  territory: 'T1',
  armyOwner: 'P1',
  armySize: 2,
  rations: 1,
  demand: 1,
  source: 'T1',
  distance: 0,
  path: ['T1'],
  reachable: ['T1'],
  selfSupplied: false,
}

const rulesDocument = '# Règles du jeu\n\nLes ordres sont résolus simultanément.\n'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('App command/report tabs', () => {
  it('keeps selection and drafts while switching between tabs', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/map') ? map : url.includes('/supply') ? supplyLine : state,
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App />)
    const commandTab = await screen.findByRole('tab', { name: /Poste de commandement/ })
    fireEvent.keyDown(commandTab, { key: 'ArrowRight' })
    expect(screen.getByRole('tab', { name: /Rapport/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    fireEvent.keyDown(screen.getByRole('tab', { name: /Rapport/ }), { key: 'ArrowLeft' })
    expect(commandTab).toHaveAttribute('aria-selected', 'true')

    const firstTerritory = await waitFor(() => {
      const territory = container.querySelector('[data-territory-id="T1"]')
      if (!territory) throw new Error('territory did not render')
      return territory
    })
    fireEvent.keyDown(firstTerritory, { key: 'Enter', code: 'Enter' })

    expect(await screen.findByText('Nobles présents')).toBeInTheDocument()
    expect(screen.getByText('Légende').closest('aside')).toBeInTheDocument()
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
    expect(screen.getByText('ROS A BRU')).toBeInTheDocument()
    expect(screen.getByText('(H ROS)').closest('li')).toHaveAttribute(
      'aria-current',
      'step',
    )
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

  it('opens the report tab after a resolved submission', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
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

    render(<App />)
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
      territory: 'T2',
      armyOwner: 'P1',
      armySize: 0,
      rations: 0,
      demand: 0,
      source: 'T2',
      distance: 0,
      path: [],
      reachable: ['T1', 'T2'],
      selfSupplied: false,
    }
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/map') ? map : url.includes('/supply') ? sourceZone : sourceState,
        text: async () => rulesDocument,
      } as Response)
    })
    vi.stubGlobal('fetch', fetchMock)

    const { container } = render(<App />)
    const sourceTerritory = await waitFor(() => {
      const territory = container.querySelector('[data-territory-id="T2"]')
      if (!territory) throw new Error('source territory did not render')
      return territory
    })
    fireEvent.keyDown(sourceTerritory, { key: 'Enter', code: 'Enter' })

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/supply?territory=T2',
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
          id: 'T1',
          code: 'ROS',
          name: 'Rosemont',
          terrain: 'plain',
          village: true,
          points: [
            [0, 0],
            [50, 0],
            [50, 50],
            [0, 50],
          ],
          adjacencies: ['T2'],
          impassable: [],
        },
        {
          id: 'T2',
          code: 'BRU',
          name: 'Bruyères',
          terrain: 'forest',
          village: false,
          points: [
            [50, 0],
            [100, 0],
            [100, 50],
            [50, 50],
          ],
          adjacencies: ['T1'],
          impassable: [],
        },
        {
          id: 'T3',
          code: 'CHA',
          name: 'Champvert',
          terrain: 'hill',
          village: false,
          points: [
            [50, 50],
            [100, 50],
            [100, 100],
            [50, 100],
          ],
          adjacencies: ['T2'],
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
          id: 'T1',
          owner: 'P1',
          resources: 10,
          army: null,
          infrastructures: [],
        },
        { id: 'T2', owner: null, resources: 0, army: null, infrastructures: [] },
        { id: 'T3', owner: null, resources: 0, army: null, infrastructures: [] },
      ],
      nobles: [],
    }
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ map: newMap, state: newState }),
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

    const { container } = render(<App />)
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
        '/api/game',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ seed: 'nouvelle-graine', players: 6 }),
        }),
      )
    })
    await waitFor(() => {
      expect(container.querySelectorAll('[data-territory-id]')).toHaveLength(3)
    })
  })
})
