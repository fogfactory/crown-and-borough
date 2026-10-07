import { useEffect, useMemo, useState } from 'react'

import type { SimulationAction, VictorySimulation } from '@/types'

export type VictorySimulationRequest = (
  actions: SimulationAction[],
  signal: AbortSignal,
) => Promise<VictorySimulation>

/** The server refused the hypothetical action (HTTP 422); the message says why. */
export class SimulationRejectedError extends Error {}

/** Delay between the last action and the simulation request. */
export const VICTORY_SIMULATION_DELAY_MS = 150

export interface VictorySimulationState {
  /** Last projection the server accepted; kept while a new one is computed. */
  simulation: VictorySimulation | null
  loading: boolean
  /** Why the server refused the latest actions, if it did. */
  rejection: string | null
  /** The request failed for another reason (network, server). */
  failed: boolean
}

/**
 * Asks the server to project the ordered hypothetical actions on the current
 * state, once the player stops editing. Runs only while enabled.
 */
export function useVictorySimulation(
  actions: SimulationAction[],
  request: VictorySimulationRequest | null,
  enabled: boolean,
  delayMs = VICTORY_SIMULATION_DELAY_MS,
): VictorySimulationState {
  const [state, setState] = useState<VictorySimulationState>({
    simulation: null,
    loading: false,
    rejection: null,
    failed: false,
  })
  const key = useMemo(() => JSON.stringify(actions), [actions])

  useEffect(() => {
    if (!enabled || !request) return
    const controller = new AbortController()
    setState((previous) => ({ ...previous, loading: true }))
    const timer = window.setTimeout(() => {
      request(actions, controller.signal)
        .then((simulation) => {
          if (controller.signal.aborted) return
          setState({ simulation, loading: false, rejection: null, failed: false })
        })
        .catch((error: unknown) => {
          if (controller.signal.aborted) return
          setState((previous) => ({
            simulation: previous.simulation,
            loading: false,
            rejection: error instanceof SimulationRejectedError ? error.message : null,
            failed: !(error instanceof SimulationRejectedError),
          }))
        })
    }, delayMs)
    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
    // `key` captures the actions' content; equal actions must not refetch.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, request, enabled, delayMs])

  return state
}
