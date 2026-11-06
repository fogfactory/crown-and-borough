import { describe, expect, it, vi } from 'vitest'

import {
  buildTimeline,
  chatLength,
  conversationKey,
  mergeMessages,
  otherHouse,
  sendChatText,
  splitChatMessage,
  toChatTimestamp,
  totalUnread,
  unreadCount,
  type ChatMessage,
  type ChatPostBody,
} from '@/lib/chat'

function message(
  id: string,
  at: string,
  turn: number,
  extra: Partial<ChatMessage> = {},
): ChatMessage {
  return {
    id,
    channel: 'public',
    authorHouseId: 'P1',
    body: id,
    createdAt: toChatTimestamp(at) as string,
    turn: { number: turn, season: 'spring', year: 1011 + Math.floor((turn - 1) / 4) },
    ...extra,
  }
}

describe('splitChatMessage', () => {
  it('keeps a short text whole and drops nothing', () => {
    expect(splitChatMessage('  hello  ')).toEqual([{ text: 'hello', start: 0 }])
    expect(splitChatMessage('   ')).toEqual([])
  })

  it('never exceeds the chunk size', () => {
    const text = Array.from({ length: 400 }, (_, i) => `mot${i}`).join(' ')
    const chunks = splitChatMessage(text, 500, 100)
    expect(chunks.length).toBeGreaterThan(1)
    for (const chunk of chunks) expect(chatLength(chunk.text)).toBeLessThanOrEqual(500)
    expect(chunks.map((chunk) => chunk.text).join(' ')).toBe(text)
  })

  it('prefers a paragraph break to a sentence end', () => {
    const first = 'a'.repeat(100)
    const text = `${first}.  ${'b'.repeat(100)}.\n\n${'c'.repeat(100)}`
    const chunks = splitChatMessage(text, 250)
    expect(chunks.map((chunk) => chunk.text)).toEqual([
      `${first}.  ${'b'.repeat(100)}.`,
      'c'.repeat(100),
    ])
  })

  it('falls back to a sentence end, then to a word boundary', () => {
    const sentences = `${'a'.repeat(100)}. ${'b'.repeat(100)}. ${'c'.repeat(100)}`
    expect(splitChatMessage(sentences, 250).map((chunk) => chunk.text)).toEqual([
      `${'a'.repeat(100)}. ${'b'.repeat(100)}.`,
      'c'.repeat(100),
    ])
    const words = `${'a'.repeat(100)} ${'b'.repeat(100)} ${'c'.repeat(100)}`
    expect(splitChatMessage(words, 250).map((chunk) => chunk.text)).toEqual([
      `${'a'.repeat(100)} ${'b'.repeat(100)}`,
      'c'.repeat(100),
    ])
  })

  it('cuts exactly at the limit when a word boundary falls there', () => {
    const text = `${'a'.repeat(9)} ${'b'.repeat(9)}`
    expect(splitChatMessage(text, 9).map((chunk) => chunk.text)).toEqual([
      'a'.repeat(9),
      'b'.repeat(9),
    ])
  })

  it('cuts inside a word only when the word alone exceeds the size', () => {
    const chunks = splitChatMessage(`abc ${'x'.repeat(12)} def`, 5, 20)
    expect(chunks.map((chunk) => chunk.text)).toEqual([
      'abc',
      'xxxxx',
      'xxxxx',
      'xx',
      'def',
    ])
  })

  it('counts characters, not bytes or UTF-16 units', () => {
    const emoji = '😀'.repeat(300)
    const text = `${emoji} ${'é'.repeat(300)}`
    const chunks = splitChatMessage(text)
    expect(chunks.map((chunk) => chunk.text)).toEqual([emoji, 'é'.repeat(300)])
    expect(chatLength(text)).toBe(601)
    const cut = splitChatMessage('😀'.repeat(600), 500, 8)
    expect(cut.map((chunk) => Array.from(chunk.text).length)).toEqual([500, 100])
    for (const chunk of cut) expect(chunk.text).toBe(Array.from(chunk.text).join(''))
  })

  it('records where each chunk starts in the trimmed text', () => {
    const text = `${'a'.repeat(300)}\n\n${'b'.repeat(300)}`
    const chunks = splitChatMessage(text)
    expect(chunks[1].start).toBe(302)
    expect(text.slice(chunks[1].start)).toBe('b'.repeat(300))
  })

  it('falls back to denser breaks when paragraph breaks would need nine parts', () => {
    const paragraph = Array.from({ length: 88 }, () => 'word').join(' ')
    const text = Array.from({ length: 9 }, () => paragraph).join('\n\n')
    expect(chatLength(text)).toBeLessThanOrEqual(4000)
    const chunks = splitChatMessage(text)
    expect(chunks.length).toBeLessThanOrEqual(8)
    for (const chunk of chunks) expect(chatLength(chunk.text)).toBeLessThanOrEqual(500)
  })
})

describe('buildTimeline', () => {
  it('opens each turn with messages with a separator and skips empty turns', () => {
    const items = buildTimeline([
      message('b', '2026-01-01T10:00:02Z', 5, { authorHouseId: 'P2' }),
      message('a', '2026-01-01T10:00:00Z', 1),
      message('c', '2026-01-01T10:00:03Z', 5),
    ])
    expect(items.map((item) => item.kind)).toEqual([
      'separator',
      'message',
      'separator',
      'message',
      'message',
    ])
    const separators = items.flatMap((item) =>
      item.kind === 'separator' ? [item.turn.number] : [],
    )
    expect(separators).toEqual([1, 5])
  })

  it('appends a final separator for the current turn only when it has no message yet', () => {
    const turn = { number: 6, season: 'summer' as const, year: 1012 }
    const items = buildTimeline([message('a', '2026-01-01T10:00:00Z', 5)], turn)
    expect(items.at(-1)).toMatchObject({ kind: 'separator', current: true, turn })
    const inTurn = buildTimeline([message('a', '2026-01-01T10:00:00Z', 6)], turn)
    expect(inTurn.filter((item) => item.kind === 'separator')).toHaveLength(1)
    expect(buildTimeline([], turn)).toEqual([])
  })

  it('groups consecutive parts of one batch by one author', () => {
    const batch = { batchId: 'x', parts: 3 }
    const items = buildTimeline([
      message('1', '2026-01-01T10:00:00Z', 1, { ...batch, part: 1 }),
      message('2', '2026-01-01T10:00:01Z', 1, { ...batch, part: 2 }),
      message('3', '2026-01-01T10:00:02Z', 1, { ...batch, part: 3 }),
      message('4', '2026-01-01T10:00:03Z', 1),
    ])
    const groups = items.flatMap((item) =>
      item.kind === 'message' ? [item.messages.length] : [],
    )
    expect(groups).toEqual([3, 1])
  })

  it('does not group parts interleaved with another author or split by a turn', () => {
    const batch = { batchId: 'x', parts: 2 }
    const interleaved = buildTimeline([
      message('1', '2026-01-01T10:00:00Z', 1, { ...batch, part: 1 }),
      message('2', '2026-01-01T10:00:01Z', 1, { authorHouseId: 'P2' }),
      message('3', '2026-01-01T10:00:02Z', 1, { ...batch, part: 2 }),
    ])
    expect(interleaved.filter((item) => item.kind === 'message')).toHaveLength(3)
    const overTurns = buildTimeline([
      message('1', '2026-01-01T10:00:00Z', 1, { ...batch, part: 1 }),
      message('2', '2026-01-01T10:00:01Z', 2, { ...batch, part: 2 }),
    ])
    expect(overTurns.filter((item) => item.kind === 'message')).toHaveLength(2)
  })
})

describe('unread counts', () => {
  const messages = [
    message('a', '2026-01-01T10:00:00Z', 1, { authorHouseId: 'P2' }),
    message('b', '2026-01-01T10:00:05Z', 1, { authorHouseId: 'P1' }),
    message('c', '2026-01-01T10:00:10Z', 1, { authorHouseId: 'P2' }),
  ]

  it('counts only others’ messages after the read cursor', () => {
    expect(unreadCount(messages, toChatTimestamp('2026-01-01T10:00:00Z'), 'P1')).toBe(1)
    expect(unreadCount(messages, toChatTimestamp('2026-01-01T10:00:10Z'), 'P1')).toBe(0)
  })

  it('counts everything from others when the channel was never read', () => {
    expect(unreadCount(messages, undefined, 'P1')).toBe(2)
  })

  it('sums the channels', () => {
    expect(totalUnread({ public: 2, P1_P2: 3 })).toBe(5)
  })
})

describe('helpers', () => {
  it('builds conversation keys like the backend', () => {
    expect(conversationKey('P3', 'P1')).toBe('P1_P3')
    expect(otherHouse('P1_P3', 'P3')).toBe('P1')
    expect(otherHouse('P1_P3', 'P2')).toBeNull()
  })

  it('keeps microseconds and a fixed width in timestamps', () => {
    expect(toChatTimestamp('2026-01-01T10:00:00.5Z')).toBe(
      '2026-01-01T10:00:00.500000000Z',
    )
    expect(toChatTimestamp({ seconds: 1767261600, nanoseconds: 123456000 })).toBe(
      '2026-01-01T10:00:00.123456000Z',
    )
    expect(toChatTimestamp('nope')).toBeUndefined()
  })

  it('merges pages without duplicates, oldest first', () => {
    const a = message('a', '2026-01-01T10:00:00Z', 1)
    const b = message('b', '2026-01-01T10:00:01Z', 1)
    expect(mergeMessages([b], [a, b]).map((m) => m.id)).toEqual(['a', 'b'])
  })
})

describe('sendChatText', () => {
  const long = `${'a'.repeat(400)}\n\n${'b'.repeat(400)}\n\n${'c'.repeat(400)}`

  it('sends one message without batch fields', async () => {
    const post = vi.fn().mockResolvedValue(undefined)
    const outcome = await sendChatText({ text: 'bonjour', post })
    expect(post).toHaveBeenCalledWith({ body: 'bonjour' })
    expect(outcome).toMatchObject({ sent: 1, total: 1, remaining: '', cancelled: false })
  })

  it('sends the chunks in order, waiting for each acknowledgement and the delay', async () => {
    const events: string[] = []
    const post = vi.fn(async (body: ChatPostBody) => {
      events.push(`post ${body.part}`)
      await Promise.resolve()
      events.push(`ack ${body.part}`)
    })
    const wait = vi.fn(async (ms: number) => {
      events.push(`wait ${ms}`)
    })
    const progress: Array<[number, number]> = []
    const outcome = await sendChatText({
      text: long,
      post,
      wait,
      newBatchId: () => 'batch-1',
      onProgress: (done, total) => progress.push([done, total]),
    })
    expect(events).toEqual([
      'post 1',
      'ack 1',
      'wait 1000',
      'post 2',
      'ack 2',
      'wait 1000',
      'post 3',
      'ack 3',
    ])
    expect(
      post.mock.calls.map(([body]) => [body.batchId, body.part, body.parts]),
    ).toEqual([
      ['batch-1', 1, 3],
      ['batch-1', 2, 3],
      ['batch-1', 3, 3],
    ])
    expect(progress).toEqual([
      [1, 3],
      [2, 3],
      [3, 3],
    ])
    expect(outcome).toMatchObject({ sent: 3, remaining: '' })
  })

  it('stops at a failure and returns the unsent text', async () => {
    const failure = new Error('boom')
    const post = vi.fn().mockResolvedValueOnce(undefined).mockRejectedValueOnce(failure)
    const outcome = await sendChatText({ text: long, post, wait: async () => undefined })
    expect(post).toHaveBeenCalledTimes(2)
    expect(outcome.sent).toBe(1)
    expect(outcome.error).toBe(failure)
    expect(outcome.remaining).toBe(`${'b'.repeat(400)}\n\n${'c'.repeat(400)}`)
  })

  it('returns the rest to the input when cancelled between two chunks', async () => {
    const controller = new AbortController()
    const post = vi.fn(async () => {
      controller.abort()
    })
    const outcome = await sendChatText({
      text: long,
      post,
      signal: controller.signal,
      wait: async () => undefined,
    })
    expect(post).toHaveBeenCalledTimes(1)
    expect(outcome).toMatchObject({ sent: 1, total: 3, cancelled: true })
    expect(outcome.remaining).toBe(`${'b'.repeat(400)}\n\n${'c'.repeat(400)}`)
  })

  it('really waits at least the minimum interval between chunks', async () => {
    vi.useFakeTimers()
    try {
      const post = vi.fn().mockResolvedValue(undefined)
      const pending = sendChatText({ text: long, post })
      await vi.advanceTimersByTimeAsync(0)
      expect(post).toHaveBeenCalledTimes(1)
      await vi.advanceTimersByTimeAsync(999)
      expect(post).toHaveBeenCalledTimes(1)
      await vi.advanceTimersByTimeAsync(1)
      expect(post).toHaveBeenCalledTimes(2)
      await vi.advanceTimersByTimeAsync(1000)
      await expect(pending).resolves.toMatchObject({ sent: 3 })
    } finally {
      vi.useRealTimers()
    }
  })
})
