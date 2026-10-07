import { useEffect, useMemo, useState } from 'react'

import type { VictoryScenario, VictorySimulation } from '@/types'

export type VictorySimulationRequest = (
  scenario: VictoryScenario,
  signal: AbortSignal,
) => Promise<VictorySimulation>

/** Delay between the last scenario change and the simulation request. */
export const VICTORY_SIMULATION_DELAY_MS = 300

export const emptyScenario: VictoryScenario = {
  marriages: [],
  marriageEnds: [],
  deaths: [],
  claims: [],
}

export interface VictorySimulationState {
  simulation: VictorySimulation | null
  loading: boolean
  failed: boolean
}

/**
 * Asks the server to project the scenario once the player stops editing it.
 * The previous projection stays visible while a new one is computed. An empty
 * scenario is still sent, so the dialog shows the current reading at once.
 */
export function useVictorySimulation(
  scenario: VictoryScenario,
  request: VictorySimulationRequest | null,
  enabled = true,
  delayMs = VICTORY_SIMULATION_DELAY_MS,
): VictorySimulationState {
  const [state, setState] = useState<VictorySimulationState>({
    simulation: null,
    loading: false,
    failed: false,
  })
  const key = useMemo(() => JSON.stringify(scenario), [scenario])

  useEffect(() => {
    if (!enabled || !request) return
    const controller = new AbortController()
    setState((previous) => ({ ...previous, loading: true }))
    const timer = window.setTimeout(() => {
      request(scenario, controller.signal)
        .then((simulation) => {
          if (!controller.signal.aborted) setState({ simulation, loading: false, failed: false })
        })
        .catch(() => {
          if (!controller.signal.aborted)
            setState({ simulation: null, loading: false, failed: true })
        })
    }, delayMs)
    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
    // `key` captures the scenario content; an equal scenario must not refetch.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, request, enabled, delayMs])

  return state
}
