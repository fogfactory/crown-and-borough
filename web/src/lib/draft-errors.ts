import { typedHeaderIndex } from '@/lib/order-text'
import { keepCardOrders, type OrdersDrafts } from '@/lib/orders-body'
import type { OrdersPreview, OrdersPreviewError, PlayerId, StateData } from '@/types'

/** An error of a draft; `line` is the 1-based line of that draft, absent when it cannot be pinned down. */
export interface DraftError {
  line?: number
  message: string
}

/** Preview errors, sorted by the draft that holds them. */
export interface DraftErrors {
  chains: Record<string, DraftError[]>
  winter: DraftError[]
  special: DraftError[]
}

/**
 * Draft line (1-based) of the line `serverLine` of the chain text built by
 * `addNobleHeader`: the noble code on line 1, then the draft without its typed
 * header, if it had one.
 */
export function chainDraftLine(
  nobleCode: string,
  draft: string,
  serverLine: number | undefined,
): number | undefined {
  if (!serverLine || serverLine < 1) return undefined
  const headerIndex = typedHeaderIndex(nobleCode, draft)
  if (serverLine === 1) return headerIndex >= 0 ? headerIndex + 1 : undefined
  const kept = serverLine - 2
  return (headerIndex >= 0 && kept >= headerIndex ? kept + 1 : kept) + 1
}

/** Draft line (1-based) of the line `serverLine` of the card orders `keepCardOrders` kept. */
function cardDraftLine(draft: string, serverLine: number): number | undefined {
  const cardLines = keepCardOrders(draft).split('\n')
  const kept = cardLines[serverLine - 1]
  if (kept === undefined) return undefined
  let seen = 0
  const lines = draft.split('\n')
  for (let index = 0; index < lines.length; index++) {
    if (keepCardOrders(lines[index]) === '') continue
    seen++
    if (seen === serverLine) return index + 1
  }
  return undefined
}

function pushUnique(list: DraftError[], error: DraftError) {
  if (!list.some((known) => known.line === error.line && known.message === error.message)) {
    list.push(error)
  }
}

/**
 * Sorts the errors of a preview by draft, with the line numbers of the draft
 * rather than those of the text `buildOrdersBody` sent: the noble header added
 * to a chain, the winter and special drafts joined in winter, and the lines
 * `keepCardOrders` dropped outside winter all shift the server lines.
 */
export function mapPreviewErrors(
  state: StateData,
  player: PlayerId,
  { chainDrafts, winterDraft }: OrdersDrafts,
  preview: OrdersPreview | null | undefined,
): DraftErrors {
  const result: DraftErrors = { chains: {}, winter: [], special: [] }
  if (!preview) return result
  const winterSeason = state.season === 'winter'
  const ownedNobles = new Set(
    state.nobles.filter((noble) => noble.owner === player).map((noble) => noble.code),
  )
  // A text sent as winter lines: the winter draft, then in winter the special one.
  const winterHasText = winterDraft.trim() !== ''
  const winterLength = winterHasText ? winterDraft.split('\n').length : 0

  const addWinter = (serverLine: number | undefined, message: string) => {
    if (!serverLine || serverLine < 1) {
      pushUnique(result.winter, { message })
    } else if (winterSeason) {
      if (serverLine <= winterLength) {
        pushUnique(result.winter, { line: serverLine, message })
      } else {
        pushUnique(result.special, { line: serverLine - winterLength, message })
      }
    } else {
      pushUnique(result.winter, { line: cardDraftLine(winterDraft, serverLine), message })
    }
  }

  const sourceOf = (error: OrdersPreviewError) => error.source ?? (error.noble ? 'chain' : undefined)
  for (const error of preview.errors) {
    switch (sourceOf(error)) {
      case 'chain': {
        if (!error.noble || !ownedNobles.has(error.noble)) break
        const list = (result.chains[error.noble] ??= [])
        pushUnique(list, {
          line: chainDraftLine(error.noble, chainDrafts[error.noble] ?? '', error.line),
          message: error.message,
        })
        break
      }
      case 'winter':
        addWinter(error.line, error.message)
        break
      case 'special':
        pushUnique(result.special, { line: error.line || undefined, message: error.message })
        break
    }
  }
  for (const line of preview.winter) {
    if (line.status === 'invalid') addWinter(line.line, line.message ?? '')
  }
  return result
}
