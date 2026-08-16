import { afterEach, describe, expect, it, vi } from 'vitest'

import { apiRequest } from '@/lib/api'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('authenticated API client', () => {
  it('gets a token immediately before sending a REST command', async () => {
    const getIdToken = vi.fn(async () => 'firebase-token')
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      expect(init?.headers).toBeInstanceOf(Headers)
      expect(new Headers(init?.headers).get('Authorization')).toBe(
        'Bearer firebase-token',
      )
      return new Response(JSON.stringify({ status: 'pending' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await apiRequest<{ status: string }>(
      { getIdToken },
      '/api/games/game-1/orders',
      { method: 'POST', body: JSON.stringify({ chains: [], winter: [] }) },
    )

    expect(result).toEqual({ status: 'pending' })
    expect(getIdToken).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('does not retry or force-refresh a rejected token', async () => {
    const getIdToken = vi.fn(async () => 'expired-token')
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({ code: 'unauthorized', message: 'token rejected' }),
            {
              status: 401,
              headers: { 'Content-Type': 'application/json' },
            },
          ),
      ),
    )

    await expect(apiRequest({ getIdToken }, '/api/auth/me')).rejects.toMatchObject({
      status: 401,
      code: 'unauthorized',
    })
    expect(getIdToken).toHaveBeenCalledOnce()
  })

  it('fails before fetch when the session has no current token', async () => {
    const getIdToken = vi.fn(async () => null)
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    await expect(apiRequest({ getIdToken }, '/api/games')).rejects.toMatchObject({
      status: 401,
      code: 'unauthorized',
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
