import { useEffect, useMemo, useState } from 'react'

import {
  VICTORY_SIMULATION_DELAY_MS,
  type VictorySimulationRequest,
} from '@/lib/use-victory-simulation'
import type { SimulationAction, StateData } from '@/types'

/**
 * Asks the server for the state each action leads to when applied after the
 * simulated history, so a menu can preview its options. Results are keyed by
 * the option's key; a refused option has no entry. Runs once the selection
 * rests, and not at all without options.
 */
export function useActionPreviews(
  request: VictorySimulationRequest | null,
  base: SimulationAction[],
  options: Array<{ key: string; action: SimulationAction }>,
): Record<string, StateData> {
  const [previews, setPreviews] = useState<{
    id: string
    states: Record<string, StateData>
  }>({
    id: '',
    states: {},
  })
  const id = useMemo(() => JSON.stringify([base, options]), [base, options])

  useEffect(() => {
    if (!request || options.length === 0) return
    const controller = new AbortController()
    const timer = window.setTimeout(() => {
      for (const option of options) {
        request([...base, option.action], controller.signal)
          .then((simulation) => {
            if (controller.signal.aborted) return
            setPreviews((previous) => ({
              id,
              states: {
                ...(previous.id === id ? previous.states : {}),
                [option.key]: simulation.state,
              },
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
