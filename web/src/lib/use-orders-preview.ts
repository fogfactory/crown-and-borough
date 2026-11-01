import { useEffect, useMemo, useState } from 'react'

import { isEmptyOrdersBody, type OrdersBody } from '@/lib/orders-body'
import type { OrdersPreview } from '@/types'

export type OrdersPreviewRequest = (
  body: OrdersBody,
  signal: AbortSignal,
) => Promise<OrdersPreview>

/** Delay between the last keystroke and the preview request. */
export const ORDERS_PREVIEW_DELAY_MS = 300

/** A preview and the body it answers. */
export interface AnsweredPreview {
  preview: OrdersPreview | null
  /** Body the preview was computed from; null without a preview. */
  body: OrdersBody | null
}

const NO_PREVIEW: AnsweredPreview = { preview: null, body: null }

/** Whether a reply has the shape of a preview the panel can read. */
function isOrdersPreview(reply: unknown): reply is OrdersPreview {
  const preview = reply as Partial<OrdersPreview> | null
  return (
    typeof preview === 'object' &&
    preview !== null &&
    Array.isArray(preview.errors) &&
    Array.isArray(preview.chains) &&
    Array.isArray(preview.winter)
  )
}

/**
 * Asks the server to dry-run the current draft once the player stops typing.
 * The previous preview stays visible while a new one is computed, next to the
 * body it answers; an empty draft or a failed request clears it, since the
 * preview is only a hint.
 */
export function useOrdersPreview(
  body: OrdersBody | null,
  request: OrdersPreviewRequest | null,
  delayMs = ORDERS_PREVIEW_DELAY_MS,
): AnsweredPreview {
  const [answered, setAnswered] = useState<AnsweredPreview>(NO_PREVIEW)
  const key = useMemo(() => (body ? JSON.stringify(body) : ''), [body])

  useEffect(() => {
    if (!body || !request || isEmptyOrdersBody(body)) {
      setAnswered(NO_PREVIEW)
      return
    }
    const controller = new AbortController()
    const timer = window.setTimeout(() => {
      request(body, controller.signal)
        .then((next) => {
          if (controller.signal.aborted) return
          // A reply without the lists of a preview is no preview: nothing is shown.
          setAnswered(isOrdersPreview(next) ? { preview: next, body } : NO_PREVIEW)
        })
        .catch(() => {
          if (!controller.signal.aborted) setAnswered(NO_PREVIEW)
        })
    }, delayMs)
    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
    // `key` captures the body content; a new but equal body must not refetch.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, request, delayMs])

  return answered
}
