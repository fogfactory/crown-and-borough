import { useCallback, useState } from 'react'

import type { SimulationAction } from '@/types'

/**
 * Message key and parameters (nobles and fiefs by code) of the history line of
 * an action. A kill with a second death or a garrison assumption reads as such.
 */
export function simulationLog(action: SimulationAction): {
  key: `sim.log.${string}`
  noble: string
  other: string
  fief: string
} {
  const noble = 'noble' in action ? action.noble : ''
  const fief = 'fief' in action ? action.fief : ''
  if (action.type === 'kill') {
    if (action.alsoKill)
      return { key: 'sim.log.killTogether', noble, other: action.alsoKill, fief }
    if (action.garrison)
      return { key: 'sim.log.killGarrison', noble, other: action.garrison, fief }
    if (action.noGarrison)
      return { key: 'sim.log.killNoGarrison', noble, other: '', fief }
  }
  const other = 'other' in action ? action.other : ''
  return { key: `sim.log.${action.type}`, noble, other, fief }
}

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
