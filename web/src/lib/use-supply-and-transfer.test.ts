import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { draftOrders } from '@/test/parse-orders'
import { useSupplyAndTransfer, type SupplyFetcher } from '@/lib/use-supply-and-transfer'
import type { StateData, SupplyLine, TransferLine } from '@/types'

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 3,
      army: { owner: 'P1', size: 2, chain: null },
      infrastructures: [],
    },
    {
      id: 'BRU',
      owner: 'P1',
      resources: 0,
      army: null,
      infrastructures: [],
    },
  ],
  nobles: [],
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
  path: ['ROS'],
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

const defaultFetcher = ((url: string) =>
  url.includes('target=')
    ? Promise.resolve(transferLine)
    : Promise.resolve(supplyLine)) as unknown as SupplyFetcher

function buildHook(
  fetcher: SupplyFetcher = defaultFetcher,
  overrides: Partial<Parameters<typeof useSupplyAndTransfer>[0]> = {},
) {
  const selectedState = state.territories.find((territory) => territory.id === 'ROS')
  const props = {
    selectedId: 'ROS',
    state,
    selectedState,
    draftOrders: draftOrders({ HUG: 'ROS T BRU 1' }),
    ownerId: 'P1',
    fetcher,
    networkErrorMessage: 'network failed',
    onAuthError: vi.fn(),
    ...overrides,
  }
  return renderHook(() => useSupplyAndTransfer(props))
}

describe('useSupplyAndTransfer', () => {
  it('loads the supply line and derives transfer targets for the owned army', async () => {
    const { result } = buildHook()

    await waitFor(() => {
      expect(result.current.supplyLine).toEqual(supplyLine)
    })
    expect(result.current.supplyLoading).toBe(false)
    expect(result.current.selectedSupplyLine).toEqual(supplyLine)
    expect(result.current.transferTargets).toEqual(['BRU'])
    expect(result.current.transferTarget).toBe('BRU')
    expect(result.current.transferLine).toEqual(transferLine)
  })

  it('skips loading outside the supply seasons', async () => {
    const fetcher = vi.fn() as unknown as SupplyFetcher
    const { result } = buildHook(fetcher, {
      state: { ...state, season: 'winter' },
    })

    expect(fetcher).not.toHaveBeenCalled()
    expect(result.current.supplyLine).toBeNull()
    expect(result.current.supplyLoading).toBe(false)
  })

  it('reports auth failures through the callback', async () => {
    const error = Object.assign(new Error('401'), { status: 401 })
    const failingFetcher = vi.fn(() => Promise.reject(error)) as unknown as SupplyFetcher
    const onAuthError = vi.fn()
    const { result } = buildHook(failingFetcher, { onAuthError })

    await waitFor(() => {
      expect(onAuthError).toHaveBeenCalled()
    })
    expect(result.current.supplyError).toBeNull()
  })

  it('reports network failures as messages', async () => {
    const failingFetcher = vi.fn(() =>
      Promise.reject(new Error('boom')),
    ) as unknown as SupplyFetcher
    const { result } = buildHook(failingFetcher)

    await waitFor(() => {
      expect(result.current.supplyError).toBe('boom')
    })
  })

  it('reloads the supply line when the selection changes', async () => {
    const { result, rerender } = buildHook()
    await waitFor(() => {
      expect(result.current.selectedSupplyLine).not.toBeNull()
    })
    rerender({
      selectedId: 'BRU',
      state,
      selectedState: state.territories.find((territory) => territory.id === 'BRU'),
      draftOrders: {},
      ownerId: 'P1',
      fetcher: vi.fn(() => Promise.resolve(supplyLine)) as unknown as SupplyFetcher,
      networkErrorMessage: 'network failed',
    })

    await waitFor(() => {
      expect(result.current.supplyLine).toEqual(supplyLine)
    })
  })

  it('derives no transfer targets without drafts', () => {
    const { result } = buildHook(undefined, { draftOrders: {} })

    expect(result.current.transferTargets).toEqual([])
    expect(result.current.transferTarget).toBeNull()
  })
})
