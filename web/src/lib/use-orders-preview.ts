import { useEffect, useMemo, useState } from 'react'

import { isEmptyOrdersBody, type OrdersBody } from '@/lib/orders-body'
import type { OrdersPreview } from '@/types'

export type OrdersPreviewRequest = (
  body: OrdersBody,
  signal: AbortSignal,
) => Promise<OrdersPreview>

/** Delay between the last keystroke and the preview request. */
export const ORDERS_PREVIEW_DELAY_MS = 300

/**
 * Asks the server to dry-run the current draft once the player stops typing.
 * The previous preview stays visible while a new one is computed; an empty
 * draft or a failed request clears it, since the preview is only a hint.
 */
export function useOrdersPreview(
  body: OrdersBody | null,
  request: OrdersPreviewRequest | null,
  delayMs = ORDERS_PREVIEW_DELAY_MS,
): OrdersPreview | null {
  const [preview, setPreview] = useState<OrdersPreview | null>(null)
  const key = useMemo(() => (body ? JSON.stringify(body) : ''), [body])

  useEffect(() => {
    if (!body || !request || isEmptyOrdersBody(body)) {
      setPreview(null)
      return
    }
    const controller = new AbortController()
    const timer = window.setTimeout(() => {
      request(body, controller.signal)
        .then((next) => {
          if (!controller.signal.aborted) setPreview(next)
        })
        .catch(() => {
          if (!controller.signal.aborted) setPreview(null)
        })
    }, delayMs)
    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
    // `key` captures the body content; a new but equal body must not refetch.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, request, delayMs])

  return preview
}
