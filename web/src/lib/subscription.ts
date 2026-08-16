import {
  collection,
  doc,
  onSnapshot,
  orderBy,
  query,
  where,
  type DocumentData,
  type FirestoreError,
} from 'firebase/firestore'
import { useEffect, useRef, useState } from 'react'

import { getFirebaseServices } from '@/lib/firebase'
import type { GameSlot, GameSummary, GameViewDocument, Season, StateData } from '@/types'

export interface SubscriptionError {
  code: string
  message: string
}

export interface GameSubscriptionState {
  summary: GameSummary | null
  view: GameViewDocument | null
  loading: boolean
  error: SubscriptionError | null
}

export interface GameListSubscriptionState {
  games: GameSummary[]
  enabled: boolean
  loading: boolean
  error: SubscriptionError | null
}

function timestampToISO(value: unknown): string | undefined {
  if (typeof value === 'string') return value
  if (value instanceof Date) return value.toISOString()
  if (typeof value === 'object' && value !== null) {
    const timestamp = value as { toDate?: () => Date; seconds?: number }
    if (timestamp.toDate) return timestamp.toDate().toISOString()
    if (typeof timestamp.seconds === 'number') {
      return new Date(timestamp.seconds * 1000).toISOString()
    }
  }
  return undefined
}

function stringValue(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback
}

function numberValue(value: unknown, fallback = 0): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function seasonValue(value: unknown): Season {
  return value === 'spring' ||
    value === 'summer' ||
    value === 'autumn' ||
    value === 'winter'
    ? value
    : 'spring'
}

function slotsFromData(data: DocumentData, currentUID?: string): GameSlot[] {
  const submittedUIDs = new Set(
    Array.isArray(data.submittedUids)
      ? data.submittedUids.filter((value): value is string => typeof value === 'string')
      : [],
  )
  const submittedPlayerIDs = new Set(
    Array.isArray(data.submitted)
      ? data.submitted.filter((value): value is string => typeof value === 'string')
      : [],
  )
  if (!Array.isArray(data.players)) return []

  return data.players.flatMap((value): GameSlot[] => {
    if (typeof value !== 'object' || value === null) return []
    const player = value as DocumentData
    const id = stringValue(player.id)
    if (!id) return []
    const actorId = stringValue(player.actorId) || undefined
    const currentActorId = actorId && actorId === currentUID ? actorId : undefined
    return [
      {
        id,
        name: stringValue(player.name, id),
        color: stringValue(player.color, '#b7a786'),
        actorId: currentActorId,
        submitted:
          player.submitted === true ||
          submittedUIDs.has(actorId ?? '') ||
          submittedPlayerIDs.has(id),
      },
    ]
  })
}

export function normalizeGameSummary(
  data: DocumentData,
  fallbackID = '',
  currentUID?: string,
): GameSummary {
  const winner = stringValue(data.winner ?? data.winnerUid) || null
  const status = data.status === 'finished' ? 'finished' : 'playing'
  const currentPlayer =
    stringValue(data.currentPlayer) ||
    (Array.isArray(data.players)
      ? stringValue(
          data.players.find(
            (player: unknown) =>
              typeof player === 'object' &&
              player !== null &&
              (player as DocumentData).actorId === currentUID,
          )?.id,
        )
      : '')
  const canInvite = data.canInvite === true || stringValue(data.ownerUid) === currentUID
  return {
    id: stringValue(data.id, fallbackID),
    name: stringValue(data.name, 'Crown & Borough'),
    seed: stringValue(data.seed),
    status,
    winner,
    ...(currentPlayer ? { currentPlayer } : {}),
    ...(canInvite ? { canInvite } : {}),
    players: slotsFromData(data, currentUID),
    turn: numberValue(data.turn, 1),
    season: seasonValue(data.season),
    revision: numberValue(data.revision),
    updatedAt: timestampToISO(data.updatedAt),
  }
}

export function normalizeStateData(value: unknown): StateData | null {
  if (typeof value !== 'object' || value === null) return null
  const state = value as Partial<StateData>
  if (
    typeof state.turn !== 'number' ||
    !Array.isArray(state.players) ||
    !Array.isArray(state.territories) ||
    !Array.isArray(state.nobles)
  ) {
    return null
  }
  return {
    turn: state.turn,
    season: seasonValue(state.season),
    players: state.players,
    territories: state.territories,
    nobles: state.nobles,
  }
}

function normalizeView(
  data: DocumentData,
  fallbackID: string,
  fallbackUID: string,
): GameViewDocument | null {
  const state = normalizeStateData(data.state ?? data)
  if (!state) return null
  return {
    gameId: stringValue(data.gameId, fallbackID),
    uid: stringValue(data.uid, fallbackUID),
    revision: numberValue(data.revision),
    turn: numberValue(data.turn, state.turn),
    season: seasonValue(data.season ?? state.season),
    state,
    updatedAt: timestampToISO(data.updatedAt),
  }
}

function mapFirestoreError(error: FirestoreError): SubscriptionError {
  const code = error.code || 'unknown'
  if (code === 'permission-denied') {
    return { code, message: 'permission denied' }
  }
  if (code === 'unauthenticated') {
    return { code, message: 'authentication is required' }
  }
  if (code === 'not-found') {
    return { code, message: 'the game projection was not found' }
  }
  return { code, message: error.message || 'real-time connection failed' }
}

export function useGameSubscription(
  gameId: string | undefined,
  uid: string | undefined,
  minimumRevision = 0,
): GameSubscriptionState {
  const services = getFirebaseServices()
  const subscriptionKey = `${gameId ?? ''}:${uid ?? ''}`
  const minimumRevisionRef = useRef({ key: subscriptionKey, revision: minimumRevision })
  if (minimumRevisionRef.current.key !== subscriptionKey) {
    minimumRevisionRef.current = { key: subscriptionKey, revision: minimumRevision }
  } else if (minimumRevision > minimumRevisionRef.current.revision) {
    minimumRevisionRef.current.revision = minimumRevision
  }
  const [state, setState] = useState<GameSubscriptionState>({
    summary: null,
    view: null,
    loading: Boolean(gameId && uid),
    error: null,
  })

  useEffect(() => {
    if (!services || !gameId || !uid) {
      setState({ summary: null, view: null, loading: false, error: null })
      return
    }

    let active = true
    let unsubscribeGame: () => void = () => undefined
    let unsubscribeView: () => void = () => undefined

    const stop = (error: SubscriptionError) => {
      if (!active) return
      active = false
      unsubscribeGame()
      unsubscribeView()
      setState((current) => ({ ...current, loading: false, error }))
    }

    setState({ summary: null, view: null, loading: true, error: null })
    try {
      unsubscribeGame = onSnapshot(
        doc(services.firestore, 'games', gameId),
        { includeMetadataChanges: true },
        (snapshot) => {
          if (!active) return
          if (!snapshot.exists()) {
            stop({ code: 'not-found', message: 'the game was not found' })
            return
          }
          const data = snapshot.data()
          if (Array.isArray(data.memberUids) && !data.memberUids.includes(uid)) {
            stop({
              code: 'not-member',
              message: 'the account is not a member of this game',
            })
            return
          }
          const summary = normalizeGameSummary(data, gameId, uid)
          if (summary.revision < minimumRevisionRef.current.revision) return
          setState((current) => ({
            ...current,
            summary,
            loading: current.view === null,
            error: null,
          }))
        },
        (error) => stop(mapFirestoreError(error)),
      )
      unsubscribeView = onSnapshot(
        doc(services.firestore, 'games', gameId, 'views', uid),
        { includeMetadataChanges: true },
        (snapshot) => {
          if (!active) return
          if (!snapshot.exists()) {
            stop({ code: 'not-found', message: 'the private game view was not found' })
            return
          }
          const view = normalizeView(snapshot.data(), gameId, uid)
          if (!view) {
            stop({ code: 'invalid-data', message: 'the private game view is invalid' })
            return
          }
          if (view.gameId !== gameId || view.uid !== uid) {
            stop({
              code: 'invalid-data',
              message: 'the private game view identity is invalid',
            })
            return
          }
          if (view.revision < minimumRevisionRef.current.revision) return
          setState((current) => ({
            ...current,
            view,
            loading: current.summary === null,
            error: null,
          }))
        },
        (error) => stop(mapFirestoreError(error)),
      )
    } catch (error) {
      stop({
        code: 'listener-failed',
        message: error instanceof Error ? error.message : 'real-time connection failed',
      })
    }

    return () => {
      active = false
      unsubscribeGame()
      unsubscribeView()
    }
  }, [gameId, services, uid])

  return state
}

export function useGameListSubscription(
  uid: string | undefined,
): GameListSubscriptionState {
  const services = getFirebaseServices()
  const [state, setState] = useState<GameListSubscriptionState>({
    games: [],
    enabled: Boolean(services && uid),
    loading: Boolean(uid),
    error: null,
  })

  useEffect(() => {
    if (!services || !uid) {
      setState({ games: [], enabled: false, loading: false, error: null })
      return
    }

    let active = true
    let unsubscribe: () => void = () => undefined
    setState({ games: [], enabled: true, loading: true, error: null })
    try {
      const gamesQuery = query(
        collection(services.firestore, 'games'),
        where('memberUids', 'array-contains', uid),
        orderBy('updatedAt', 'desc'),
      )
      unsubscribe = onSnapshot(
        gamesQuery,
        { includeMetadataChanges: true },
        (snapshot) => {
          if (!active) return
          setState({
            games: snapshot.docs.map((document) =>
              normalizeGameSummary(document.data(), document.id),
            ),
            enabled: true,
            loading: false,
            error: null,
          })
        },
        (error) => {
          if (!active) return
          setState((current) => ({
            ...current,
            enabled: true,
            loading: false,
            error: mapFirestoreError(error),
          }))
          if (error.code === 'permission-denied' || error.code === 'unauthenticated') {
            unsubscribe()
          }
        },
      )
    } catch (error) {
      setState({
        games: [],
        enabled: true,
        loading: false,
        error: {
          code: 'listener-failed',
          message: error instanceof Error ? error.message : 'real-time connection failed',
        },
      })
    }

    return () => {
      active = false
      unsubscribe()
    }
  }, [services, uid])

  return state
}
