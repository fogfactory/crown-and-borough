import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { OrdersBody } from '@/lib/orders-body'
import { useOrdersPreview, type OrdersPreviewRequest } from '@/lib/use-orders-preview'
import type { OrdersPreview } from '@/types'

const preview: OrdersPreview = { errors: [], chains: [], winter: [] }
const body = (text: string): OrdersBody => ({
  chains: [{ noble: 'HUG', text }],
  winter: [],
  special: [],
})

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useOrdersPreview', () => {
  it('waits for the player to stop typing before asking the server once', async () => {
    const request = vi.fn<OrdersPreviewRequest>(() => Promise.resolve(preview))
    const { result, rerender } = renderHook(
      ({ draft }) => useOrdersPreview(draft, request),
      { initialProps: { draft: body('HUG\nROS') } },
    )

    rerender({ draft: body('HUG\nROS A') })
    rerender({ draft: body('HUG\nROS A BRU') })
    expect(request).not.toHaveBeenCalled()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.calls[0][0]).toEqual(body('HUG\nROS A BRU'))
    expect(result.current).toBe(preview)
  })

  it('does not refetch for an equal draft', async () => {
    const request = vi.fn<OrdersPreviewRequest>(() => Promise.resolve(preview))
    const { rerender } = renderHook(({ draft }) => useOrdersPreview(draft, request), {
      initialProps: { draft: body('HUG\nROS H') },
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    rerender({ draft: body('HUG\nROS H') })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(request).toHaveBeenCalledTimes(1)
  })

  it('clears the preview for an empty draft or a failed request', async () => {
    const request = vi.fn<OrdersPreviewRequest>(() => Promise.resolve(preview))
    const { result, rerender } = renderHook(
      ({ draft, send }) => useOrdersPreview(draft, send),
      { initialProps: { draft: body('HUG\nROS H'), send: request } },
    )
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(result.current).toBe(preview)

    rerender({ draft: { chains: [], winter: [], special: [] }, send: request })
    expect(result.current).toBeNull()

    const failing = vi.fn<OrdersPreviewRequest>(() => Promise.reject(new Error('down')))
    rerender({ draft: body('HUG\nROS A BRU'), send: failing })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(result.current).toBeNull()
  })
})
