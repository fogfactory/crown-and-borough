import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

const firestoreHarness = vi.hoisted(() => ({
  services: { firestore: {} },
  snapshots: [] as Array<{
    next: (snapshot: unknown) => void
    error: (error: { code: string; message: string }) => void
  }>,
  unsubscribes: [] as Array<ReturnType<typeof vi.fn>>,
}))

vi.mock('@/lib/firebase', () => ({
  getFirebaseServices: () => firestoreHarness.services,
}))

vi.mock('firebase/firestore', () => ({
  collection: vi.fn((...path: string[]) => ({ path })),
  doc: vi.fn((...path: string[]) => ({ path })),
  onSnapshot: vi.fn(
    (
      _reference: unknown,
      _options: unknown,
      next: (snapshot: unknown) => void,
      error: (error: { code: string; message: string }) => void,
    ) => {
      firestoreHarness.snapshots.push({ next, error })
      const unsubscribe = vi.fn()
      firestoreHarness.unsubscribes.push(unsubscribe)
      return unsubscribe
    },
  ),
  orderBy: vi.fn((field: string, direction: string) => ({ field, direction })),
  query: vi.fn((...parts: unknown[]) => ({ parts })),
  where: vi.fn((field: string, operator: string, value: string) => ({
    field,
    operator,
    value,
  })),
}))

import {
  normalizeGameSummary,
  useGameListSubscription,
  useGameSubscription,
} from '@/lib/subscription'

afterEach(() => {
  firestoreHarness.snapshots.length = 0
  firestoreHarness.unsubscribes.length = 0
})

const state = {
  turn: 1,
  season: 'spring',
  players: [{ id: 'P1', name: 'Alice', color: '#a84632' }],
  territories: [],
  nobles: [],
}

describe('Firestore game subscriptions', () => {
  it('normalizes the public summary without exposing raw document concerns to the UI', () => {
    expect(
      normalizeGameSummary(
        {
          id: 'game-1',
          ownerUid: 'alice-uid',
          memberUids: ['alice-uid'],
          players: [{ id: 'P1', name: 'Alice', color: '#a84632', actorId: 'alice-uid' }],
          submittedUids: ['alice-uid'],
          revision: 4,
          turn: 2,
          season: 'summer',
          status: 'playing',
        },
        'game-1',
        'alice-uid',
      ),
    ).toEqual({
      id: 'game-1',
      name: 'Crown & Borough',
      seed: '',
      currentPlayer: 'P1',
      canInvite: true,
      status: 'playing',
      winner: null,
      players: [
        {
          id: 'P1',
          name: 'Alice',
          color: '#a84632',
          actorId: 'alice-uid',
          submitted: true,
        },
      ],
      turn: 2,
      season: 'summer',
      revision: 4,
    })
  })

  it('subscribes to the public summary and only the current UID private view', async () => {
    const { result, unmount } = renderHook(() =>
      useGameSubscription('game-1', 'alice-uid'),
    )
    await waitFor(() => expect(firestoreHarness.snapshots).toHaveLength(2))

    act(() => {
      firestoreHarness.snapshots[0]?.next({
        exists: () => true,
        data: () => ({
          id: 'game-1',
          memberUids: ['alice-uid'],
          players: [{ id: 'P1', name: 'Alice', color: '#a84632', actorId: 'alice-uid' }],
          revision: 1,
          turn: 1,
          season: 'spring',
          status: 'playing',
        }),
      })
      firestoreHarness.snapshots[1]?.next({
        exists: () => true,
        data: () => ({
          gameId: 'game-1',
          uid: 'alice-uid',
          revision: 1,
          turn: 1,
          season: 'spring',
          state,
        }),
      })
    })

    await waitFor(() => {
      expect(result.current.summary?.id).toBe('game-1')
      expect(result.current.view?.uid).toBe('alice-uid')
    })
    unmount()
    expect(firestoreHarness.unsubscribes[0]).toHaveBeenCalledOnce()
    expect(firestoreHarness.unsubscribes[1]).toHaveBeenCalledOnce()
  })

  it('ignores snapshots older than a REST response revision', async () => {
    const { result } = renderHook(() => useGameSubscription('game-1', 'alice-uid', 4))
    await waitFor(() => expect(firestoreHarness.snapshots).toHaveLength(2))

    act(() => {
      firestoreHarness.snapshots[0]?.next({
        exists: () => true,
        data: () => ({
          id: 'game-1',
          memberUids: ['alice-uid'],
          players: [],
          revision: 3,
          turn: 2,
          season: 'summer',
          status: 'playing',
        }),
      })
      firestoreHarness.snapshots[1]?.next({
        exists: () => true,
        data: () => ({ gameId: 'game-1', uid: 'alice-uid', revision: 3, state }),
      })
    })

    expect(result.current.summary).toBeNull()
    expect(result.current.view).toBeNull()
  })

  it('cleans both listeners when Firestore denies access', async () => {
    const { result } = renderHook(() => useGameSubscription('game-1', 'alice-uid'))
    await waitFor(() => expect(firestoreHarness.snapshots).toHaveLength(2))

    act(() => {
      firestoreHarness.snapshots[0]?.error({
        code: 'permission-denied',
        message: 'denied',
      })
    })

    await waitFor(() => expect(result.current.error?.code).toBe('permission-denied'))
    expect(firestoreHarness.unsubscribes[0]).toHaveBeenCalledOnce()
    expect(firestoreHarness.unsubscribes[1]).toHaveBeenCalledOnce()
  })

  it('keeps two game subscriptions isolated from each other', async () => {
    const first = renderHook(() => useGameSubscription('game-1', 'alice-uid'))
    const second = renderHook(() => useGameSubscription('game-2', 'alice-uid'))
    await waitFor(() => expect(firestoreHarness.snapshots).toHaveLength(4))

    act(() => {
      firestoreHarness.snapshots[0]?.next({
        exists: () => true,
        data: () => ({
          id: 'game-1',
          memberUids: ['alice-uid'],
          players: [],
          revision: 1,
        }),
      })
      firestoreHarness.snapshots[1]?.next({
        exists: () => true,
        data: () => ({ gameId: 'game-1', uid: 'alice-uid', revision: 1, state }),
      })
      firestoreHarness.snapshots[2]?.next({
        exists: () => true,
        data: () => ({
          id: 'game-2',
          memberUids: ['alice-uid'],
          players: [],
          revision: 2,
        }),
      })
      firestoreHarness.snapshots[3]?.next({
        exists: () => true,
        data: () => ({ gameId: 'game-2', uid: 'alice-uid', revision: 2, state }),
      })
    })

    await waitFor(() => {
      expect(first.result.current.summary?.id).toBe('game-1')
      expect(second.result.current.summary?.id).toBe('game-2')
      expect(first.result.current.view?.gameId).toBe('game-1')
      expect(second.result.current.view?.gameId).toBe('game-2')
    })
    first.unmount()
    second.unmount()
  })

  it('uses the membership query for the home list and stops it on permission errors', async () => {
    const { result } = renderHook(() => useGameListSubscription('alice-uid'))
    await waitFor(() => expect(firestoreHarness.snapshots).toHaveLength(1))

    act(() => {
      firestoreHarness.snapshots[0]?.next({
        docs: [
          {
            id: 'game-1',
            data: () => ({ id: 'game-1', name: 'First game', players: [], revision: 1 }),
          },
        ],
      })
    })
    await waitFor(() => expect(result.current.games[0]?.id).toBe('game-1'))

    act(() => {
      firestoreHarness.snapshots[0]?.error({
        code: 'permission-denied',
        message: 'denied',
      })
    })
    await waitFor(() => expect(result.current.error?.code).toBe('permission-denied'))
    expect(firestoreHarness.unsubscribes[0]).toHaveBeenCalledOnce()
  })
})
