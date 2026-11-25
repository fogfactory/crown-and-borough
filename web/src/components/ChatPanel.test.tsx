import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
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
const acknowledgeChat = vi.hoisted(() => vi.fn().mockResolvedValue({}))
const postChatMessage = vi.hoisted(() => vi.fn().mockResolvedValue({}))
vi.mock('@/lib/chat-api', () => ({ markChatRead, postChatMessage, acknowledgeChat }))

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

const CHANNELS = 'Rooms and conversations'
const NEW_CONVERSATION = 'New conversation'

function trigger(name: string) {
  return screen.getByRole('combobox', { name: new RegExp(`^${name}`) })
}

function openSelect(name: string) {
  fireEvent.click(trigger(name))
}

function chooseOption(select: string, option: string | RegExp) {
  openSelect(select)
  fireEvent.click(screen.getByRole('option', { name: option }))
}

function shields(element: HTMLElement) {
  return element.querySelectorAll('[data-shield-icon]')
}

function setChat(
  channels: UseChatResult['channels'],
  lastRead: UseChatResult['lastRead'] = {},
  acknowledged: UseChatResult['acknowledged'] = {},
) {
  chat.result = {
    channels,
    lastRead,
    acknowledged,
    loadMore: vi.fn(),
  } satisfies UseChatResult
}

// The reader's conversations (P1 against P2, P3 and P4), loaded and empty.
const NO_CONVERSATION = {
  P1_P2: channel([]),
  P1_P3: channel([]),
  P1_P4: channel([]),
}

function channel(messages: ChatMessage[]) {
  return { messages, loaded: true, hasMore: false, loadingMore: false, error: null }
}

function renderPanel(props: Partial<Parameters<typeof ChatPanel>[0]> = {}) {
  const onUnreadChange = vi.fn()
  const tree = () => (
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
    </LanguageProvider>
  )
  const view = render(tree())
  // Renders again (a new element: React skips an identical one) after `setChat`
  // changed what the mocked chat hook returns.
  return { onUnreadChange, reload: () => view.rerender(tree()) }
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
    expect(trigger(NEW_CONVERSATION)).toHaveTextContent('Write to a house…')
    openSelect(NEW_CONVERSATION)
    const options = screen.getAllByRole('option')
    expect(options.map((option) => option.textContent)).toEqual(['Maison Brieuc'])
    expect(shields(options[0])).toHaveLength(1)
  })

  it('shows the shield and name of the active conversation in the trigger', () => {
    setChat(
      {
        public: channel([]),
        P1_P3: channel([message('1', 'P3', 1, 'P1_P3'), message('2', 'P1', 1, 'P1_P3')]),
      },
      { P1_P3: toChatTimestamp('2026-01-01T10:00:09Z') as string },
    )
    renderPanel()
    expect(trigger(CHANNELS)).toHaveTextContent('Public room')
    expect(shields(trigger(CHANNELS))).toHaveLength(0)
    chooseOption(CHANNELS, /^Maison Cérès/)
    expect(trigger(CHANNELS)).toHaveTextContent('Maison Cérès')
    expect(shields(trigger(CHANNELS))).toHaveLength(1)
  })

  it('names both lists and gives every entry its house shield', () => {
    setChat({
      public: channel([]),
      P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
    })
    renderPanel()
    openSelect(CHANNELS)
    const list = screen.getByRole('listbox', { name: CHANNELS })
    const [publicRoom, conversation] = within(list).getAllByRole('option')
    expect(shields(publicRoom)).toHaveLength(0)
    expect(shields(conversation)).toHaveLength(1)
  })

  it('names the trigger with the open conversation for screen readers', () => {
    setChat({ public: channel([]) })
    renderPanel()
    expect(
      screen.getByRole('combobox', { name: 'Rooms and conversations Public room' }),
    ).toBeInTheDocument()
  })

  it('returns focus to the channel trigger after the last new conversation is picked', async () => {
    setChat({
      public: channel([]),
      P1_P2: channel([]),
      P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
      P1_P4: channel([]),
    })
    renderPanel()
    chooseOption(NEW_CONVERSATION, 'Maison Brieuc')
    expect(screen.queryByRole('combobox', { name: NEW_CONVERSATION })).toBeNull()
    await waitFor(() => expect(trigger(CHANNELS)).toHaveFocus())
    expect(trigger(CHANNELS)).toHaveTextContent('Maison Brieuc')
  })

  it('keeps focus on the new conversation trigger when its list is closed without a pick', async () => {
    setChat({
      public: channel([]),
      P1_P2: channel([]),
      P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
      P1_P4: channel([]),
    })
    renderPanel()
    openSelect(NEW_CONVERSATION)
    fireEvent.keyDown(screen.getByRole('listbox', { name: NEW_CONVERSATION }), {
      key: 'Escape',
    })
    await waitFor(() => expect(trigger(NEW_CONVERSATION)).toHaveFocus())
  })

  it('opens the channel list with the keyboard', () => {
    setChat({ public: channel([]) })
    renderPanel()
    fireEvent.keyDown(trigger(CHANNELS), { key: 'ArrowDown' })
    expect(screen.getByRole('listbox', { name: CHANNELS })).toBeInTheDocument()
  })

  it('keeps an existing conversation with an eliminated house usable', () => {
    setChat({
      public: channel([]),
      P1_P4: channel([message('1', 'P4', 1, 'P1_P4')]),
    })
    renderPanel()
    chooseOption(CHANNELS, /^Maison Dorian/)
    expect(trigger(CHANNELS)).toHaveTextContent('Maison Dorian')
    expect(screen.getByRole('log', { name: /Maison Dorian/ })).toBeInTheDocument()
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
    openSelect(CHANNELS)
    expect(
      screen.getByRole('option', { name: /^Public room\s*1 unread messages/ }),
    ).toBeInTheDocument()
  })

  it('opens the only conversation with unread messages', () => {
    setChat(
      {
        public: channel([]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([]),
        P1_P4: channel([]),
      },
      {},
    )
    renderPanel({ visible: true })
    expect(trigger(CHANNELS)).toHaveTextContent('Maison Brieuc')
  })

  it('keeps the public room when it also has unread messages', () => {
    setChat(
      {
        public: channel([message('1', 'P2', 1)]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([]),
        P1_P4: channel([]),
      },
      {},
    )
    renderPanel({ visible: true })
    expect(trigger(CHANNELS)).toHaveTextContent('Public room')
  })

  it('keeps the public room when several conversations have unread messages', () => {
    setChat(
      {
        public: channel([]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([message('4', 'P3', 1, 'P1_P3')]),
        P1_P4: channel([]),
      },
      {},
    )
    renderPanel({ visible: true })
    expect(trigger(CHANNELS)).toHaveTextContent('Public room')
    expect(
      trigger(CHANNELS).querySelector('[data-chat-attention-elsewhere]'),
    ).not.toBeNull()
  })

  it('keeps the channel the reader picked while the first one was pending', () => {
    const read = { P1_P3: toChatTimestamp('2026-01-01T10:00:09Z') as string }
    setChat(
      {
        public: channel([]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([message('4', 'P3', 1, 'P1_P3')]),
      },
      read,
    )
    const { reload } = renderPanel({ visible: true })
    expect(trigger(CHANNELS)).toHaveTextContent('Public room')
    chooseOption(CHANNELS, /^Maison Cérès/)
    // P1_P4 loads at last: P2 is the only conversation with unread messages.
    setChat(
      {
        public: channel([]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([message('4', 'P3', 1, 'P1_P3')]),
        P1_P4: channel([]),
      },
      read,
    )
    reload()
    expect(trigger(CHANNELS)).toHaveTextContent('Maison Cérès')
  })

  it('waits for the public room before choosing the first channel', () => {
    setChat({ ...NO_CONVERSATION, P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]) }, {})
    const { reload } = renderPanel({ visible: true })
    expect(trigger(CHANNELS)).toHaveTextContent('Public room')
    setChat(
      {
        public: channel([message('1', 'P3', 1)]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([]),
        P1_P4: channel([]),
      },
      {},
    )
    reload()
    expect(trigger(CHANNELS)).toHaveTextContent('Public room')
  })

  it('marks nothing read before the first channel is chosen', () => {
    setChat({ public: channel([message('1', 'P3', 1)]), P1_P2: channel([]) }, {})
    const { reload } = renderPanel({ visible: true })
    expect(markChatRead).not.toHaveBeenCalled()
    setChat(
      {
        public: channel([message('1', 'P3', 1)]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([]),
        P1_P4: channel([]),
      },
      {},
    )
    reload()
    // The public room had unread messages too: it stays, and is read.
    expect(trigger(CHANNELS)).toHaveTextContent('Public room')
    expect(markChatRead).toHaveBeenCalledWith(expect.anything(), 'g1', 'public')
  })

  it('shows one pill per author house, in its color', () => {
    setChat(
      {
        public: channel([
          message('1', 'P2', 1),
          message('2', 'P3', 1),
          message('3', 'P3', 1),
        ]),
        ...NO_CONVERSATION,
      },
      {},
    )
    renderPanel()
    openSelect(CHANNELS)
    const option = screen.getByRole('option', { name: /^Public room/ })
    // One eye for the whole channel, whoever wrote.
    const pills = option.querySelectorAll<HTMLElement>('[data-chat-unread]')
    expect(pills).toHaveLength(1)
    expect(pills[0]).toHaveAttribute('aria-label', '3 unread messages')
  })

  it('signals on the trigger the unread messages of the other channels', () => {
    setChat(
      {
        public: channel([]),
        P1_P2: channel([message('3', 'P2', 1, 'P1_P2')]),
        P1_P3: channel([message('4', 'P3', 1, 'P1_P3')]),
        P1_P4: channel([]),
      },
      {},
    )
    renderPanel({ visible: true })
    expect(trigger(CHANNELS)).toHaveAccessibleName(
      /2 unread in other rooms: Maison Brieuc \(1\), Maison Cérès \(1\)/,
    )
    const chips = trigger(CHANNELS).querySelectorAll('[data-chat-unread-channel]')
    expect(
      [...chips].map((chip) => chip.getAttribute('data-chat-unread-channel')),
    ).toEqual(['P1_P2', 'P1_P3'])
    expect(trigger(CHANNELS).querySelectorAll('[data-chat-unread]')).toHaveLength(2)
  })

  it('marks the open channel read when the chat is on screen', () => {
    setChat({ ...NO_CONVERSATION, public: channel([message('1', 'P2', 1)]) }, {})
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
    chooseOption(CHANNELS, 'Maison Brieuc and Maison Cérès')
    expect(shields(trigger(CHANNELS))).toHaveLength(2)
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

  it('sends on Shift+Enter and keeps Enter for a new line', async () => {
    setChat({ public: channel([]) })
    postChatMessage.mockResolvedValue({})
    renderPanel({ visible: true })
    const field = screen.getByLabelText('Your message')
    fireEvent.change(field, { target: { value: 'hello' } })
    fireEvent.keyDown(field, { key: 'Enter' })
    expect(postChatMessage).not.toHaveBeenCalled()
    fireEvent.keyDown(field, { key: 'Enter', shiftKey: true })
    await waitFor(() => expect(postChatMessage).toHaveBeenCalledTimes(1))
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
    expect(log.querySelectorAll('[data-map-link]')).toHaveLength(2)
    fireEvent.click(within(log).getByRole('button', { name: 'Hugon' }))
    expect(links.focus).toHaveBeenCalledWith('AAA', 'HUO')
    fireEvent.click(within(log).getByRole('button', { name: 'Rochevent' }))
    expect(links.focus).toHaveBeenLastCalledWith('AAA', undefined)
    expect(within(log).getByText(/près de Rochevental\./)).toBeInTheDocument()
  })

  describe('unanswered channels', () => {
    const READ = { P1_P3: toChatTimestamp('2026-01-01T10:00:09Z') as string }

    it('opens the only unanswered channel when nothing is unread', async () => {
      setChat(
        {
          public: channel([]),
          P1_P2: channel([]),
          P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
          P1_P4: channel([]),
        },
        READ,
      )
      renderPanel({ visible: true })
      await waitFor(() => expect(trigger(CHANNELS)).toHaveTextContent('Maison Cérès'))
    })

    it('prefers unread channels over unanswered ones', async () => {
      setChat(
        {
          public: channel([]),
          P1_P2: channel([message('5', 'P2', 1, 'P1_P2', '2026-01-01T10:00:15Z')]),
          P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
          P1_P4: channel([]),
        },
        READ,
      )
      renderPanel({ visible: true })
      await waitFor(() => expect(trigger(CHANNELS)).toHaveTextContent('Maison Brieuc'))
    })

    it('shows the marker and acknowledges without sending a message', async () => {
      setChat(
        {
          ...NO_CONVERSATION,
          public: channel([]),
          P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
        },
        READ,
      )
      renderPanel({ visible: true })
      await waitFor(() => expect(trigger(CHANNELS)).toHaveTextContent('Maison Cérès'))
      fireEvent.click(screen.getByRole('button', { name: 'Acknowledged' }))
      await waitFor(() =>
        expect(acknowledgeChat).toHaveBeenCalledWith(expect.anything(), 'g1', 'P3'),
      )
      expect(postChatMessage).not.toHaveBeenCalled()
    })

    it('hides the button once acknowledged or answered', async () => {
      setChat(
        {
          ...NO_CONVERSATION,
          public: channel([]),
          P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
        },
        READ,
        { P1_P3: toChatTimestamp('2026-01-01T10:00:30Z') as string },
      )
      renderPanel({ visible: true })
      await waitFor(() => expect(trigger(CHANNELS)).toBeInTheDocument())
      expect(screen.queryByRole('button', { name: 'Acknowledged' })).toBeNull()
    })

    it('flags another unanswered channel on the trigger', () => {
      setChat(
        {
          ...NO_CONVERSATION,
          public: channel([]),
          P1_P3: channel([message('1', 'P3', 1, 'P1_P3')]),
        },
        READ,
      )
      renderPanel()
      expect(
        trigger(CHANNELS).querySelector('[data-chat-unanswered-channel="P1_P3"]'),
      ).not.toBeNull()
    })
  })
})
