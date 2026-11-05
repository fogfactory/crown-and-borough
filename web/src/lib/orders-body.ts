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

const CARD_ORDER = /^\s*(R\s+N|C\s+N|D\s+N)\s/i

/** Draft and 1-based line that a line of the winter sheet comes from. */
export interface WinterLineOrigin {
  draft: 'winter' | 'special'
  line: number
}

/**
 * The winter sheet sent for the drafts, one origin per line: in winter the
 * winter draft then the special one (card discards share the sheet), otherwise
 * only the lines that play a noble-deck card, the only winter-sheet orders
 * accepted outside winter.
 */
export function winterSheet(
  state: StateData,
  { winterDraft, specialDraft }: Pick<OrdersDrafts, 'winterDraft' | 'specialDraft'>,
): { lines: string; origins: WinterLineOrigin[] } {
  const parts: Array<[string, WinterLineOrigin['draft']]> =
    state.season === 'winter'
      ? [
          [winterDraft, 'winter'],
          [specialDraft, 'special'],
        ]
      : [[winterDraft, 'winter']]
  const kept = parts
    .filter(([text]) => text.trim() !== '')
    .flatMap(([text, draft]) =>
      text.split('\n').map((line, index) => ({ text: line, origin: { draft, line: index + 1 } })),
    )
    .filter(({ text }) => state.season === 'winter' || CARD_ORDER.test(text))
  return {
    lines: kept.map(({ text }) => text).join('\n'),
    origins: kept.map(({ origin }) => origin),
  }
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
  const winterLines = winterSheet(state, { winterDraft, specialDraft }).lines
  return {
    chains,
    winter: winterLines !== '' ? [{ lines: winterLines }] : [],
    special: !winterSeason && specialDraft.trim() !== '' ? [{ text: specialDraft }] : [],
  }
}

export function isEmptyOrdersBody(body: OrdersBody): boolean {
  return body.chains.length === 0 && body.winter.length === 0 && body.special.length === 0
}
