import { useMemo } from 'react'

import { buildIntentions, type Intention } from '@/lib/intent-overlay'
import { draftOrdersByNoble } from '@/lib/transfer-preview'
import { buildWinterIntentions, type WinterIntention } from '@/lib/winter-overlay'
import type {
  MapData,
  OrdersPreview,
  PlayerId,
  StateData,
  SubmittedOrdersResponse,
} from '@/types'

const DEFAULT_INTENTIONS_COLOR = '#a84632'

export interface UseGameIntentionsOptions {
  state: StateData | null
  map: MapData | null
  /** Player whose drafts are edited; hotseat passes the selected player. */
  playerID: PlayerId | null
  /** Server dry run of the active player's drafts. */
  preview: OrdersPreview | null
  /** Winter sheet sent in the preview request, used to label winter markers. */
  winterText: string
  /** Online spectator: shows installed chains for everyone plus submissions. */
  spectator: boolean
  /** Online spectator only: other players' submitted orders and their preview. */
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
 * colored per player. Drafts are drawn from the server preview, so the client
 * never parses or simulates orders itself.
 */
export function useGameIntentions({
  state,
  map,
  playerID,
  preview,
  winterText,
  spectator,
  submittedOrders = null,
}: UseGameIntentionsOptions): GameIntentions {
  const playerColor = (player: PlayerId) =>
    state?.players.find((candidate) => candidate.id === player)?.color

  const submittedIntentions = useMemo(() => {
    if (!spectator || !state || !map || state.season === 'winter') return []
    return (submittedOrders?.submissions ?? []).flatMap((submission) =>
      buildIntentions(
        map,
        state,
        submission.player,
        draftOrdersByNoble(submission.preview?.chains),
        {
          includeInstalled: false,
          source: 'submitted',
          color: state.players.find((player) => player.id === submission.player)?.color,
        },
      ),
    )
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
          ? buildIntentions(
              map ?? { territories: [] },
              state,
              playerID,
              draftOrdersByNoble(preview?.chains),
              { includeInstalledInWinter: state.season === 'winter' },
            )
          : [],
    [installedIntentions, map, playerID, preview, spectator, state, submittedIntentions],
  )

  const winterIntentions = useMemo(() => {
    if (!state || state.season !== 'winter') return []
    if (spectator) {
      return (submittedOrders?.submissions ?? []).flatMap((submission) =>
        buildWinterIntentions(
          submission.preview?.winter ?? [],
          submission.winter?.lines ?? '',
          {
            source: 'submitted',
            color: state.players.find((player) => player.id === submission.player)?.color,
          },
        ),
      )
    }
    if (!playerID) return []
    return buildWinterIntentions(preview?.winter ?? [], winterText)
  }, [playerID, preview, spectator, state, submittedOrders, winterText])

  const intentionsColor =
    (playerID ? playerColor(playerID) : undefined) ?? DEFAULT_INTENTIONS_COLOR

  return { intentions, winterIntentions, intentionsColor }
}
