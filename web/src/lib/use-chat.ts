import {
  collection,
  doc,
  getDocs,
  limit as limitTo,
  onSnapshot,
  orderBy,
  query,
  startAfter,
  where,
  type CollectionReference,
  type DocumentData,
  type FirestoreError,
  type Query,
  type QueryDocumentSnapshot,
} from 'firebase/firestore'
import { useCallback, useEffect, useRef, useState } from 'react'

import { getFirebaseServices } from '@/lib/firebase'
import {
  CHAT_PAGE_SIZE,
  PUBLIC_CHANNEL,
  mergeMessages,
  normalizeChatMessage,
  toChatTimestamp,
  type ChatMessage,
} from '@/lib/chat'

export interface ChatChannelState {
  messages: ChatMessage[]
  loaded: boolean
  /** Older messages may exist beyond the ones held. */
  hasMore: boolean
  loadingMore: boolean
  /** Firestore error code of the last failed read. */
  error: string | null
}

export interface ChatSubscriptionSpec {
  channel: string
  /** The conversation includes the reader: queried with their uid. */
  own: boolean
  /** Only the latest message is wanted (existence and recency of a conversation). */
  peek: boolean
}

interface UseChatOptions {
  gameId: string | undefined
  uid: string | undefined
  /** Channels to listen to; the hook never subscribes beyond this list. */
  specs: readonly ChatSubscriptionSpec[]
}

export interface UseChatResult {
  channels: Record<string, ChatChannelState>
  /** `lastReadAt` per channel; undefined until the cursors are loaded. */
  lastRead: Record<string, string> | undefined
  loadMore: (channel: string) => Promise<void>
}

const EMPTY_CHANNEL: ChatChannelState = {
  messages: [],
  loaded: false,
  hasMore: false,
  loadingMore: false,
  error: null,
}

function specId(spec: ChatSubscriptionSpec): string {
  return `${spec.channel}|${spec.own ? 'own' : 'view'}|${spec.peek ? 'peek' : 'full'}`
}

function messagesQuery(
  gameId: string,
  uid: string,
  spec: ChatSubscriptionSpec,
): Query<DocumentData> | null {
  const services = getFirebaseServices()
  if (!services) return null
  const base: CollectionReference<DocumentData> =
    spec.channel === PUBLIC_CHANNEL
      ? collection(services.firestore, 'games', gameId, 'chat_public')
      : collection(
          services.firestore,
          'games',
          gameId,
          'chat_private',
          spec.channel,
          'messages',
        )
  // The rules authorize a private message to its two participants and to the
  // observer: the participant query is the one every reader may run.
  return spec.channel !== PUBLIC_CHANNEL && spec.own
    ? query(
        base,
        where('participantUids', 'array-contains', uid),
        orderBy('createdAt', 'desc'),
      )
    : query(base, orderBy('createdAt', 'desc'))
}

function readMessages(
  docs: readonly QueryDocumentSnapshot<DocumentData>[],
): ChatMessage[] {
  return docs.flatMap((snapshot) => {
    const message = normalizeChatMessage(snapshot.data(), snapshot.id)
    return message ? [message] : []
  })
}

/**
 * Live chat messages (`onSnapshot` on the readable projections, newest page
 * only) and the reader's read cursors. Older messages come through
 * `loadMore`, by cursor after the oldest message held.
 */
export function useChat({ gameId, uid, specs }: UseChatOptions): UseChatResult {
  const [channels, setChannels] = useState<Record<string, ChatChannelState>>({})
  const [lastRead, setLastRead] = useState<Record<string, string> | undefined>(undefined)
  const unsubscribers = useRef(new Map<string, () => void>())
  const cursors = useRef(new Map<string, QueryDocumentSnapshot<DocumentData>>())
  const paged = useRef(new Set<string>())
  const specsRef = useRef(specs)
  useEffect(() => {
    specsRef.current = specs
  })

  const update = useCallback(
    (channel: string, change: (current: ChatChannelState) => ChatChannelState) => {
      setChannels((current) => ({
        ...current,
        [channel]: change(current[channel] ?? EMPTY_CHANNEL),
      }))
    },
    [],
  )

  // Read cursors and teardown, per game and account.
  useEffect(() => {
    const services = getFirebaseServices()
    const subscriptions = unsubscribers.current
    const cursorDocs = cursors.current
    const pagedChannels = paged.current
    let unsubscribeReads: () => void = () => undefined
    if (services && gameId && uid) {
      unsubscribeReads = onSnapshot(
        doc(services.firestore, 'games', gameId, 'chat_reads', uid),
        (snapshot) => {
          const channelsData = (snapshot.data()?.channels ?? {}) as Record<
            string,
            unknown
          >
          const cursorsRead: Record<string, string> = {}
          for (const [channel, value] of Object.entries(channelsData)) {
            const at = toChatTimestamp(value)
            if (at) cursorsRead[channel] = at
          }
          setLastRead(cursorsRead)
        },
        // Without cursors nothing is counted unread, rather than everything.
        () => undefined,
      )
    }
    return () => {
      unsubscribeReads()
      for (const unsubscribe of subscriptions.values()) unsubscribe()
      subscriptions.clear()
      cursorDocs.clear()
      pagedChannels.clear()
      setChannels({})
      setLastRead(undefined)
    }
  }, [gameId, uid])

  // Reconcile the listeners with the wanted specs.
  const specKey = specs.map(specId).join(',')
  useEffect(() => {
    if (!gameId || !uid) return
    const wanted = new Map(specsRef.current.map((spec) => [specId(spec), spec]))
    const subscriptions = unsubscribers.current
    for (const [id, unsubscribe] of subscriptions) {
      if (!wanted.has(id)) {
        unsubscribe()
        subscriptions.delete(id)
      }
    }
    for (const [id, spec] of wanted) {
      if (subscriptions.has(id)) continue
      const base = messagesQuery(gameId, uid, spec)
      if (!base) continue
      const size = spec.peek ? 1 : CHAT_PAGE_SIZE
      subscriptions.set(
        id,
        onSnapshot(
          query(base, limitTo(size)),
          (snapshot) => {
            if (!paged.current.has(spec.channel) && snapshot.docs.length > 0) {
              cursors.current.set(spec.channel, snapshot.docs[snapshot.docs.length - 1])
            }
            const live = readMessages(snapshot.docs)
            update(spec.channel, (current) => ({
              ...current,
              messages: mergeMessages(current.messages, live),
              loaded: true,
              hasMore: paged.current.has(spec.channel)
                ? current.hasMore
                : snapshot.docs.length >= size,
              error: null,
            }))
          },
          (error) =>
            update(spec.channel, (current) => ({
              ...current,
              loaded: true,
              error: error.code || 'unknown',
            })),
        ),
      )
    }
  }, [gameId, uid, specKey, update])

  const loadMore = useCallback(
    async (channel: string) => {
      const spec = specsRef.current.find((candidate) => candidate.channel === channel)
      const cursor = cursors.current.get(channel)
      if (!gameId || !uid || !spec || !cursor) return
      const base = messagesQuery(gameId, uid, spec)
      if (!base) return
      update(channel, (current) => ({ ...current, loadingMore: true, error: null }))
      try {
        const page = await getDocs(
          query(base, startAfter(cursor), limitTo(CHAT_PAGE_SIZE)),
        )
        paged.current.add(channel)
        if (page.docs.length > 0) {
          cursors.current.set(channel, page.docs[page.docs.length - 1])
        }
        const older = readMessages(page.docs)
        update(channel, (current) => ({
          ...current,
          messages: mergeMessages(current.messages, older),
          hasMore: page.docs.length >= CHAT_PAGE_SIZE,
          loadingMore: false,
        }))
      } catch (error) {
        update(channel, (current) => ({
          ...current,
          loadingMore: false,
          error: (error as Partial<FirestoreError>).code || 'unknown',
        }))
      }
    },
    [gameId, uid, update],
  )

  return { channels, lastRead, loadMore }
}
