import { useMemo } from 'react'

import { buildIntentions, type Intention } from '@/lib/intent-overlay'
import { stripNobleHeader } from '@/lib/order-text'
import { buildWinterIntentions, type WinterIntention } from '@/lib/winter-overlay'
import type {
  MapData,
  PlayerId,
  StateData,
  SubmittedOrdersResponse,
  WinterCosts,
} from '@/types'

const DEFAULT_INTENTIONS_COLOR = '#a84632'

export interface UseGameIntentionsOptions {
  state: StateData | null
  map: MapData | null
  /** Player whose drafts are edited; hotseat passes the selected player. */
  playerID: PlayerId | null
  /** Drafts keyed by noble code for the active player. */
  chainDrafts: Record<string, string>
  /** Winter orders text for the active player. */
  winterDraft: string
  winterCosts: WinterCosts | null
  /** Online spectator: shows installed chains for everyone plus submissions. */
  spectator: boolean
  /** Online spectator only: other players' submitted orders. */
  submittedOrders?: SubmittedOrdersResponse | null
}

export interface GameIntentions {
  intentions: Intention[]
  winterIntentions: WinterIntention[]
  intentionsColor: string
}

/**
 * Shared intent-overlay state for the hotseat and online game screens. The
 * active player sees their own drafts plus installed chains; an online
 * spectator additionally sees every submitted chain and winter investment,
 * colored per player.
 */
export function useGameIntentions({
  state,
  map,
  playerID,
  chainDrafts,
  winterDraft,
  winterCosts,
  spectator,
  submittedOrders = null,
}: UseGameIntentionsOptions): GameIntentions {
  const submittedIntentions = useMemo(() => {
    if (!spectator || !state || !map || state.season === 'winter') return []
    return (submittedOrders?.submissions ?? []).flatMap((submission) => {
      const drafts = Object.fromEntries(
        submission.chains.map((chain) => [
          chain.noble,
          stripNobleHeader(chain.noble, chain.text),
        ]),
      )
      const color = state.players.find((player) => player.id === submission.player)?.color
      return buildIntentions(map, state, submission.player, drafts, {
        includeInstalled: false,
        source: 'submitted',
        color,
      })
    })
  }, [map, spectator, state, submittedOrders])

  const installedIntentions = useMemo(() => {
    if (!spectator || !state || !map) return []
    return state.players.flatMap((player) =>
      buildIntentions(
        map,
        state,
        player.id,
        {},
        {
          color: player.color,
          includeInstalledInWinter: state.season === 'winter',
        },
      ),
    )
  }, [map, spectator, state])

  const intentions = useMemo(
    () =>
      spectator
        ? [...installedIntentions, ...submittedIntentions]
        : state && playerID
          ? buildIntentions(map ?? { territories: [] }, state, playerID, chainDrafts, {
              includeInstalledInWinter: state.season === 'winter',
            })
          : [],
    [
      chainDrafts,
      installedIntentions,
      map,
      playerID,
      spectator,
      state,
      submittedIntentions,
    ],
  )

  const winterIntentions = useMemo(() => {
    if (!state || !map || state.season !== 'winter') return []
    if (spectator) {
      return (submittedOrders?.submissions ?? []).flatMap((submission) => {
        const draft = submission.winter?.lines ?? ''
        if (!draft.trim()) return []
        const color = state.players.find(
          (player) => player.id === submission.player,
        )?.color
        return buildWinterIntentions(map, state, submission.player, draft, {
          source: 'submitted',
          color,
          costs: winterCosts,
        })
      })
    }
    if (!playerID) return []
    return buildWinterIntentions(map, state, playerID, winterDraft, {
      costs: winterCosts,
    })
  }, [map, playerID, spectator, state, submittedOrders, winterCosts, winterDraft])

  const intentionsColor =
    state?.players.find((player) => player.id === playerID)?.color ??
    DEFAULT_INTENTIONS_COLOR

  return { intentions, winterIntentions, intentionsColor }
}
