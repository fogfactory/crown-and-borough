import { typedHeaderIndex } from '@/lib/order-text'
import { winterSheet, type OrdersDrafts } from '@/lib/orders-body'
import type { OrdersPreview, OrdersPreviewWarning, PlayerId, StateData } from '@/types'

/** An error of a draft; `line` is the 1-based line of that draft, absent when it cannot be pinned down. */
export interface DraftError {
  line?: number
  message: string
  /** Set on a preview warning: `note` informs, `warning` flags an order that will not run as written. */
  level?: OrdersPreviewWarning['level']
  /** Engine rejection reason of a winter line: the client words it (`reports.reason.*`). */
  reason?: string
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

/** Codes of the nobles of `player`. */
function ownedNobleCodes(state: StateData, player: PlayerId): Set<string> {
  return new Set(state.nobles.filter((noble) => noble.owner === player).map((noble) => noble.code))
}

/**
 * Zone and draft line (1-based) of a line of the winter sheet. No special
 * editor in winter: the winter zone then holds the special lines, lineless.
 */
function winterPlacement(
  state: StateData,
  origins: ReturnType<typeof winterSheet>['origins'],
  serverLine: number | undefined,
): { zone: 'winter' | 'special'; line?: number } {
  const origin = serverLine ? origins[serverLine - 1] : undefined
  if (!origin) return { zone: 'winter' }
  if (origin.draft === 'special') {
    return state.season === 'winter' ? { zone: 'winter' } : { zone: 'special', line: origin.line }
  }
  return { zone: 'winter', line: origin.line }
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
 * special drafts, shift the server lines. The lines are right as long as the
 * drafts still send the text the preview answered.
 */
export function mapPreviewErrors(
  state: StateData,
  player: PlayerId,
  drafts: OrdersDrafts,
  preview: OrdersPreview | null | undefined,
): DraftErrors {
  const result: DraftErrors = { chains: {}, winter: [], special: [], other: [] }
  if (!preview) return result
  const ownedNobles = ownedNobleCodes(state, player)
  const { origins } = winterSheet(state, drafts)

  // No special editor in winter: the winter zone tells its errors, lineless.
  const addSpecial = (line: number | undefined, message: string) => {
    if (state.season === 'winter') pushUnique(result.winter, { message })
    else pushUnique(result.special, { line, message })
  }
  const addWinter = (serverLine: number | undefined, message: string) => {
    const place = winterPlacement(state, origins, serverLine)
    pushUnique(result[place.zone], { line: place.line, message })
  }

  for (const error of preview.errors) {
    switch (error.source) {
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
        addSpecial(error.line || undefined, error.message)
        break
    }
  }
  for (const line of preview.winter) {
    if (line.status === 'invalid') addWinter(line.line, line.message ?? '')
  }
  return result
}

/** Preview warnings, sorted by draft like the errors. */
export interface DraftWarnings {
  chains: Record<string, DraftError[]>
  winter: DraftError[]
  special: DraftError[]
  /** Chain warnings naming no noble of the player: no editor holds them. */
  other: DraftError[]
}

/**
 * Sorts the warnings of a preview by draft, with the line numbers of the
 * draft (see `mapPreviewErrors`). A warning from the winter sheet carries the
 * engine's rejection reason in `reason`, which the client words.
 */
export function mapPreviewWarnings(
  state: StateData,
  player: PlayerId,
  drafts: OrdersDrafts,
  preview: OrdersPreview | null | undefined,
): DraftWarnings {
  const result: DraftWarnings = { chains: {}, winter: [], special: [], other: [] }
  if (!preview) return result
  const ownedNobles = ownedNobleCodes(state, player)
  const { origins } = winterSheet(state, drafts)

  for (const warning of preview.warnings ?? []) {
    const base = { message: warning.message, level: warning.level }
    switch (warning.source) {
      case 'chain': {
        if (!warning.noble || !ownedNobles.has(warning.noble)) {
          pushUnique(result.other, base)
          break
        }
        pushUnique((result.chains[warning.noble] ??= []), {
          ...base,
          line: chainDraftLine(warning.noble, drafts.chainDrafts[warning.noble] ?? '', warning.line),
        })
        break
      }
      case 'winter': {
        const place = winterPlacement(state, origins, warning.line)
        pushUnique(result[place.zone], { ...base, line: place.line, reason: warning.code })
        break
      }
    }
  }
  return result
}
