import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiRequest } from '@/lib/api'
import {
  API_CALL_BUFFER_SIZE,
  GITHUB_URL_LIMIT,
  MASKED,
  captureBugContext,
  clearApiCalls,
  githubIssueUrl,
  maskSecrets,
  recentApiCalls,
  recordApiCall,
  contextJson,
  currentGameId,
} from '@/lib/bug-report'

beforeEach(() => {
  clearApiCalls()
  vi.unstubAllGlobals()
})

describe('maskSecrets', () => {
  it('masks sensitive keys and token-like strings, deeply', () => {
    const masked = maskSecrets({
      idToken: 'abc',
      nested: { Authorization: 'Bearer xyz', inviteCode: 'ABC123', ok: 'fine' },
      list: ['Bearer sometoken.value', 'plain'],
      jwt: 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig',
    })
    const text = JSON.stringify(masked)
    expect(text).not.toContain('abc"')
    expect(text).not.toContain('xyz')
    expect(text).not.toContain('ABC123')
    expect(text).not.toContain('sometoken')
    expect(text).not.toContain('eyJhbGci')
    expect(text).toContain('fine')
    expect(text).toContain('plain')
    expect((masked as { idToken: string }).idToken).toBe(MASKED)
  })
})

describe('API call buffer', () => {
  it('keeps only the most recent calls', () => {
    for (let i = 0; i < API_CALL_BUFFER_SIZE + 5; i++) {
      recordApiCall('get', `/api/games/${i}`, 200)
    }
    const calls = recentApiCalls()
    expect(calls).toHaveLength(API_CALL_BUFFER_SIZE)
    expect(calls[0].path).toBe('/api/games/5')
    expect(calls[0].method).toBe('GET')
  })

  it('masks sensitive query parameters', () => {
    recordApiCall('GET', '/api/x?token=secret123&lang=fr', 200)
    const [call] = recentApiCalls()
    expect(call.path).not.toContain('secret123')
    expect(call.path).toContain('lang=fr')
  })

  it('keeps escaped query values encoded and survives malformed escapes', () => {
    recordApiCall('GET', '/api/x?q=a%26token%3Dx&bad=%E0%A4%A&sig=s?1', 200)
    const [call] = recentApiCalls()
    expect(call.path).toBe(`/api/x?q=a%26token%3Dx&bad=%E0%A4%A&sig=${MASKED}`)
  })

  it('records apiRequest calls without the Authorization header', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response('{}', {
            status: 404,
            headers: { 'Content-Type': 'application/json' },
          }),
      ),
    )
    await apiRequest(
      { getIdToken: async () => 'firebase-secret-token' },
      '/api/games/g1',
      {
        method: 'POST',
        body: '{"a":1}',
      },
    ).catch(() => undefined)
    const calls = recentApiCalls()
    expect(calls).toHaveLength(1)
    expect(calls[0]).toMatchObject({ method: 'POST', path: '/api/games/g1', status: 404 })
    expect(JSON.stringify(calls)).not.toContain('firebase-secret-token')
  })

  it('records a status 0 when the network fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('offline')))
    await apiRequest({ getIdToken: async () => 't' }, '/api/games').catch(() => undefined)
    expect(recentApiCalls()[0].status).toBe(0)
  })
})

describe('captureBugContext', () => {
  it('captures version, user agent and recent calls', () => {
    vi.stubEnv('VITE_APP_VERSION', '3.1.4')
    recordApiCall('GET', '/api/games', 200)
    const context = captureBugContext('fr')
    expect(context.version).toBe('3.1.4')
    expect(context.userAgent).toBe(navigator.userAgent)
    expect(context.language).toBe('fr')
    expect(context.recentApiCalls).toHaveLength(1)
  })
})

describe('contextJson', () => {
  it('adds the masked game view', () => {
    const json = contextJson(captureBugContext('en'), {
      gameId: 'g1',
      inviteCode: 'ZZZ999',
    })
    expect(json).toContain('g1')
    expect(json).not.toContain('ZZZ999')
    expect(JSON.parse(contextJson(captureBugContext('en'), null)).game).toBeNull()
  })
})

describe('currentGameId', () => {
  it('reads the game id of a game page only', () => {
    expect(currentGameId('/games/abc%20d')).toBe('abc d')
    expect(currentGameId('/rules')).toBeNull()
  })
})

describe('githubIssueUrl', () => {
  it('targets the repository issue form', () => {
    const url = new URL(githubIssueUrl('A title', 'A body'))
    expect(url.origin + url.pathname).toBe(
      'https://github.com/fogfactory/crown-and-borough/issues/new',
    )
    expect(url.searchParams.get('title')).toBe('A title')
    expect(url.searchParams.get('body')).toBe('A body')
  })

  it('trims the body to stay under the URL limit', () => {
    const url = githubIssueUrl('t', 'é & ligne\n'.repeat(3000))
    expect(url.length).toBeLessThanOrEqual(GITHUB_URL_LIMIT)
    expect(new URL(url).searchParams.get('body')!.length).toBeGreaterThan(100)
  })

  it('never splits an emoji when trimming', () => {
    const body = new URL(githubIssueUrl('t', '😀'.repeat(3000))).searchParams.get('body')!
    expect(body).not.toContain('�')
    expect(body).toMatch(/^(😀)+…$/u)
  })
})
