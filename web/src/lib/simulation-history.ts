import { useCallback, useState } from 'react'

import type { SimulationAction } from '@/types'

interface History {
  past: SimulationAction[]
  future: SimulationAction[]
}

const empty: History = { past: [], future: [] }

/** Ordered hypothetical actions with undo and redo. */
export function useSimulationHistory() {
  const [history, setHistory] = useState<History>(empty)

  const push = useCallback(
    (action: SimulationAction) =>
      setHistory((h) => ({ past: [...h.past, action], future: [] })),
    [],
  )
  const undo = useCallback(
    () =>
      setHistory((h) =>
        h.past.length === 0
          ? h
          : { past: h.past.slice(0, -1), future: [h.past[h.past.length - 1], ...h.future] },
      ),
    [],
  )
  const redo = useCallback(
    () =>
      setHistory((h) =>
        h.future.length === 0
          ? h
          : { past: [...h.past, h.future[0]], future: h.future.slice(1) },
      ),
    [],
  )
  /** Forgets the last action for good, e.g. because the server refused it. */
  const dropLast = useCallback(
    () => setHistory((h) => ({ past: h.past.slice(0, -1), future: [] })),
    [],
  )
  const reset = useCallback(() => setHistory(empty), [])

  return {
    actions: history.past,
    canUndo: history.past.length > 0,
    canRedo: history.future.length > 0,
    push,
    undo,
    redo,
    dropLast,
    reset,
  }
}
