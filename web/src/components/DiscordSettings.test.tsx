import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { DiscordSettings, PREVIEW_DEBOUNCE_MS } from '@/components/DiscordSettings'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { DiscordWebhookStatus } from '@/types'

const tokenProvider = { getIdToken: async () => 'test-id-token' }

const DEFAULTS = {
  fr: { newTurn: 'Tour {turn} de {game}', publicMessage: 'Message dans {game}' },
  en: { newTurn: 'Turn {turn} of {game}', publicMessage: 'Message in {game}' },
}

function status(overrides: Partial<DiscordWebhookStatus> = {}): DiscordWebhookStatus {
  return {
    configured: false,
    templates: { newTurn: '', publicMessage: '' },
    disabled: false,
    defaults: DEFAULTS,
    ...overrides,
  }
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

interface Call {
  method: string
  path: string
  body: Record<string, unknown> | null
}

/** Routes the Discord API of a game; `handlers` override one route each. */
function mockApi(
  initial: DiscordWebhookStatus,
  handlers: Partial<Record<string, (call: Call) => Response>> = {},
) {
  const calls: Call[] = []
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const call: Call = {
      method: init?.method ?? 'GET',
      path: String(input),
      body: init?.body
        ? (JSON.parse(String(init.body)) as Record<string, unknown>)
        : null,
    }
    calls.push(call)
    const route = `${call.method} ${call.path.replace('/api/games/G1/discord-webhook', '')}`
    const handler = handlers[route.trim()]
    if (handler) return handler(call)
    if (route.startsWith('POST /preview')) {
      return json({
        content: `rendered ${String(call.body?.template ?? '')}`,
        length: 12,
        mentionEveryone: String(call.body?.template).includes('@here'),
        mentionRoles: [],
      })
    }
    return json(initial)
  })
  vi.stubGlobal('fetch', fetchMock)
  return calls
}

function renderSettings(onActiveChange?: (active: boolean) => void) {
  return render(
    <LanguageProvider initialLanguage="en">
      <DiscordSettings
        gameId="G1"
        tokenProvider={tokenProvider}
        onActiveChange={onActiveChange}
      />
    </LanguageProvider>,
  )
}

const writes = (calls: Call[]) => calls.filter((call) => call.method !== 'GET')

describe('DiscordSettings', () => {
  beforeEach(() => vi.unstubAllGlobals())
  afterEach(() => vi.useRealTimers())

  it('reports an unavailable channel', async () => {
    mockApi(status(), {
      GET: () => json({ error: 'discord_disabled', message: 'off' }, 404),
    })
    renderSettings()
    expect(
      await screen.findByText('Discord webhooks are not available here.'),
    ).toBeInTheDocument()
  })

  it('asks for the URL before the first save and never sends without one', async () => {
    const calls = mockApi(status())
    renderSettings()
    await screen.findByText('No webhook is configured.')
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Paste the webhook URL first.')).toBeInTheDocument()
    expect(writes(calls).filter((call) => call.method === 'PUT')).toHaveLength(0)
  })

  it('saves the typed URL with the language and templates', async () => {
    const url = 'https://discord.com/api/webhooks/1/secret-token'
    const changes = vi.fn()
    const calls = mockApi(status(), {
      PUT: () =>
        json(
          status({
            configured: true,
            language: 'en',
            templates: { newTurn: '@here go', publicMessage: '' },
          }),
        ),
    })
    renderSettings(changes)
    await screen.findByText('No webhook is configured.')
    fireEvent.change(screen.getByLabelText('Webhook URL'), { target: { value: url } })
    fireEvent.change(screen.getByLabelText('Language of the announcements'), {
      target: { value: 'fr' },
    })
    fireEvent.change(screen.getByLabelText('New turn'), { target: { value: '@here go' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Discord settings saved.')).toBeInTheDocument()
    const put = calls.find((call) => call.method === 'PUT')
    expect(put?.body).toEqual({
      url,
      language: 'fr',
      templates: { newTurn: '@here go', publicMessage: '' },
    })
    expect(changes).toHaveBeenLastCalledWith(true)
    // The URL is neither read back nor displayed once saved.
    expect(
      screen.getByText('A webhook is configured. Its address is kept secret.'),
    ).toBeInTheDocument()
    expect(screen.queryByDisplayValue(url)).not.toBeInTheDocument()
    expect(document.body.textContent).not.toContain('secret-token')
  })

  it('keeps the stored URL when saving settings alone', async () => {
    const calls = mockApi(status({ configured: true, language: 'en' }), {
      PUT: () => json(status({ configured: true, language: 'en' })),
    })
    renderSettings()
    await screen.findByText('A webhook is configured. Its address is kept secret.')
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await screen.findByText('Discord settings saved.')
    const put = calls.find((call) => call.method === 'PUT')
    expect(put?.body).not.toHaveProperty('url')
  })

  it('shows the placeholder default and the unknown variable of a rejected template', async () => {
    mockApi(status({ configured: true, language: 'en' }), {
      PUT: () =>
        json({ error: 'invalid_template', message: 'x', variable: 'player' }, 400),
    })
    renderSettings()
    const field = await screen.findByLabelText('New turn')
    expect(field).toHaveAttribute('placeholder', DEFAULTS.en.newTurn)
    fireEvent.change(field, { target: { value: '{player}' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unknown variable player: use {game}, {turn}, {season} or {url}.',
    )
  })

  it('warns the host when Discord disabled the webhook', async () => {
    mockApi(status({ configured: true, disabled: true, disabledReason: 'webhook_gone' }))
    const changes = vi.fn()
    renderSettings(changes)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Discord refused the webhook (deleted or invalid)',
    )
    expect(changes).toHaveBeenLastCalledWith(false)
  })

  it('tests the webhook and reads the disabled state back when Discord refuses it', async () => {
    const calls = mockApi(status({ configured: true }), {
      'POST /test': () => json({ error: 'webhook_gone', message: 'gone' }, 409),
    })
    renderSettings()
    await screen.findByText('A webhook is configured. Its address is kept secret.')
    fireEvent.click(screen.getByRole('button', { name: 'Send a test' }))
    expect(
      await screen.findByText(
        /Discord refused the webhook \(deleted or invalid\): it is disabled/,
      ),
    ).toBeInTheDocument()
    await waitFor(() =>
      expect(calls.filter((call) => call.method === 'GET').length).toBeGreaterThanOrEqual(
        2,
      ),
    )
  })

  it('tests the typed URL without saving anything or clearing the form', async () => {
    const url = 'https://discord.com/api/webhooks/1/secret-token'
    const calls = mockApi(status(), {
      'POST /test': () => json(status()),
    })
    renderSettings()
    await screen.findByText('No webhook is configured.')
    expect(screen.queryByRole('button', { name: 'Send a test' })).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Webhook URL'), { target: { value: url } })
    fireEvent.change(screen.getByLabelText('New turn'), { target: { value: '@here go' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send a test' }))
    await screen.findByRole('status')
    const sent = writes(calls)
    expect(sent.map((call) => `${call.method} ${call.path}`)).toEqual([
      'POST /api/games/G1/discord-webhook/test',
    ])
    expect(sent[0].body).toEqual({ url, language: 'en' })
    expect(screen.getByLabelText('Webhook URL')).toHaveValue(url)
    expect(screen.getByLabelText('New turn')).toHaveValue('@here go')
  })

  it('removes the webhook after a confirmation', async () => {
    const changes = vi.fn()
    const calls = mockApi(status({ configured: true }), {
      DELETE: () => json(status()),
    })
    renderSettings(changes)
    await screen.findByText('A webhook is configured. Its address is kept secret.')
    fireEvent.click(screen.getByRole('button', { name: 'Remove the webhook' }))
    expect(calls.some((call) => call.method === 'DELETE')).toBe(false)
    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    expect(await screen.findByText('Webhook removed.')).toBeInTheDocument()
    expect(screen.getByText('No webhook is configured.')).toBeInTheDocument()
    expect(changes).toHaveBeenLastCalledWith(false)
  })
})

describe('DiscordSettings preview', () => {
  beforeEach(() => vi.unstubAllGlobals())
  afterEach(() => vi.useRealTimers())

  it('requests one preview per editor once typing pauses, and shows its mentions', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const calls = mockApi(status({ configured: true, language: 'en' }))
    renderSettings()
    const field = await screen.findByLabelText('New turn')
    const previews = () => calls.filter((call) => call.path.endsWith('/preview'))

    await act(() => vi.advanceTimersByTimeAsync(PREVIEW_DEBOUNCE_MS + 10))
    // One initial preview per event, of the (empty) template: the default one.
    expect(
      previews()
        .map((call) => call.body?.event)
        .sort(),
    ).toEqual(['newTurn', 'publicMessage'])

    fireEvent.change(field, { target: { value: '@' } })
    fireEvent.change(field, { target: { value: '@here' } })
    fireEvent.change(field, { target: { value: '@here turn' } })
    await act(() => vi.advanceTimersByTimeAsync(PREVIEW_DEBOUNCE_MS - 100))
    expect(previews()).toHaveLength(2)
    await act(() => vi.advanceTimersByTimeAsync(200))
    expect(previews()).toHaveLength(3)
    expect(previews()[2]?.body).toEqual({
      event: 'newTurn',
      language: 'en',
      template: '@here turn',
    })
    expect(screen.getByText('rendered @here turn')).toBeInTheDocument()
    expect(
      screen.getByText(/Mentions everyone who can see the channel\./),
    ).toBeInTheDocument()
  })

  it('shows a preview error inline', async () => {
    mockApi(status({ configured: true, language: 'en' }), {
      'POST /preview': () =>
        json({ error: 'invalid_template', message: 'x', variable: 'oops' }, 400),
    })
    renderSettings()
    expect(
      (await screen.findAllByText(/No preview: Unknown variable oops/)).length,
    ).toBeGreaterThan(0)
  })
})
