import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from 'react'
import { IconLock, IconSend } from '@tabler/icons-react'

import { ShieldIcon } from '@/components/ArmsIcons'
import { MapMention } from '@/components/MapLinks'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { fieldClassName } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useLanguage } from '@/i18n/LanguageContext'
import { ApiError, type TokenProvider } from '@/lib/api'
import {
  CHAT_MAX_CHARS,
  CHAT_MAX_PARTS,
  PUBLIC_CHANNEL,
  buildTimeline,
  chatLength,
  conversationHouses,
  conversationKey,
  apiChannel,
  isHouseInPlay,
  isOwnChannel,
  otherHouse,
  sendChatText,
  splitChatMessage,
  totalUnread,
  unreadCount,
  type ChatMessage,
} from '@/lib/chat'
import {
  buildMentionIndex,
  tokenizeMentions,
  type MentionIndex,
} from '@/lib/chat-mentions'
import { markChatRead, postChatMessage } from '@/lib/chat-api'
import { playerArms } from '@/lib/heraldry'
import { playerDisplayName } from '@/lib/player-label'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { useChat, type ChatSubscriptionSpec } from '@/lib/use-chat'
import { cn } from '@/lib/utils'
import type { MapData, Player, PlayerId, StateData } from '@/types'

interface ChatPanelProps {
  gameId: string
  uid: string
  tokenProvider: TokenProvider
  state: StateData
  /** Communes named in messages link to the map. */
  map?: MapData | null
  /** The reader's house; null for the observer, who plays none. */
  house: PlayerId | null
  /** The observer plays no house and reads every conversation. */
  observer: boolean
  /** The chat is on screen: what is read there counts as read. */
  visible: boolean
  onUnreadChange?: (count: number) => void
}

/** Pill with an unread count (drawer bookmark, channel buttons). */
export function ChatUnreadBadge({ count }: { count: number }) {
  const { t } = useLanguage()
  if (count <= 0) return null
  return (
    <span
      data-chat-unread={count}
      role="img"
      aria-label={t('chat.unreadMany', { count })}
      className="inline-flex min-w-4 items-center justify-center rounded-full bg-vermilion-700 px-1 text-2xs font-semibold leading-4 text-paper-50"
    >
      {count > 99 ? '99+' : count}
    </span>
  )
}

function sendErrorKey(error: unknown) {
  if (error instanceof ApiError) {
    if (error.status === 429 || error.code === 'rate_limited')
      return 'chat.error.rateLimited'
    if (error.status === 0) return 'chat.error.network'
    if (error.code === 'invalid_recipient') return 'chat.error.recipient'
    return null
  }
  return 'chat.error.network'
}

function messageTime(message: ChatMessage, locale: string): string {
  const date = new Date(`${message.createdAt.slice(0, 23)}Z`)
  return Number.isNaN(date.getTime())
    ? ''
    : date.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })
}

/** A house's shield. Several can be mounted at once (list, trigger), so ids come from useId. */
function HouseMark({
  player,
  className,
}: {
  player: Player | undefined
  className?: string
}) {
  const id = useId()
  if (!player) return null
  return <ShieldIcon id={id} arms={playerArms(player)} className={className} />
}

// Field height (40px) on desktop, 44px touch target on phones (charte graphique).
const SELECT_TRIGGER_CLASS = 'w-full data-[size=default]:h-11 lg:data-[size=default]:h-10'
// Rows keep a 44px touch target on phones; the label may shrink and truncate.
const SELECT_ITEM_CLASS = 'max-lg:min-h-11'

/** A selector entry: the arms of the house(s) concerned, the label, then the unread suffix. */
function ChannelOption({
  houses,
  label,
  suffix,
  playerOf,
}: {
  houses: string[]
  label: string
  suffix?: string
  playerOf: (id: string) => Player | undefined
}) {
  return (
    <span className="flex min-w-0 items-center gap-1.5">
      {houses.map((houseId) => (
        <HouseMark key={houseId} player={playerOf(houseId)} className="size-5 shrink-0" />
      ))}
      <span className="truncate">{label}</span>
      {suffix && <span className="shrink-0">{suffix}</span>}
    </span>
  )
}

export function ChatPanel({
  gameId,
  uid,
  tokenProvider,
  state,
  map,
  house,
  observer,
  visible,
  onUnreadChange,
}: ChatPanelProps) {
  const { language, t } = useLanguage()
  const [open, setOpen] = useState<string>(PUBLIC_CHANNEL)
  const [opened, setOpened] = useState<ReadonlySet<string>>(
    () => new Set([PUBLIC_CHANNEL]),
  )
  const [drafts, setDrafts] = useState<Record<string, string>>({})
  const [sending, setSending] = useState<{
    channel: string
    done: number
    total: number
  } | null>(null)
  const [sendError, setSendError] = useState<{ channel: string; text: string } | null>(
    null,
  )
  const abort = useRef<AbortController | null>(null)
  const marked = useRef(new Map<string, string>())

  const houseName = useCallback(
    (id: string) => playerDisplayName(id, [state.players], id),
    [state.players],
  )
  const playersById = useMemo(
    () => new Map(state.players.map((player) => [player.id, player])),
    [state.players],
  )
  const playerOf = useCallback((id: string) => playersById.get(id), [playersById])
  // Names a message may cite: built once per map and state, read at render.
  const mentions = useMemo(
    () =>
      buildMentionIndex({
        territories: map?.territories ?? [],
        nobles: state.nobles ?? [],
      }),
    [map, state.nobles],
  )

  // Every conversation the reader may see: their own, and for the observer all pairs.
  const privateKeys = useMemo(() => {
    const ids = state.players.map((player) => player.id)
    const keys: string[] = []
    if (observer) {
      for (let i = 0; i < ids.length; i++) {
        for (let j = i + 1; j < ids.length; j++)
          keys.push(conversationKey(ids[i], ids[j]))
      }
    } else if (house) {
      for (const id of ids) if (id !== house) keys.push(conversationKey(house, id))
    }
    return keys
  }, [observer, house, state.players])

  const specs = useMemo<ChatSubscriptionSpec[]>(
    () => [
      { channel: PUBLIC_CHANNEL, own: false, peek: false },
      ...privateKeys.map((key) => {
        const own = house !== null && otherHouse(key, house) !== null
        return { channel: key, own, peek: !own && !opened.has(key) }
      }),
    ],
    [privateKeys, house, opened],
  )
  const { channels, lastRead, loadMore } = useChat({ gameId, uid, specs })

  const unread = useMemo(() => {
    const counts: Record<string, number> = {}
    if (!house || !lastRead) return counts
    for (const spec of specs) {
      if (!spec.own && spec.channel !== PUBLIC_CHANNEL) continue
      const messages = channels[spec.channel]?.messages ?? []
      counts[spec.channel] =
        visible && open === spec.channel
          ? 0
          : unreadCount(messages, lastRead[spec.channel], house)
    }
    return counts
  }, [channels, house, lastRead, open, specs, visible])
  const unreadTotal = totalUnread(unread)
  useEffect(() => {
    onUnreadChange?.(unreadTotal)
  }, [onUnreadChange, unreadTotal])

  // Reading the open channel moves its cursor.
  const openState = channels[open]
  const latestFromOthers = openState?.messages
    .filter((message) => message.authorHouseId !== house)
    .at(-1)?.createdAt
  useEffect(() => {
    if (!visible || !house || !lastRead || !openState?.loaded || !latestFromOthers) return
    if ((lastRead[open] ?? '') >= latestFromOthers) return
    if ((marked.current.get(open) ?? '') >= latestFromOthers) return
    marked.current.set(open, latestFromOthers)
    const channel = apiChannel(open, house)
    if (!channel) return
    markChatRead(tokenProvider, gameId, channel).catch(() => marked.current.delete(open))
  }, [
    gameId,
    house,
    lastRead,
    latestFromOthers,
    open,
    openState?.loaded,
    tokenProvider,
    visible,
  ])

  /** Houses a channel concerns for this reader: the other one, or both for the observer. */
  const channelHouses = useCallback(
    (key: string): string[] => {
      const houses = conversationHouses(key)
      if (!houses) return []
      const other = house ? otherHouse(key, house) : null
      return other ? [other] : houses
    },
    [house],
  )

  const channelLabel = useCallback(
    (key: string) => {
      if (key === PUBLIC_CHANNEL) return t('chat.public')
      const [first, second] = channelHouses(key)
      if (!first) return key
      return second
        ? t('chat.conversationBetween', {
            first: houseName(first),
            second: houseName(second),
          })
        : houseName(first)
    },
    [channelHouses, houseName, t],
  )

  const privateChannels = useMemo(() => {
    const latest = (key: string) => channels[key]?.messages.at(-1)?.createdAt ?? ''
    return privateKeys
      .filter((key) => (channels[key]?.messages.length ?? 0) > 0 || key === open)
      .sort((a, b) => latest(b).localeCompare(latest(a)))
  }, [channels, open, privateKeys])
  const newConversationTargets = useMemo(
    () =>
      house
        ? state.players.filter((player) => {
            const key = conversationKey(house, player.id)
            const channel = channels[key]
            return (
              player.id !== house &&
              isHouseInPlay(state, player.id) &&
              Boolean(channel?.loaded || channel?.error) &&
              (channel?.messages.length ?? 0) === 0 &&
              key !== open
            )
          })
        : [],
    [channels, house, open, state],
  )

  const select = (channel: string) => {
    setOpen(channel)
    setOpened((current) => new Set(current).add(channel))
  }

  // The closed Select still renders its items: keep their elements stable so that
  // typing in the composer does not rebuild every shield.
  const channelItems = useMemo(
    () =>
      [PUBLIC_CHANNEL, ...privateChannels].map((key) => {
        const count = unread[key] ?? 0
        return (
          <SelectItem key={key} value={key} className={SELECT_ITEM_CLASS}>
            <ChannelOption
              houses={channelHouses(key)}
              label={channelLabel(key)}
              suffix={count > 0 ? ` (${count > 99 ? '99+' : count})` : undefined}
              playerOf={playerOf}
            />
          </SelectItem>
        )
      }),
    [channelHouses, channelLabel, playerOf, privateChannels, unread],
  )
  const newConversationItems = useMemo(
    () =>
      newConversationTargets.map((player) => (
        <SelectItem key={player.id} value={player.id} className={SELECT_ITEM_CLASS}>
          <ChannelOption
            houses={[player.id]}
            label={houseName(player.id)}
            playerOf={playerOf}
          />
        </SelectItem>
      )),
    [houseName, newConversationTargets, playerOf],
  )

  // Picking a house opens its conversation, and the new-conversation list may
  // vanish with its last target: focus then moves to the channel trigger.
  // Closing the list without a pick keeps Radix's default focus restore.
  const channelTrigger = useRef<HTMLButtonElement>(null)
  const pickedNewConversation = useRef(false)

  const canWrite = house !== null && isOwnChannel(open, house)
  const draft = drafts[open] ?? ''
  const length = chatLength(draft)
  const parts = useMemo(() => splitChatMessage(draft).length, [draft])
  const tooLong = length > CHAT_MAX_CHARS
  const tooManyParts = !tooLong && parts > CHAT_MAX_PARTS
  const canSend = canWrite && !sending && length > 0 && !tooLong && !tooManyParts

  const send = async () => {
    if (!canSend || !house) return
    const channel = open
    const recipient = apiChannel(channel, house)
    if (!recipient) return
    const controller = new AbortController()
    abort.current = controller
    setSendError(null)
    setDrafts((current) => ({ ...current, [channel]: '' }))
    setSending({ channel, done: 0, total: parts })
    const outcome = await sendChatText({
      text: draft,
      signal: controller.signal,
      post: (body) => postChatMessage(tokenProvider, gameId, recipient, body),
      onProgress: (done, total) => setSending({ channel, done, total }),
    })
    abort.current = null
    setSending(null)
    if (outcome.remaining) {
      setDrafts((current) => ({
        ...current,
        [channel]: current[channel]
          ? `${outcome.remaining}\n\n${current[channel]}`
          : outcome.remaining,
      }))
    }
    if (outcome.error) {
      const key = sendErrorKey(outcome.error)
      setSendError({
        channel,
        text: key
          ? t(key)
          : t('chat.error.generic', {
              message: outcome.error instanceof Error ? outcome.error.message : '',
            }),
      })
    }
  }

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault()
      void send()
    }
  }

  const observerPair = observer ? conversationHouses(open) : null
  return (
    <section
      aria-label={t('chat.panel')}
      data-chat-panel
      className="flex min-h-0 flex-col gap-3"
    >
      <nav aria-label={t('chat.channels')} className="space-y-2">
        <Select value={open} onValueChange={select}>
          <SelectTrigger
            ref={channelTrigger}
            aria-label={`${t('chat.channels')} ${channelLabel(open)}`}
            className={SELECT_TRIGGER_CLASS}
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent position="popper" aria-label={t('chat.channels')}>
            {channelItems}
          </SelectContent>
        </Select>
        {house &&
          (newConversationTargets.length > 0 ? (
            <Select
              value=""
              onValueChange={(value) => {
                if (!value) return
                pickedNewConversation.current = true
                select(conversationKey(house, value))
              }}
            >
              <SelectTrigger
                aria-label={t('chat.newConversation')}
                className={SELECT_TRIGGER_CLASS}
              >
                <SelectValue placeholder={t('chat.newConversationPlaceholder')} />
              </SelectTrigger>
              <SelectContent
                position="popper"
                aria-label={t('chat.newConversation')}
                onCloseAutoFocus={(event) => {
                  if (!pickedNewConversation.current) return
                  pickedNewConversation.current = false
                  event.preventDefault()
                  channelTrigger.current?.focus()
                }}
              >
                {newConversationItems}
              </SelectContent>
            </Select>
          ) : (
            privateChannels.length === 0 && (
              <p className="text-xs text-ink-500">{t('chat.noHouses')}</p>
            )
          ))}
      </nav>

      <MessageList
        key={open}
        channel={open}
        label={t('chat.messages', { channel: channelLabel(open) })}
        messages={openState?.messages ?? []}
        loaded={Boolean(openState?.loaded)}
        hasMore={Boolean(openState?.hasMore)}
        loadingMore={Boolean(openState?.loadingMore)}
        error={openState?.error ?? null}
        state={state}
        house={house}
        visible={visible}
        language={language}
        houseName={houseName}
        playerOf={playerOf}
        mentions={mentions}
        emptyText={
          open === PUBLIC_CHANNEL
            ? t('chat.emptyPublic')
            : t('chat.emptyPrivate', { house: channelLabel(open) })
        }
        onLoadMore={() => void loadMore(open)}
      />

      {sendError?.channel === open && (
        <Alert role="alert" className="flex items-start justify-between gap-2">
          <span>{sendError.text}</span>
          <button
            type="button"
            className="shrink-0 underline"
            onClick={() => setSendError(null)}
          >
            {t('chat.error.dismiss')}
          </button>
        </Alert>
      )}

      {canWrite ? (
        <form
          className="space-y-1.5"
          onSubmit={(event) => {
            event.preventDefault()
            void send()
          }}
        >
          <label className="sr-only" htmlFor="chat-compose">
            {t('chat.compose')}
          </label>
          <textarea
            id="chat-compose"
            rows={3}
            value={draft}
            readOnly={sending?.channel === open}
            placeholder={t('chat.placeholder')}
            aria-describedby="chat-compose-hint"
            aria-invalid={tooLong || tooManyParts}
            className={cn(fieldClassName, 'h-auto min-h-20 w-full resize-y py-2')}
            onChange={(event) =>
              setDrafts((current) => ({ ...current, [open]: event.target.value }))
            }
            onKeyDown={onKeyDown}
          />
          <div
            id="chat-compose-hint"
            className="flex items-center justify-between gap-2 text-xs text-ink-500"
          >
            <span>{t('chat.hint')}</span>
            <span className={cn(tooLong && 'font-semibold text-vermilion-700')}>
              {t('chat.counter', { count: length, max: CHAT_MAX_CHARS })}
            </span>
          </div>
          {tooLong && (
            <p role="alert" className="text-xs text-vermilion-700">
              {t('chat.tooLong', { max: CHAT_MAX_CHARS })}
            </p>
          )}
          {tooManyParts && (
            <p role="alert" className="text-xs text-vermilion-700">
              {t('chat.tooManyParts', { max: CHAT_MAX_PARTS })}
            </p>
          )}
          {!sending && parts > 1 && !tooManyParts && !tooLong && (
            <p className="text-xs text-ink-500">{t('chat.splitNotice', { parts })}</p>
          )}
          <div className="flex items-center justify-end gap-2">
            {sending?.channel === open && (
              <>
                <span role="status" className="text-xs text-ink-700">
                  {t('chat.sendingProgress', {
                    done: sending.done,
                    total: sending.total,
                  })}
                </span>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => abort.current?.abort()}
                >
                  {t('chat.cancel')}
                </Button>
              </>
            )}
            <Button type="submit" disabled={!canSend}>
              <IconSend aria-hidden="true" />
              {t('chat.send')}
            </Button>
          </div>
        </form>
      ) : (
        <Alert tone="info" className="flex items-center gap-2" data-chat-readonly>
          <IconLock aria-hidden="true" className="size-4 shrink-0" />
          <span>
            {observerPair
              ? t('chat.readOnlyHost', {
                  first: houseName(observerPair[0]),
                  second: houseName(observerPair[1]),
                })
              : t('chat.readOnlyObserver')}
          </span>
        </Alert>
      )}
    </section>
  )
}

/** A message's text, with the communes and nobles it names linked to the map. */
function MessageBody({ body, mentions }: { body: string; mentions: MentionIndex }) {
  const segments = useMemo(() => tokenizeMentions(body, mentions), [body, mentions])
  return segments.map((segment, index) =>
    segment.kind === 'text' ? (
      segment.text
    ) : (
      <MapMention
        key={index}
        territory={segment.target.territory}
        nobles={segment.target.nobles}
      >
        {segment.text}
      </MapMention>
    ),
  )
}

interface MessageListProps {
  channel: string
  label: string
  messages: ChatMessage[]
  loaded: boolean
  hasMore: boolean
  loadingMore: boolean
  error: string | null
  state: StateData
  house: PlayerId | null
  visible: boolean
  language: string
  houseName: (id: string) => string
  playerOf: (id: string) => Player | undefined
  mentions: MentionIndex
  emptyText: string
  onLoadMore: () => void
}

function MessageList({
  channel,
  label,
  messages,
  loaded,
  hasMore,
  loadingMore,
  error,
  state,
  house,
  visible,
  language,
  houseName,
  playerOf,
  mentions,
  emptyText,
  onLoadMore,
}: MessageListProps) {
  const { t } = useLanguage()
  const scroller = useRef<HTMLDivElement>(null)
  const stickToBottom = useRef(true)
  const anchor = useRef<number | null>(null)
  const timeline = useMemo(
    () =>
      buildTimeline(messages, {
        number: state.turn,
        season: state.season,
        year: state.year ?? Math.floor((state.turn - 1) / 4) + 1,
      }),
    [messages, state.season, state.turn, state.year],
  )
  const oldestId = messages[0]?.id
  const newestId = messages.at(-1)?.id

  // Older messages arriving keep the reader where they were.
  useLayoutEffect(() => {
    const element = scroller.current
    if (element && anchor.current !== null) {
      element.scrollTop += element.scrollHeight - anchor.current
      anchor.current = null
    }
  }, [oldestId])
  // New messages (and showing the panel) scroll to the end unless the reader went up.
  useLayoutEffect(() => {
    const element = scroller.current
    if (element && stickToBottom.current && anchor.current === null) {
      element.scrollTop = element.scrollHeight
    }
  }, [newestId, visible, loaded])

  const seasonLabel = (season: ChatMessage['turn']['season']) =>
    t(SEASON_LABEL_KEYS[season])

  return (
    <div
      ref={scroller}
      role="log"
      aria-live="polite"
      aria-label={label}
      tabIndex={0}
      data-chat-log={channel}
      className="h-[min(26rem,50dvh)] min-h-48 space-y-2 overflow-y-auto rounded-md border border-line bg-paper-100 p-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      onScroll={(event) => {
        const element = event.currentTarget
        stickToBottom.current =
          element.scrollHeight - element.scrollTop - element.clientHeight < 48
      }}
    >
      {hasMore && (
        <div className="text-center">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={loadingMore}
            onClick={() => {
              anchor.current = scroller.current?.scrollHeight ?? null
              onLoadMore()
            }}
          >
            {loadingMore ? t('chat.loadingMore') : t('chat.loadMore')}
          </Button>
        </div>
      )}
      {error && (
        <Alert role="alert">
          {messages.length > 0
            ? t('chat.error.loadMore')
            : t('chat.error.read', { code: error })}
        </Alert>
      )}
      {!loaded && !error && (
        <p className="p-2 text-sm text-ink-500">{t('chat.loading')}</p>
      )}
      {loaded && messages.length === 0 && !error && (
        <p data-chat-empty className="p-3 text-center text-sm text-ink-500">
          {emptyText}
        </p>
      )}
      {timeline.map((item) => {
        if (item.kind === 'separator') {
          const values = { season: seasonLabel(item.turn.season), year: item.turn.year }
          return (
            <div
              key={item.key}
              role="separator"
              data-chat-separator={item.turn.number}
              className="flex items-center gap-2 py-1 text-2xs font-semibold uppercase tracking-wide text-ink-500"
            >
              <span aria-hidden="true" className="h-px flex-1 bg-line" />
              <span>
                {t(item.current ? 'chat.currentTurn' : 'chat.turnSeparator', values)}
              </span>
              <span aria-hidden="true" className="h-px flex-1 bg-line" />
            </div>
          )
        }
        const own = item.authorHouseId === house
        return (
          <article
            key={item.key}
            data-chat-message={item.authorHouseId}
            className={cn(
              'rounded-md border bg-paper-50 px-2.5 py-1.5',
              own ? 'border-brass-700/60' : 'border-line',
            )}
          >
            <header className="flex items-center gap-1.5 text-xs">
              <HouseMark player={playerOf(item.authorHouseId)} />
              <span className="font-semibold text-ink-900">
                {houseName(item.authorHouseId)}
                {own && (
                  <span className="font-normal text-ink-500"> ({t('chat.you')})</span>
                )}
              </span>
              <time
                className="ml-auto text-ink-500"
                dateTime={item.messages[0].createdAt}
              >
                {messageTime(item.messages[0], language)}
              </time>
            </header>
            {item.messages.map((message) => (
              <p
                key={message.id}
                className="whitespace-pre-wrap break-words text-sm text-ink-900"
                title={
                  message.parts && message.parts > 1
                    ? t('chat.part', { part: message.part ?? 1, parts: message.parts })
                    : undefined
                }
              >
                <MessageBody body={message.body} mentions={mentions} />
              </p>
            ))}
          </article>
        )
      })}
    </div>
  )
}
