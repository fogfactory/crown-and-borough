import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import App from '@/App'
import type { MapData, StateData, TurnReport } from '@/types'

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
  asOf: { T1: 1, T2: 1 },
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

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('App command/report tabs', () => {
  it('keeps selection and drafts while switching between tabs', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      return Promise.resolve({
        ok: true,
        json: async () => (url.includes('/map') ? map : state),
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
    expect(screen.getByText('JEA · Jean de Rosemont')).toBeInTheDocument()
    expect(screen.getAllByText(/Robert de Rosemont/)).not.toHaveLength(0)
    expect(screen.getByText('Otage')).toBeInTheDocument()
    expect(screen.getByText('ROS A BRU')).toBeInTheDocument()
    expect(screen.getByText('(H ROS)').closest('li')).toHaveAttribute(
      'aria-current',
      'step',
    )
    expect(screen.getByLabelText('Couleur de One')).toBeInTheDocument()

    const draft = screen.getByLabelText('Chaîne de JEA')
    fireEvent.change(draft, { target: { value: 'ROS A BRU' } })
    fireEvent.click(screen.getByRole('tab', { name: /Rapport/ }))
    expect(screen.getByText('Aucun rapport disponible')).toBeInTheDocument()

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
        json: async () => (url.includes('/map') ? map : state),
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
})
