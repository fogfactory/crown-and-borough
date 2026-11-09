import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ChatPanel } from '@/components/ChatPanel'
import { MapLinksProvider, type MapLinks } from '@/components/MapLinks'
import { LanguageProvider } from '@/i18n/LanguageContext'
import { toChatTimestamp, type ChatMessage } from '@/lib/chat'
import type { UseChatResult } from '@/lib/use-chat'
import type { MapData, StateData } from '@/types'

const chat = vi.hoisted(() => ({ result: null as unknown }))
vi.mock('@/lib/use-chat', () => ({ useChat: () => chat.result }))

const markChatRead = vi.hoisted(() => vi.fn().mockResolvedValue({}))
const postChatMessage = vi.hoisted(() => vi.fn().mockResolvedValue({}))
vi.mock('@/lib/chat-api', () => ({ markChatRead, postChatMessage }))

function message(
  id: string,
  author: string,
  turn: number,
  channel = 'public',
  at = `2026-01-01T10:00:0${id}Z`,
): ChatMessage {
  return {
    id,
    channel,
    authorHouseId: author,
    body: `body ${id}`,
    createdAt: toChatTimestamp(at) as string,
    turn: { number: turn, season: 'spring', year: 1011 },
  }
}

const state = {
  turn: 2,
  season: 'summer',
  year: 1011,
  players: [
    { id: 'P1', name: 'Maison Aubrac', color: '#a33' },
    { id: 'P2', name: 'Maison Brieuc', color: '#33a' },
    { id: 'P3', name: 'Maison Cérès', color: '#3a3' },
    { id: 'P4', name: 'Maison Dorian', color: '#aa3' },
  ],
  // P4 owns nothing: eliminated.
  territories: [
    { id: 'AAA', owner: 'P1', army: null },
    { id: 'BBB', owner: 'P2', army: null },
    { id: 'CCC', owner: 'P3', army: null },
  ],
  fiefs: [],
} as unknown as StateData

function setChat(
  channels: UseChatResult['channels'],
  lastRead: UseChatResult['lastRead'] = {},
) {
  chat.result = { channels, lastRead, loadMore: vi.fn() } satisfies UseChatResult
}

function channel(messages: ChatMessage[]) {
  return { messages, loaded: true, hasMore: false, loadingMore: false, error: null }
}

function renderPanel(props: Partial<Parameters<typeof ChatPanel>[0]> = {}) {
  const onUnreadChange = vi.fn()
  render(
    <LanguageProvider initialLanguage="en">
      <ChatPanel
        gameId="g1"
        uid="u1"
        tokenProvider={{ getIdToken: async () => 'token' }}
        state={state}
        house="P1"
        observer={false}
        visible={false}
        onUnreadChange={onUnreadChange}
        {...props}
      />
    </LanguageProvider>,
  )
  return { onUnreadChange }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('ChatPanel', () => {
  it('shows house names with separators, never account data', () => {
    setChat({
      public: channel([message('1', 'P2', 1), message('2', 'P1', 2)]),
    })
    renderPanel({ visible: true })
    const log = screen.getByRole('log', { name: /Public room/ })
    expect(within(log).getByText('Maison Brieuc')).toBeInTheDocument()
    expect(within(log).getAllByRole('separator')).toHaveLength(2)
    expect(within(log).getAllByText('Spring 1011')).toHaveLength(2)
  })

  it('offers new conversations only with houses still in play', () => {
    setChat({
      public: channel([]),
      P1_P2: channel([]),
      P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
      P1_P4: channel([]),
    })
    renderPanel()
    const options = within(screen.getByLabelText('New conversation')).getAllByRole(
      'option',
    )
    expect(options.map((option) => option.textContent)).toEqual([
      'Write to a house…',
      'Maison Brieuc',
    ])
  })

  it('keeps an existing conversation with an eliminated house usable', () => {
    setChat({
      public: channel([]),
      P1_P4: channel([message('1', 'P4', 1, 'P1_P4')]),
    })
    renderPanel()
    fireEvent.change(screen.getByRole('combobox', { name: 'Rooms and conversations' }), {
      target: { value: 'P1_P4' },
    })
    expect(screen.getByLabelText('Your message')).toBeEnabled()
  })

  it('counts unread messages per channel and in total', () => {
    setChat(
      {
        public: channel([message('1', 'P2', 1), message('2', 'P1', 1)]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2'), message('4', 'P2', 1, 'P1_P2')]),
      },
      { public: toChatTimestamp('2026-01-01T09:00:00Z') as string },
    )
    const { onUnreadChange } = renderPanel()
    expect(onUnreadChange).toHaveBeenLastCalledWith(3)
    expect(screen.getByRole('option', { name: 'Public room (1)' })).toBeInTheDocument()
  })

  it('marks the open channel read when the chat is on screen', () => {
    setChat({ public: channel([message('1', 'P2', 1)]) }, {})
    renderPanel({ visible: true })
    expect(markChatRead).toHaveBeenCalledWith(expect.anything(), 'g1', 'public')
  })

  it('does not mark anything read while hidden', () => {
    setChat({ public: channel([message('1', 'P2', 1)]) }, {})
    renderPanel({ visible: false })
    expect(markChatRead).not.toHaveBeenCalled()
  })

  it('shows another player’s conversation to the observer as read only', () => {
    setChat({
      public: channel([]),
      P2_P3: channel([message('1', 'P2', 1, 'P2_P3')]),
    })
    renderPanel({ house: null, observer: true })
    fireEvent.change(screen.getByRole('combobox', { name: 'Rooms and conversations' }), {
      target: { value: 'P2_P3' },
    })
    expect(screen.queryByLabelText('Your message')).not.toBeInTheDocument()
    expect(
      screen.getByText(/Read only: you are viewing the conversation of Maison Brieuc/),
    ).toBeInTheDocument()
  })

  it('announces the split of a long text and counts its characters', async () => {
    setChat({ public: channel([]) })
    renderPanel({ visible: true })
    const text = `${'a'.repeat(400)}\n\n${'b'.repeat(400)}`
    fireEvent.change(screen.getByLabelText('Your message'), { target: { value: text } })
    expect(screen.getByText('This message will be sent in 2 parts.')).toBeInTheDocument()
    expect(screen.getByText('802 / 4000')).toBeInTheDocument()
  })

  it('refuses a text over 4000 characters', () => {
    setChat({ public: channel([]) })
    renderPanel()
    fireEvent.change(screen.getByLabelText('Your message'), {
      target: { value: 'x '.repeat(2001) },
    })
    expect(screen.getByText('The message is over 4000 characters.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
  })

  it('links the communes and nobles a message names to the map', () => {
    const withBody = {
      ...message('1', 'P2', 2),
      body: 'Hugon passera à Rochevent, près de Rochevental.',
    }
    setChat({ public: channel([withBody]) })
    const links: MapLinks = {
      focus: vi.fn(),
      follow: vi.fn(),
      showEvent: vi.fn(),
      highlight: vi.fn(),
      regionTerritories: (seed) => [seed],
      hasTerritory: (id) => id === 'AAA',
      territoryName: () => undefined,
      nobleName: () => undefined,
      nobleTerritory: (code) => (code === 'HUO' ? 'AAA' : undefined),
      isDeceased: () => false,
    }
    const map = { territories: [{ id: 'AAA', name: 'Rochevent' }] } as unknown as MapData
    const nobles = [{ code: 'HUO', name: 'Sieur Hugon de Rochevent', firstName: 'Hugon' }]
    render(
      <LanguageProvider initialLanguage="en">
        <MapLinksProvider value={links}>
          <ChatPanel
            gameId="g1"
            uid="u1"
            tokenProvider={{ getIdToken: async () => 'token' }}
            state={{ ...state, nobles } as unknown as StateData}
            map={map}
            house="P1"
            observer={false}
            visible
          />
        </MapLinksProvider>
      </LanguageProvider>,
    )
    const log = screen.getByRole('log', { name: /Public room/ })
    expect(log.querySelectorAll('[data-chat-mention]')).toHaveLength(2)
    fireEvent.click(within(log).getByRole('button', { name: 'Hugon' }))
    expect(links.focus).toHaveBeenCalledWith('AAA', 'HUO')
    fireEvent.click(within(log).getByRole('button', { name: 'Rochevent' }))
    expect(links.focus).toHaveBeenLastCalledWith('AAA', undefined)
    expect(within(log).getByText(/près de Rochevental\./)).toBeInTheDocument()
  })
})
