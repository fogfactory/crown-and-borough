import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { useDraftedProjection } from '@/lib/use-drafted-projection'
import type { StateData } from '@/types'

const state: StateData = {
  turn: 11,
  season: 'autumn',
  players: [{ id: 'P1', name: 'One', color: '#a84632', projectedIncome: 5 }],
  territories: [],
  nobles: [],
}

describe('useDraftedProjection', () => {
  it('returns the state untouched without a draft', () => {
    const fetcher = vi.fn()
    const { result } = renderHook(() =>
      useDraftedProjection({ state, playerId: 'P1', basePath: '/api/games/g', fetcher }),
    )
    expect(result.current).toBe(state)
    expect(fetcher).not.toHaveBeenCalled()
  })

  it('overlays the projection computed with the drafted cards', async () => {
    const projected = { ...state, players: [{ ...state.players[0], projectedIncome: 9 }] }
    const fetcher = vi.fn().mockResolvedValue(projected)
    const { result } = renderHook(() =>
      useDraftedProjection({
        state,
        playerId: 'P1',
        specialDraft: 'P RH EPR',
        basePath: '/api/games/g',
        fetcher,
      }),
    )
    await waitFor(() => expect(result.current?.players[0].projectedIncome).toBe(9))
    expect(fetcher.mock.calls[0][0]).toBe('/api/games/g/projection?special=P+RH+EPR')
  })

  it('keeps the plain projection when the request fails', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() =>
      useDraftedProjection({
        state,
        playerId: 'P1',
        specialDraft: 'P RH EPR',
        basePath: '/api/games/g',
        fetcher,
      }),
    )
    await waitFor(() => expect(fetcher).toHaveBeenCalled())
    expect(result.current).toBe(state)
  })
})
