import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SelectedTerritoryDetails } from '@/components/SelectedTerritoryDetails'
import type { MapData, StateData, SupplyLine } from '@/types'

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
    { id: 'P1', name: 'Alice', color: '#a84632', capitalTerritory: 'ROS' },
    { id: 'P2', name: 'P2', color: '#2d5f9e' },
  ],
  territories: [
    {
      id: 'ROS',
      owner: 'P2',
      resources: 3,
      army: {
        owner: 'P2',
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
      infrastructures: [{ type: 'castle', level: 2 }],
    },
  ],
  nobles: [
    {
      id: 'N1',
      code: 'JEA',
      name: 'Jean de Rosemont',
      owner: 'P2',
      location: 'ROS',
      status: 'free',
    },
    {
      id: 'N2',
      code: 'ROB',
      name: 'Robert de Rosemont',
      owner: 'P1',
      location: 'ROS',
      status: 'hostage',
    },
  ],
}

const armySupply: SupplyLine = {
  kind: 'army',
  territory: 'ROS',
  armyOwner: 'P2',
  armySize: 2,
  rations: 1,
  demand: 2,
  source: 'BRU',
  distance: 2,
  path: ['BRU', 'ROS'],
  reachable: ['ROS', 'BRU'],
  selfSupplied: false,
}

describe('SelectedTerritoryDetails', () => {
  it('renders nobles, complete army details, supply, and infrastructure using preferred names', () => {
    render(
      <SelectedTerritoryDetails
        state={state}
        selectedTerritory={map.territories[0]}
        selectedState={state.territories[0]}
        preferredPlayers={[
          { id: 'P1', name: 'Alice' },
          { id: 'P2', name: 'Bob' },
        ]}
        selectedSupplyLine={armySupply}
        sourceTerritory={map.territories[1]}
        supplyLoading={false}
        supplyError={null}
      />,
    )

    expect(screen.getByText('Capital of Alice')).toBeInTheDocument()
    expect(screen.getByText('Plain')).toBeInTheDocument()
    expect(screen.getAllByText('Bob').length).toBeGreaterThan(0)
    expect(screen.getByText('2 troops')).toBeInTheDocument()
    expect(screen.getByText(/Source:/)).toBeInTheDocument()
    expect(screen.getByText(/BRU · Brisecote/)).toBeInTheDocument()
    expect(screen.getByText('Distance: 2 territories')).toBeInTheDocument()
    expect(screen.getByText('Local rations')).toBeInTheDocument()
    expect(screen.getByText('Demand to cover')).toBeInTheDocument()
    expect(screen.getByText('Castle')).toBeInTheDocument()
    expect(screen.getByText('Capital', { exact: true })).toBeInTheDocument()

    const noblesSection = screen.getByText('Nobles present').closest('div')
    if (!noblesSection) throw new Error('nobles section did not render')
    const nobleItems = Array.from(noblesSection.querySelectorAll('li'))
    expect(
      nobleItems.some((item) => item.textContent?.includes('JEA · Jean de Rosemont')),
    ).toBe(true)
    expect(
      nobleItems.some((item) => item.textContent?.includes('ROB · Robert de Rosemont')),
    ).toBe(true)
    expect(within(noblesSection).getAllByText('Owner')).toHaveLength(2)
    expect(within(noblesSection).getByText('Bob')).toBeInTheDocument()
    expect(within(noblesSection).getByText('held by Bob')).toBeInTheDocument()
    expect(within(noblesSection).queryByText('P2')).not.toBeInTheDocument()
  })

  it('renders the supply zone for an army-less territory', () => {
    const sourceState: StateData = {
      ...state,
      season: 'spring',
      territories: [
        {
          ...state.territories[0],
          owner: 'P1',
          army: null,
          infrastructures: [{ type: 'village', level: 1 }],
        },
      ],
      nobles: [],
    }
    const sourceSupply: SupplyLine = {
      ...armySupply,
      kind: 'source',
      territory: 'ROS',
      armySize: 0,
      rations: 0,
      demand: 0,
      source: 'ROS',
      distance: 0,
      path: [],
      reachable: ['ROS', 'BRU'],
    }

    render(
      <SelectedTerritoryDetails
        state={sourceState}
        selectedTerritory={map.territories[0]}
        selectedState={sourceState.territories[0]}
        selectedSupplyLine={sourceSupply}
        sourceTerritory={map.territories[0]}
        supplyLoading={false}
        supplyError={null}
      />,
    )

    expect(screen.getByText('Supply zone')).toBeInTheDocument()
    expect(screen.getByText('2 territories reachable.')).toBeInTheDocument()
    expect(screen.getByText('Village')).toBeInTheDocument()
    expect(
      screen.queryByText('There is no supply phase in winter.'),
    ).not.toBeInTheDocument()
  })

  it('renders the hidden-chain message without exposing its order stack', () => {
    const hiddenState: StateData = {
      ...state,
      season: 'winter',
      territories: [
        {
          ...state.territories[0],
          army: {
            ...state.territories[0].army!,
            chain: { visibility: 'hidden' },
          },
        },
      ],
      nobles: [],
    }

    render(
      <SelectedTerritoryDetails
        state={hiddenState}
        selectedTerritory={map.territories[0]}
        selectedState={hiddenState.territories[0]}
        selectedSupplyLine={null}
        sourceTerritory={null}
        supplyLoading={false}
        supplyError={null}
      />,
    )

    expect(
      screen.getByText('This chain is hidden from the selected player.'),
    ).toBeInTheDocument()
    expect(screen.getByText('There is no supply phase in winter.')).toBeInTheDocument()
    expect(screen.queryByText('Order stack')).not.toBeInTheDocument()
  })
})
