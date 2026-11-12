import { useEffect, useMemo, useState, type MutableRefObject } from 'react'

import {
  VICTORY_SIMULATION_DELAY_MS,
  type VictorySimulationRequest,
} from '@/lib/use-victory-simulation'
import type { SimulationAction, StateData } from '@/types'

/** Previews already computed, by history and action; the owner clears it when the real state moves on. */
export type ActionPreviewCache = Map<string, StateData>

const CACHE_LIMIT = 50

/**
 * Asks the server for the state each action leads to when applied after the
 * simulated history, so a menu can preview its options. Results are keyed by
 * the option's key; a refused option has no entry. Runs once the selection
 * rests, skips the options the cache already knows, and does nothing without
 * options.
 */
export function useActionPreviews(
  request: VictorySimulationRequest | null,
  base: SimulationAction[],
  options: Array<{ key: string; action: SimulationAction }>,
  cache: MutableRefObject<ActionPreviewCache>,
): Record<string, StateData> {
  const [previews, setPreviews] = useState<{ id: string; states: Record<string, StateData> }>({
    id: '',
    states: {},
  })
  const id = useMemo(() => JSON.stringify([base, options]), [base, options])

  useEffect(() => {
    if (!request || options.length === 0) return
    const cacheKey = (action: SimulationAction) => JSON.stringify([base, action])
    const known: Record<string, StateData> = {}
    const missing = options.filter((option) => {
      const hit = cache.current.get(cacheKey(option.action))
      if (hit) known[option.key] = hit
      return !hit
    })
    setPreviews({ id, states: known })
    if (missing.length === 0) return
    const controller = new AbortController()
    const timer = window.setTimeout(() => {
      for (const option of missing) {
        request([...base, option.action], controller.signal)
          .then((simulation) => {
            if (controller.signal.aborted) return
            const store = cache.current
            store.set(cacheKey(option.action), simulation.state)
            if (store.size > CACHE_LIMIT) store.delete(store.keys().next().value!)
            setPreviews((previous) => ({
              id,
              states: { ...(previous.id === id ? previous.states : {}), [option.key]: simulation.state },
            }))
          })
          .catch(() => {
            // A refused or failed preview just leaves the option without one.
          })
      }
    }, VICTORY_SIMULATION_DELAY_MS)
    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
    // `id` captures the content of base and options.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, request])

  return previews.id === id ? previews.states : {}
}
