import type { PlayerId, Season, StateData } from '@/types'

/** Channel identifier of the public room (same as the backend). */
export const PUBLIC_CHANNEL = 'public'
/** Longest text a player may write in one go. */
export const CHAT_MAX_CHARS = 4000
/** Longest body of one stored message. */
export const CHAT_CHUNK_SIZE = 500
/** Longest batch (4 000 / 500). */
export const CHAT_MAX_PARTS = 8
/** Minimum delay between two messages of a player (backend rate limit). */
export const CHAT_MIN_INTERVAL_MS = 1000
/** Messages read at start and per "load more". */
export const CHAT_PAGE_SIZE = 50

export interface ChatTurn {
  number: number
  season: Season
  year: number
}

/** A stored message. `createdAt` is an RFC 3339 UTC string (see toChatTimestamp). */
export interface ChatMessage {
  id: string
  channel: string
  authorHouseId: PlayerId
  body: string
  createdAt: string
  turn: ChatTurn
  batchId?: string
  part?: number
  parts?: number
}

const SEASONS: readonly Season[] = ['spring', 'summer', 'autumn', 'winter']

// ---------------------------------------------------------------------------
// Channels

/** Key of the private conversation between two houses: sorted ids joined by `_`. */
export function conversationKey(a: PlayerId, b: PlayerId): string {
  return a > b ? `${b}_${a}` : `${a}_${b}`
}

/** The two houses of a conversation key. */
export function conversationHouses(key: string): [PlayerId, PlayerId] | null {
  const index = key.indexOf('_')
  if (index <= 0 || index === key.length - 1) return null
  return [key.slice(0, index), key.slice(index + 1)]
}

/** The other house of a conversation `house` takes part in. */
export function otherHouse(key: string, house: PlayerId): PlayerId | null {
  const houses = conversationHouses(key)
  if (!houses) return null
  if (houses[0] === house) return houses[1]
  if (houses[1] === house) return houses[0]
  return null
}

/** Whether `house` takes part in the channel (the public room, or its own conversation). */
export function isOwnChannel(key: string, house: PlayerId | null): boolean {
  return key === PUBLIC_CHANNEL || (house !== null && otherHouse(key, house) !== null)
}

/** The channel as the API names it: "public", or the other house of a conversation. */
export function apiChannel(key: string, house: PlayerId): string | null {
  return key === PUBLIC_CHANNEL ? PUBLIC_CHANNEL : otherHouse(key, house)
}

// ---------------------------------------------------------------------------
// Timestamps

const RFC3339 = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(\d+))?(Z|[+-]\d{2}:\d{2})$/

/**
 * Normalizes a timestamp to `YYYY-MM-DDTHH:MM:SS.nnnnnnnnnZ`: a fixed width,
 * so strings compare like instants, and nanoseconds, so cursors keep the
 * microseconds Firestore stores. Accepts RFC 3339 strings, Dates and
 * Firestore timestamps.
 */
export function toChatTimestamp(value: unknown): string | undefined {
  if (typeof value === 'string') {
    const match = RFC3339.exec(value)
    if (!match) return undefined
    if (match[3] === 'Z')
      return `${match[1]}.${(match[2] ?? '').padEnd(9, '0').slice(0, 9)}Z`
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? undefined : toChatTimestamp(date)
  }
  if (value instanceof Date) {
    if (Number.isNaN(value.getTime())) return undefined
    const iso = value.toISOString()
    return `${iso.slice(0, 19)}.${iso.slice(20, 23)}000000Z`
  }
  if (typeof value === 'object' && value !== null) {
    const timestamp = value as { seconds?: unknown; nanoseconds?: unknown }
    if (typeof timestamp.seconds === 'number') {
      const nanos = typeof timestamp.nanoseconds === 'number' ? timestamp.nanoseconds : 0
      const base = new Date(timestamp.seconds * 1000).toISOString().slice(0, 19)
      return `${base}.${String(nanos).padStart(9, '0')}Z`
    }
    const toDate = (value as { toDate?: () => Date }).toDate
    if (typeof toDate === 'function') return toChatTimestamp(toDate.call(value))
  }
  return undefined
}

/** Reads a stored message document; null when it is malformed. */
export function normalizeChatMessage(
  data: Record<string, unknown>,
  fallbackId = '',
): ChatMessage | null {
  const createdAt = toChatTimestamp(data.createdAt)
  const turn = data.turn as Partial<ChatTurn> | undefined
  const id = typeof data.id === 'string' && data.id ? data.id : fallbackId
  if (
    !createdAt ||
    !id ||
    typeof data.channel !== 'string' ||
    typeof data.authorHouseId !== 'string' ||
    typeof data.body !== 'string' ||
    !turn ||
    typeof turn.number !== 'number' ||
    typeof turn.year !== 'number' ||
    !SEASONS.includes(turn.season as Season)
  ) {
    return null
  }
  return {
    id,
    channel: data.channel,
    authorHouseId: data.authorHouseId,
    body: data.body,
    createdAt,
    turn: { number: turn.number, season: turn.season as Season, year: turn.year },
    ...(typeof data.batchId === 'string' && data.batchId
      ? {
          batchId: data.batchId,
          part: typeof data.part === 'number' ? data.part : undefined,
          parts: typeof data.parts === 'number' ? data.parts : undefined,
        }
      : {}),
  }
}

/** Orders messages by creation time then id, like the stored order. */
export function compareMessages(a: ChatMessage, b: ChatMessage): number {
  if (a.createdAt !== b.createdAt) return a.createdAt < b.createdAt ? -1 : 1
  return a.id < b.id ? -1 : a.id > b.id ? 1 : 0
}

/** Union of two message lists by id (`incoming` wins), oldest first. */
export function mergeMessages(
  current: readonly ChatMessage[],
  incoming: readonly ChatMessage[],
): ChatMessage[] {
  const byId = new Map<string, ChatMessage>()
  for (const message of current) byId.set(message.id, message)
  for (const message of incoming) byId.set(message.id, message)
  return [...byId.values()].sort(compareMessages)
}

// ---------------------------------------------------------------------------
// Splitting

export interface ChatChunk {
  text: string
  /** Offset of the chunk in the normalized (trimmed) text. */
  start: number
}

/** Length of a text in characters (code points), as the backend counts runes. */
export function chatLength(text: string): number {
  return Array.from(text.trim()).length
}

const isSpace = (char: string) => /\s/.test(char)
const isNewline = (char: string) => char === '\n'
const SENTENCE_END = new Set(['.', '!', '?', '…'])
const CLOSERS = new Set(['"', "'", '»', ')', ']', '”', '’'])

type BreakKind = 'paragraph' | 'sentence' | 'word'

/** Whitespace runs of `chars` inside (0, limit], with the kind of break they form. */
function breakCandidates(chars: string[], limit: number) {
  const found: Record<BreakKind, { end: number; next: number } | null> = {
    paragraph: null,
    sentence: null,
    word: null,
  }
  let index = 1
  while (index <= Math.min(limit, chars.length - 1)) {
    if (!isSpace(chars[index]) || isSpace(chars[index - 1])) {
      index++
      continue
    }
    // chars[index] opens a whitespace run that starts after a word.
    let next = index
    let newlines = 0
    while (next < chars.length && isSpace(chars[next])) {
      if (isNewline(chars[next])) newlines++
      next++
    }
    let before = index - 1
    while (before > 0 && CLOSERS.has(chars[before])) before--
    const kind: BreakKind =
      newlines >= 2 ? 'paragraph' : SENTENCE_END.has(chars[before]) ? 'sentence' : 'word'
    found[kind] = { end: index, next }
    index = next
  }
  return found
}

function splitWith(text: string, size: number, kinds: readonly BreakKind[]): ChatChunk[] {
  const chars = Array.from(text)
  const offsets: number[] = []
  let total = 0
  for (const char of chars) {
    offsets.push(total)
    total += char.length
  }
  const chunks: ChatChunk[] = []
  let position = 0
  while (position < chars.length) {
    const rest = chars.slice(position)
    if (rest.length <= size) {
      chunks.push({ text: rest.join(''), start: offsets[position] })
      break
    }
    const candidates = breakCandidates(rest, size)
    const choice = kinds.map((kind) => candidates[kind]).find((value) => value !== null)
    if (!choice) {
      // A word longer than a chunk: the only case where a word is cut.
      chunks.push({ text: rest.slice(0, size).join(''), start: offsets[position] })
      position += size
      continue
    }
    chunks.push({ text: rest.slice(0, choice.end).join(''), start: offsets[position] })
    position += choice.next
  }
  return chunks
}

/**
 * Splits a long text into chunks of at most `size` characters, cutting at a
 * paragraph break, else after a sentence, else between two words, and inside
 * a word only when that word alone exceeds `size`. When the best breaks give
 * more than `maxParts` chunks, the denser word breaks are used instead.
 */
export function splitChatMessage(
  text: string,
  size = CHAT_CHUNK_SIZE,
  maxParts = CHAT_MAX_PARTS,
): ChatChunk[] {
  const normalized = text.trim()
  if (normalized === '') return []
  const preferred = splitWith(normalized, size, ['paragraph', 'sentence', 'word'])
  if (preferred.length <= maxParts) return preferred
  const dense = splitWith(normalized, size, ['word'])
  return dense.length < preferred.length ? dense : preferred
}

// ---------------------------------------------------------------------------
// Timeline

export type TimelineItem =
  | { kind: 'separator'; key: string; turn: ChatTurn; current: boolean }
  | {
      kind: 'message'
      key: string
      authorHouseId: PlayerId
      /** One message, or the consecutive parts of one batch. */
      messages: ChatMessage[]
    }

/**
 * Lays messages out for display: a separator opens each turn that has
 * messages, consecutive parts of a batch by one author form one entry, and a
 * final separator names the current turn when the last message predates it.
 */
export function buildTimeline(
  messages: readonly ChatMessage[],
  currentTurn?: ChatTurn,
): TimelineItem[] {
  const sorted = [...messages].sort(compareMessages)
  const items: TimelineItem[] = []
  let previous: ChatMessage | undefined
  for (const message of sorted) {
    if (!previous || previous.turn.number !== message.turn.number) {
      items.push({
        kind: 'separator',
        key: `turn-${message.turn.number}-${message.id}`,
        turn: message.turn,
        current: false,
      })
    }
    const last = items[items.length - 1]
    if (
      previous &&
      last.kind === 'message' &&
      message.batchId &&
      previous.batchId === message.batchId &&
      previous.authorHouseId === message.authorHouseId &&
      previous.turn.number === message.turn.number
    ) {
      last.messages.push(message)
    } else {
      items.push({
        kind: 'message',
        key: message.id,
        authorHouseId: message.authorHouseId,
        messages: [message],
      })
    }
    previous = message
  }
  if (previous && currentTurn && previous.turn.number !== currentTurn.number) {
    items.push({
      kind: 'separator',
      key: `turn-${currentTurn.number}-current`,
      turn: currentTurn,
      current: true,
    })
  }
  return items
}

// ---------------------------------------------------------------------------
// Unread

/**
 * Messages of a channel written by others after `lastReadAt` (everything,
 * when the channel was never read).
 */
export function unreadCount(
  messages: readonly ChatMessage[],
  lastReadAt: string | undefined,
  ownHouse: PlayerId | null,
): number {
  return messages.filter(
    (message) =>
      message.authorHouseId !== ownHouse &&
      (!lastReadAt || message.createdAt > lastReadAt),
  ).length
}

export function totalUnread(counts: Record<string, number>): number {
  return Object.values(counts).reduce((sum, count) => sum + count, 0)
}

// ---------------------------------------------------------------------------
// Sending

export interface ChatPostBody {
  body: string
  batchId?: string
  part?: number
  parts?: number
}

export interface SendOutcome {
  /** Chunks acknowledged by the server. */
  sent: number
  total: number
  /** Text not sent, to put back in the input; empty when everything went. */
  remaining: string
  cancelled: boolean
  error?: unknown
}

export interface SendOptions {
  text: string
  post: (body: ChatPostBody) => Promise<unknown>
  signal?: AbortSignal
  onProgress?: (sent: number, total: number) => void
  /** Resolves after `ms`, or early when `signal` aborts. */
  wait?: (ms: number, signal?: AbortSignal) => Promise<void>
  minIntervalMs?: number
  newBatchId?: () => string
}

export function abortableSleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal?.aborted) {
      resolve()
      return
    }
    const done = () => {
      clearTimeout(timer)
      signal?.removeEventListener('abort', done)
      resolve()
    }
    const timer = setTimeout(done, ms)
    signal?.addEventListener('abort', done)
  })
}

function defaultBatchId(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

/**
 * Sends a text as one message, or as a batch of chunks sent one after the
 * other: each chunk waits for the acknowledgement of the previous one and at
 * least `minIntervalMs` after it. On failure or cancellation, nothing more is
 * sent and the unsent text is returned.
 */
export async function sendChatText(options: SendOptions): Promise<SendOutcome> {
  const {
    text,
    post,
    signal,
    onProgress,
    wait = abortableSleep,
    minIntervalMs = CHAT_MIN_INTERVAL_MS,
    newBatchId = defaultBatchId,
  } = options
  const normalized = text.trim()
  const chunks = splitChatMessage(normalized)
  const batchId = chunks.length > 1 ? newBatchId() : undefined
  const remainingFrom = (index: number) =>
    index < chunks.length ? normalized.slice(chunks[index].start) : ''

  for (let index = 0; index < chunks.length; index++) {
    if (signal?.aborted) {
      return {
        sent: index,
        total: chunks.length,
        remaining: remainingFrom(index),
        cancelled: true,
      }
    }
    try {
      await post({
        body: chunks[index].text,
        ...(batchId ? { batchId, part: index + 1, parts: chunks.length } : {}),
      })
    } catch (error) {
      return {
        sent: index,
        total: chunks.length,
        remaining: remainingFrom(index),
        cancelled: false,
        error,
      }
    }
    onProgress?.(index + 1, chunks.length)
    if (index < chunks.length - 1) await wait(minIntervalMs, signal)
  }
  return { sent: chunks.length, total: chunks.length, remaining: '', cancelled: false }
}

// ---------------------------------------------------------------------------
// Houses

/**
 * Whether a house still plays: it holds a territory, a fief, a capital or an
 * army. Only such houses may be offered a new conversation.
 */
export function isHouseInPlay(
  state: Pick<StateData, 'players' | 'territories' | 'fiefs'>,
  houseId: PlayerId,
): boolean {
  const player = state.players.find((candidate) => candidate.id === houseId)
  if (!player) return false
  return (
    Boolean(player.capitalTerritory) ||
    state.territories.some(
      (territory) => territory.owner === houseId || territory.army?.owner === houseId,
    ) ||
    (state.fiefs ?? []).some((fief) => fief.owner === houseId)
  )
}
