import { typedHeaderIndex } from '@/lib/order-text'
import { winterSheet, type OrdersDrafts } from '@/lib/orders-body'
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
  /** Chain errors naming no noble of the player: no editor holds them. */
  other: DraftError[]
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

function pushUnique(list: DraftError[], error: DraftError) {
  if (!list.some((known) => known.line === error.line && known.message === error.message)) {
    list.push(error)
  }
}

/**
 * Sorts the errors of a preview by draft, with the line numbers of the draft
 * rather than those of the text `buildOrdersBody` sent: the noble header added
 * to a chain, and the winter sheet `winterSheet` builds from the winter and
 * special drafts, shift the server lines. `drafts` must be the drafts the
 * preview was computed from.
 */
export function mapPreviewErrors(
  state: StateData,
  player: PlayerId,
  drafts: OrdersDrafts,
  preview: OrdersPreview | null | undefined,
): DraftErrors {
  const result: DraftErrors = { chains: {}, winter: [], special: [], other: [] }
  if (!preview) return result
  const ownedNobles = new Set(
    state.nobles.filter((noble) => noble.owner === player).map((noble) => noble.code),
  )
  const { origins } = winterSheet(state, drafts)

  const addWinter = (serverLine: number | undefined, message: string) => {
    const origin = serverLine ? origins[serverLine - 1] : undefined
    if (!origin) {
      pushUnique(result.winter, { message })
    } else if (origin.draft === 'special' && state.season === 'winter') {
      // No special editor in winter: the winter zone tells the error.
      pushUnique(result.winter, { message })
    } else {
      pushUnique(result[origin.draft], { line: origin.line, message })
    }
  }

  const sourceOf = (error: OrdersPreviewError) => error.source ?? (error.noble ? 'chain' : undefined)
  for (const error of preview.errors) {
    switch (sourceOf(error)) {
      case 'chain': {
        if (!error.noble || !ownedNobles.has(error.noble)) {
          pushUnique(result.other, { message: error.message })
          break
        }
        const list = (result.chains[error.noble] ??= [])
        pushUnique(list, {
          line: chainDraftLine(error.noble, drafts.chainDrafts[error.noble] ?? '', error.line),
          message: error.message,
        })
        break
      }
      case 'winter':
        addWinter(error.line, error.message)
        break
      case 'special':
        pushUnique(state.season === 'winter' ? result.winter : result.special, {
          line: state.season === 'winter' ? undefined : error.line || undefined,
          message: error.message,
        })
        break
    }
  }
  for (const line of preview.winter) {
    if (line.status === 'invalid') addWinter(line.line, line.message ?? '')
  }
  return result
}
