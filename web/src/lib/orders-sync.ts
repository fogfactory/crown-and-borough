import { stripNobleHeader } from '@/lib/order-text'
import type { OrdersBody } from '@/lib/orders-body'
import type { MySubmissionResponse } from '@/types'

/** Orders reduced to what the server stores, comparable whatever the objects holding them. */
export interface NormalizedOrders {
  /** Chain per noble code, without its header; empty chains are left out. */
  chains: Record<string, string>
  winter: string
  special: string
}

/** Where the local orders differ from the server version, per sheet. */
export interface OrdersDifference {
  any: boolean
  chains: Record<string, boolean>
  winter: boolean
  special: boolean
}

/** Trailing spaces, line endings and blank edges do not make orders different. */
export function normalizeOrdersText(text: string): string {
  return text
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.trimEnd())
    .join('\n')
    .trim()
}

/** The content of a body sent to the server (or read back from it), normalized. */
export function normalizeOrders(body: {
  chains?: Array<{ noble: string; text: string }>
  winter?: Array<{ lines: string }>
  special?: Array<{ text: string }>
}): NormalizedOrders {
  const chains: Record<string, string> = {}
  for (const chain of body.chains ?? []) {
    const text = normalizeOrdersText(stripNobleHeader(chain.noble, chain.text))
    if (text !== '') chains[chain.noble] = text
  }
  return {
    chains,
    winter: normalizeOrdersText((body.winter ?? []).map((entry) => entry.lines).join('\n')),
    special: normalizeOrdersText((body.special ?? []).map((entry) => entry.text).join('\n')),
  }
}

/** What the server holds as a body: the draft, the sealed orders, or nothing when pending. */
export function submissionBody(submission: MySubmissionResponse | null): OrdersBody {
  return {
    chains: submission?.chains ?? [],
    winter: submission?.winter && submission.winter.lines !== '' ? [submission.winter] : [],
    special: submission?.special ?? [],
  }
}

/**
 * Compares the local orders with the server version. A pending server version
 * is empty: any typed order differs from it. Independent of the status.
 */
export function compareOrders(local: NormalizedOrders, server: NormalizedOrders): OrdersDifference {
  const chains: Record<string, boolean> = {}
  for (const noble of new Set([...Object.keys(local.chains), ...Object.keys(server.chains)])) {
    if ((local.chains[noble] ?? '') !== (server.chains[noble] ?? '')) chains[noble] = true
  }
  const winter = local.winter !== server.winter
  const special = local.special !== server.special
  return { any: Object.keys(chains).length > 0 || winter || special, chains, winter, special }
}
