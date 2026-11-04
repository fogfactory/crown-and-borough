import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { MemoryRouter } from 'react-router-dom'

import { BugReportButton } from '@/components/BugReportDialog'
import { LanguageProvider } from '@/i18n/LanguageContext'
import { clearApiCalls } from '@/lib/bug-report'

const getIdToken = vi.hoisted(() => vi.fn(async () => 'test-id-token'))
vi.mock('@/auth/useAuth', () => ({ useAuth: () => ({ getIdToken }) }))

function mockFetch(enabled: boolean, post?: () => Response) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    if (init?.method === 'POST')
      return post?.() ?? json({ url: 'https://github.com/o/r/issues/5' }, 201)
    void input
    return json({ enabled })
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubClipboard(writeText: ReturnType<typeof vi.fn>) {
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText },
    configurable: true,
  })
}

async function openAndFill(path = '/') {
  render(
    <MemoryRouter initialEntries={[path]}>
      <LanguageProvider initialLanguage="en">
        <BugReportButton />
      </LanguageProvider>
    </MemoryRouter>,
  )
  fireEvent.click(screen.getByRole('button', { name: 'Report a bug' }))
  fireEvent.change(await screen.findByLabelText('Short summary'), {
    target: { value: 'Map frozen' },
  })
  fireEvent.change(screen.getByLabelText('What happened?'), {
    target: { value: 'It froze after resolving.' },
  })
}

describe('BugReportButton', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    clearApiCalls()
    vi.stubEnv('VITE_APP_VERSION', '9.9.9')
  })

  it('shows the captured context and the public notice', async () => {
    mockFetch(true)
    await openAndFill()
    expect(screen.getByText(/create[s]? a public issue/i)).toBeTruthy()
    expect(screen.getByLabelText('Captured context (JSON)').textContent).toContain(
      '9.9.9',
    )
  })

  it('sends the game id, never game state, and previews the fetched view', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'POST')
        return json({ url: 'https://github.com/o/r/issues/6' }, 201)
      if (String(input).endsWith('/state'))
        return json({ revision: 4, inviteCode: 'ZZ9' })
      return json({ enabled: true })
    })
    vi.stubGlobal('fetch', fetchMock)
    await openAndFill('/games/g1')
    await waitFor(() =>
      expect(screen.getByLabelText('Captured context (JSON)').textContent).toContain(
        '"revision": 4',
      ),
    )
    expect(screen.getByLabelText('Captured context (JSON)').textContent).not.toContain(
      'ZZ9',
    )
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))
    fireEvent.click(screen.getByRole('button', { name: 'Publish the issue' }))
    await screen.findByText(/your report was published/)
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    const body = JSON.parse(post[1]!.body as string)
    expect(body.gameId).toBe('g1')
    expect(body.context.game).toBeUndefined()
    expect(JSON.stringify(body)).not.toContain('revision')
  })

  it('asks for confirmation before publishing, then posts the report', async () => {
    const fetchMock = mockFetch(true)
    await openAndFill()
    const send = screen.getByRole('button', { name: 'Send' })
    await waitFor(() => expect((send as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(send)
    expect(screen.getByRole('alert').textContent).toMatch(/public issue/)
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'POST')).toBe(false)

    fireEvent.click(screen.getByRole('button', { name: 'Publish the issue' }))
    expect(await screen.findByText(/your report was published/)).toBeTruthy()
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    const body = JSON.parse(post[1]!.body as string)
    expect(body.title).toBe('Map frozen')
    expect(body.context.version).toBe('9.9.9')
    expect(JSON.stringify(body)).not.toContain('test-id-token')
    expect(
      screen.getByRole('link', { name: 'View the issue' }).getAttribute('href'),
    ).toBe('https://github.com/o/r/issues/5')
  })

  it('disables Send when the server has no token configured', async () => {
    mockFetch(false)
    await openAndFill()
    await screen.findAllByText(/not enabled on this server/)
    expect(
      (screen.getByRole('button', { name: 'Send' }) as HTMLButtonElement).disabled,
    ).toBe(true)
    expect(
      (screen.getByRole('button', { name: 'Fill in on GitHub' }) as HTMLButtonElement)
        .disabled,
    ).toBe(false)
  })

  it('shows an error when the server refuses the report', async () => {
    mockFetch(true, () => json({ error: 'rate_limited', message: 'slow down' }, 429))
    await openAndFill()
    const send = screen.getByRole('button', { name: 'Send' })
    await waitFor(() => expect((send as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(send)
    fireEvent.click(screen.getByRole('button', { name: 'Publish the issue' }))
    expect((await screen.findByRole('alert')).textContent).toMatch(/could not be sent/)
  })

  it('copies the JSON and opens the prefilled GitHub issue', async () => {
    mockFetch(true)
    const writeText = vi.fn().mockResolvedValue(undefined)
    stubClipboard(writeText)
    const open = vi.spyOn(window, 'open').mockReturnValue(null)
    await openAndFill()
    fireEvent.click(screen.getByRole('button', { name: 'Fill in on GitHub' }))
    await screen.findByText(/copied to your clipboard: paste it/)
    expect(JSON.parse(writeText.mock.calls[0][0]).version).toBe('9.9.9')
    const url = new URL(open.mock.calls[0][0] as string)
    expect(url.pathname).toBe('/fogfactory/crown-and-borough/issues/new')
    expect(url.searchParams.get('title')).toBe('Map frozen')
    expect(url.searchParams.get('body')).toContain('It froze after resolving.')
    open.mockRestore()
  })

  it('tells the player when the clipboard is unavailable', async () => {
    mockFetch(true)
    stubClipboard(vi.fn().mockRejectedValue(new Error('denied')))
    const open = vi.spyOn(window, 'open').mockReturnValue(null)
    await openAndFill()
    fireEvent.click(screen.getByRole('button', { name: 'Fill in on GitHub' }))
    expect(await screen.findByText(/could not be copied/)).toBeTruthy()
    open.mockRestore()
  })
})
