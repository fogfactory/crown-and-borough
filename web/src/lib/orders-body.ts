import { addNobleHeader, hasChainContent } from '@/lib/order-text'
import type { PlayerId, StateData } from '@/types'

/** Request body shared by `POST .../orders` and `POST .../orders/preview`. */
export interface OrdersBody {
  chains: Array<{ noble: string; text: string }>
  winter: Array<{ lines: string }>
  special: Array<{ text: string }>
}

export interface OrdersDrafts {
  chainDrafts: Record<string, string>
  winterDraft: string
  specialDraft: string
}

/**
 * Builds the order submission of `player` from its drafts: one chain per
 * noble able to emit (with its header), the winter sheet in winter, where card
 * discards share the sheet, and the card orders otherwise.
 */
export function buildOrdersBody(
  state: StateData,
  player: PlayerId,
  { chainDrafts, winterDraft, specialDraft }: OrdersDrafts,
): OrdersBody {
  const winterSeason = state.season === 'winter'
  const chains = winterSeason
    ? []
    : state.nobles
        .filter((noble) => noble.owner === player && noble.status !== 'dungeon')
        .map((noble) => ({
          noble: noble.code,
          text: addNobleHeader(noble.code, chainDrafts[noble.code] ?? ''),
        }))
        .filter((chain) => hasChainContent(chain.noble, chain.text))
  const winterLines = [winterDraft, winterSeason ? specialDraft : '']
    .filter((text) => text.trim() !== '')
    .join('\n')
  return {
    chains,
    winter: winterSeason && winterLines !== '' ? [{ lines: winterLines }] : [],
    special: !winterSeason && specialDraft.trim() !== '' ? [{ text: specialDraft }] : [],
  }
}

export function isEmptyOrdersBody(body: OrdersBody): boolean {
  return body.chains.length === 0 && body.winter.length === 0 && body.special.length === 0
}
