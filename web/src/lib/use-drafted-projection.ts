import { useEffect, useMemo, useState } from 'react'

import type { SupplyFetcher } from '@/lib/use-supply-and-transfer'
import type { PlayerId, StateData } from '@/types'

export interface UseDraftedProjectionOptions {
  state: StateData | null
  playerId: PlayerId | null
  /** The viewer's drafted special-card orders. */
  specialDraft?: string
  /** Resource root of the projection endpoint ('/api/games/{id}'). */
  basePath?: string
  fetcher: SupplyFetcher
}

const DEBOUNCE_MS = 300

/**
 * The state whose income, mill and ration projections also reflect the
 * regional bonus cards (fair weather, abundant harvest) the viewer drafted:
 * the server replays the weather of the season with them. Without a draft,
 * a spectator, a winter state or a failed request, the state is returned as
 * is, so the treasury never shows less than the plain projection.
 */
export function useDraftedProjection({
  state,
  playerId,
  specialDraft = '',
  basePath = '/api',
  fetcher,
}: UseDraftedProjectionOptions): StateData | null {
  const [overlay, setOverlay] = useState<{ scope: string; projected: StateData } | null>(null)
  const draft = specialDraft.trim()
  const enabled = Boolean(
    state && playerId && draft !== '' && state.season !== 'winter' && basePath !== '/api',
  )
  // The previous projection stays on display while the next one loads, so the
  // figures do not flicker back to the plain forecast at every keystroke.
  const scope = enabled ? `${state?.turn}|${playerId}` : ''

  useEffect(() => {
    if (!enabled) return
    const controller = new AbortController()
    const timer = setTimeout(() => {
      const query = new URLSearchParams({ special: specialDraft })
      fetcher<StateData>(`${basePath}/projection?${query.toString()}`, controller.signal)
        .then((projected) => {
          if (!controller.signal.aborted && projected && Array.isArray(projected.players)) {
            setOverlay({ scope, projected })
          }
        })
        .catch(() => {
          // The plain projection stays on display.
        })
    }, DEBOUNCE_MS)
    return () => {
      clearTimeout(timer)
      controller.abort()
    }
    // A new state (a game update) projects the draft again.
  }, [basePath, enabled, fetcher, scope, specialDraft, state])

  return useMemo(() => {
    if (!state || !enabled || !overlay || overlay.scope !== scope) return state
    const projected = overlay.projected
    return {
      ...state,
      players: state.players.map((player) => {
        const next = projected.players.find((candidate) => candidate.id === player.id)
        return next
          ? {
              ...player,
              projectedIncome: next.projectedIncome,
              projectedMillIncome: next.projectedMillIncome,
              projectedConsumption: next.projectedConsumption,
              armiesAtRisk: next.armiesAtRisk,
            }
          : player
      }),
      forecastEffects: projected.forecastEffects ?? [],
      fiefs: state.fiefs?.map((fief) => {
        const next = projected.fiefs?.find((candidate) => candidate.capital === fief.capital)
        return next ? { ...fief, projectedIncome: next.projectedIncome } : fief
      }),
    }
  }, [enabled, scope, overlay, state])
}
