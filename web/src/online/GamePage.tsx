import { useEffect, useMemo, useCallback, useRef, useState } from 'react'
import {
  IconArrowLeft,
  IconTrophy,
  IconUsersGroup,
  IconWifi,
  IconWifiOff,
} from '@tabler/icons-react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { GameLayout } from '@/components/GameLayout'
import { GamePanelCard } from '@/components/GamePanelCard'
import { MapViewer } from '@/components/MapViewer'
import { SelectedTerritoryDetails } from '@/components/SelectedTerritoryDetails'
import { OrdersPanel } from '@/components/OrdersPanel'
import { ReportPane, type ReportSummary } from '@/components/ReportPane'
import { RulesPanel, type RulesSection } from '@/components/RulesPanel'
import type { Panel } from '@/components/CommandReportRulesTabs'
import { Scoreboard } from '@/components/Scoreboard'
import { SubmissionDots } from '@/components/SubmissionDots'
import { Button } from '@/components/ui/button'
import { HeaderPopover } from '@/components/ui/header-popover'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ApiError, apiRequest, type TokenProvider } from '@/lib/api'
import { stripNobleHeader } from '@/lib/order-text'
import { buildOrdersBody } from '@/lib/orders-body'
import { draftOrdersByNoble } from '@/lib/transfer-preview'
import { useOrdersPreview, type OrdersPreviewRequest } from '@/lib/use-orders-preview'
import {
  internalYear,
  ownerName,
  remainingTurns,
  remainingYears,
} from '@/lib/game-progress'
import { useGameIntentions } from '@/lib/use-game-intentions'
import { useSupplyAndTransfer } from '@/lib/use-supply-and-transfer'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { useLocalStorageState } from '@/lib/storage'
import {
  normalizeGameSummary,
  normalizeStateData,
  useGameSubscription,
} from '@/lib/subscription'
import { useLanguage } from '@/i18n/LanguageContext'
import type {
  GameSummary,
  GameViewDocument,
  MapData,
  MySubmissionResponse,
  OrdersResponse,
  PlayerId,
  StateData,
  SubmittedOrdersResponse,
  TurnReport,
  OrdersPreview,
} from '@/types'

interface Invitation {
  gameId: string
  inviteCode: string
  inviteUrl: string
}

function errorText(error: unknown, fallback: string): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return fallback
}

function newerSummary(
  left: GameSummary | null,
  right: GameSummary | null,
): GameSummary | null {
  if (!left) return right
  if (!right) return left
  if (right.revision < left.revision) return left
  if (right.revision === left.revision) {
    if (!right.updatedAt) return left
    if (left.updatedAt && right.updatedAt < left.updatedAt) return left
  }
  return {
    ...right,
    ...(right.canInvite === undefined && left.canInvite !== undefined
      ? { canInvite: left.canInvite }
      : {}),
    ...(right.inviteAvailable === undefined && left.inviteAvailable !== undefined
      ? { inviteAvailable: left.inviteAvailable }
      : {}),
    ...(right.spectator === undefined && left.spectator !== undefined
      ? { spectator: left.spectator }
      : {}),
  }
}

function createView(
  gameId: string,
  uid: string,
  state: StateData,
  revision: number,
): GameViewDocument {
  return {
    gameId,
    uid,
    state,
    revision,
    turn: state.turn,
    season: state.season,
  }
}

function Lobby({
  summary,
  uid,
  currentPlayer,
  invitation,
  onInvite,
  inviting,
  scores,
}: {
  summary: GameSummary
  uid: string
  currentPlayer?: PlayerId
  invitation: Invitation | null
  onInvite: () => void
  inviting: boolean
  scores?: StateData['scores']
}) {
  const { t } = useLanguage()
  return (
    <Card className="border-[#b7a786] bg-[#fffaf0]">
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="font-serif text-2xl text-[#30291f]">
              {t('online.lobby')}
            </CardTitle>
            <CardDescription className="mt-1 text-[#806f57]">
              {summary.players.length} {t('app.players').toLowerCase()}
            </CardDescription>
          </div>
          {summary.canInvite && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={inviting || summary.inviteAvailable === false}
              onClick={onInvite}
            >
              {inviting ? t('online.inviteLoading') : t('online.invite')}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <ul className="grid gap-2 sm:grid-cols-2">
          {summary.players.map((player) => (
            <li key={player.id} className="rounded-md bg-[#f3ead9] px-3 py-2 text-sm">
              <div className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className="size-3 shrink-0 rounded-full border border-[#30291f]/30"
                  style={{ backgroundColor: player.color }}
                />
                <span className="min-w-0 flex-1 truncate">
                  {player.name || t('online.emptySlot')}
                </span>
                {(player.actorId === uid || player.id === currentPlayer) && (
                  <span className="shrink-0 text-[10px] font-semibold uppercase tracking-[0.1em] text-[#a84632]">
                    {t('online.you')}
                  </span>
                )}
              </div>
              <p className="mt-1 text-[10px] uppercase tracking-[0.1em] text-[#806f57]">
                {player.submitted ? t('online.submitted') : t('online.waiting')}
              </p>
            </li>
          ))}
        </ul>
        <Scoreboard players={summary.players} scores={scores} />
        {invitation && (
          <div className="rounded-lg border border-[#815f1e]/40 bg-[#f8e8ae]/50 px-3 py-3 text-sm">
            <p className="font-semibold text-[#6d5118]">
              {t('home.invitationCode')}: {invitation.inviteCode}
            </p>
            <p className="mt-1 break-all text-xs text-[#806f57]">
              {invitation.inviteUrl}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function GamePage() {
  const { gameId } = useParams<{ gameId: string }>()
  const { user, getIdToken, signOut } = useAuth()
  const { language, t } = useLanguage()
  const navigate = useNavigate()
  const [summaryFromAPI, setSummaryFromAPI] = useState<GameSummary | null>(null)
  const [restView, setRestView] = useState<GameViewDocument | null>(null)
  const [map, setMap] = useState<MapData | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const [chainDrafts, setChainDrafts] = useState<Record<string, string>>({})
  const [winterDraft, setWinterDraft] = useState('')
  const [serverSubmission, setServerSubmission] = useState<MySubmissionResponse | null>(
    null,
  )
  const [submittedOrders, setSubmittedOrders] = useState<SubmittedOrdersResponse | null>(
    null,
  )
  const [specialDraft, setSpecialDraft] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [confirmResolve, setConfirmResolve] = useState(false)
  const [activePanel, setActivePanel] = useState<Panel>('command')
  const [showOwnership, setShowOwnership] = useLocalStorageState(
    'cb.ownershipOverlay',
    true,
  )
  const [showRegions, setShowRegions] = useLocalStorageState('cb.regionsOverlay', true)
  const [rulesNavigation, setRulesNavigation] = useState<{
    section: RulesSection
    key: number
  } | null>(null)
  const [invitation, setInvitation] = useState<Invitation | null>(null)
  const [inviting, setInviting] = useState(false)
  const [report, setReport] = useState<TurnReport | null>(null)
  const [reportSummaries, setReportSummaries] = useState<ReportSummary[]>([])
  const [reportLoading, setReportLoading] = useState(false)
  const [reportError, setReportError] = useState<string | null>(null)
  const [offline, setOffline] = useState(!window.navigator.onLine)
  const [showIntentions, setShowIntentions] = useLocalStorageState(
    'cb.intentionsOverlay',
    true,
  )
  const [showCalamities, setShowCalamities] = useLocalStorageState(
    'cb.calamitiesOverlay',
    true,
  )
  const [showCards, setShowCards] = useLocalStorageState('cb.cardsOverlay', true)
  const [mapFocusSignal, setMapFocusSignal] = useState(0)
  const lastTurn = useRef<number | null>(null)
  const hydratedTurnRef = useRef<number | null>(null)
  const submittedOrdersContextRef = useRef<string | null>(null)
  const tokenProvider: TokenProvider = { getIdToken }

  const subscription = useGameSubscription(gameId, user?.uid, restView?.revision ?? 0)

  useEffect(() => {
    if (!gameId || !user) return
    const controller = new AbortController()
    const encodedID = encodeURIComponent(gameId)
    setSummaryFromAPI(null)
    setRestView(null)
    setMap(null)
    setSelectedId(null)
    setServerSubmission(null)
    setSubmittedOrders(null)
    setReport(null)
    setReportSummaries([])
    setReportError(null)
    setLoadError(null)
    void Promise.all([
      apiRequest<unknown>({ getIdToken }, `/api/games/${encodedID}`),
      apiRequest<MapData>({ getIdToken }, `/api/games/${encodedID}/map`),
      apiRequest<StateData & { revision: number }>(
        { getIdToken },
        `/api/games/${encodedID}/state`,
      ),
      apiRequest<MySubmissionResponse>(
        { getIdToken },
        `/api/games/${encodedID}/my-submission`,
      ).catch(() => null),
    ])
      .then(([detail, mapData, stateResponse, mySubmission]) => {
        if (controller.signal.aborted) return
        const summary = normalizeGameSummary(
          detail as Record<string, unknown>,
          gameId,
          user.uid,
        )
        const state = normalizeStateData(stateResponse)
        if (!state) throw new Error('the private state is invalid')
        setSummaryFromAPI(summary)
        setMap(mapData)
        setRestView(createView(gameId, user.uid, state, stateResponse.revision))
        if (mySubmission) {
          setServerSubmission(mySubmission)
        }
      })
      .catch((loadFailure: unknown) => {
        if (controller.signal.aborted) return
        if (loadFailure instanceof ApiError && loadFailure.status === 401) {
          void signOut().catch(() => undefined)
          navigate('/signin', { replace: true })
          return
        }
        setLoadError(errorText(loadFailure, t('error.serverUnavailable')))
      })
    return () => controller.abort()
  }, [gameId, getIdToken, navigate, signOut, t, user])

  useEffect(() => {
    const handleOnline = () => setOffline(false)
    const handleOffline = () => setOffline(true)
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [])

  const summary = newerSummary(summaryFromAPI, subscription.summary)
  const summaryRevision = summary?.revision
  const hasSummary = Boolean(summary)
  const view =
    subscription.view && (!restView || subscription.view.revision >= restView.revision)
      ? subscription.view
      : restView
  const state = view?.state ?? null
  const currentSlot = summary?.players.find(
    (player) =>
      player.actorId === user?.uid ||
      (summary.currentPlayer !== undefined && player.id === summary.currentPlayer),
  )
  const playerID = currentSlot?.id ?? null
  const spectator = summary?.spectator === true
  const stateTurn = state?.turn
  const stateSeason = state?.season

  useEffect(() => {
    if (!gameId || !spectator || stateTurn === undefined || stateSeason === undefined) {
      if (!spectator) {
        submittedOrdersContextRef.current = null
        setSubmittedOrders(null)
      }
      return
    }
    const contextKey = `${gameId}:${stateTurn}:${stateSeason}`
    if (submittedOrdersContextRef.current !== contextKey) {
      submittedOrdersContextRef.current = contextKey
      setSubmittedOrders(null)
    }
    let active = true
    const encodedID = encodeURIComponent(gameId)
    void apiRequest<SubmittedOrdersResponse>(
      { getIdToken },
      `/api/games/${encodedID}/submitted-orders`,
    )
      .then((response) => {
        if (active && response.turn === stateTurn && response.season === stateSeason) {
          setSubmittedOrders(response)
        }
      })
      .catch((submissionFailure: unknown) => {
        if (!active) return
        if (submissionFailure instanceof ApiError && submissionFailure.status === 401) {
          void signOut().catch(() => undefined)
          navigate('/signin', { replace: true })
        }
      })
    return () => {
      active = false
    }
  }, [
    gameId,
    getIdToken,
    navigate,
    signOut,
    spectator,
    stateSeason,
    stateTurn,
    summaryRevision,
  ])

  const serverChains = useMemo(() => {
    const result: Record<string, string> = {}
    for (const chain of serverSubmission?.chains ?? []) {
      result[chain.noble] = stripNobleHeader(chain.noble, chain.text)
    }
    return result
  }, [serverSubmission])

  useEffect(() => {
    if (!serverSubmission || !state || serverSubmission.turn !== state.turn) return
    if (hydratedTurnRef.current === state.turn) return
    hydratedTurnRef.current = state.turn

    if (serverSubmission.submitted) {
      if (serverSubmission.season === 'winter') {
        setWinterDraft((current) =>
          current.trim() === '' ? (serverSubmission.winter?.lines ?? '') : current,
        )
      } else {
        setChainDrafts((current) => {
          const next = { ...current }
          for (const chain of serverSubmission.chains) {
            if ((next[chain.noble] ?? '').trim() === '') {
              next[chain.noble] = stripNobleHeader(chain.noble, chain.text)
            }
          }
          return next
        })
      }
    }
  }, [serverSubmission, state])

  const draftDiffers = useMemo(() => {
    if (
      !serverSubmission ||
      !serverSubmission.submitted ||
      !state ||
      serverSubmission.turn !== state.turn
    ) {
      return undefined
    }
    if (state.season === 'winter') {
      const serverWinter = (serverSubmission.winter?.lines ?? '').trim()
      const localWinter = winterDraft.trim()
      return {
        winter: localWinter !== serverWinter,
      }
    }
    const chainsDiff: Record<string, boolean> = {}
    let anyChainDiff = false
    for (const noble of state.nobles) {
      if (noble.owner !== playerID || noble.status === 'dungeon') continue
      const serverChain = (serverChains[noble.code] ?? '').trim()
      const localChain = (chainDrafts[noble.code] ?? '').trim()
      if (localChain !== serverChain) {
        chainsDiff[noble.code] = true
        anyChainDiff = true
      }
    }
    return {
      chains: anyChainDiff ? chainsDiff : undefined,
    }
  }, [chainDrafts, playerID, serverChains, serverSubmission, state, winterDraft])

  const restoreFromServer = (target?: string) => {
    if (!serverSubmission || !state) return
    if (target === 'winter' || state.season === 'winter') {
      setWinterDraft(serverSubmission.winter?.lines ?? '')
      return
    }
    if (target) {
      const serverText = serverChains[target] ?? ''
      setChainDrafts((current) => ({
        ...current,
        [target]: serverText,
      }))
      return
    }
    setChainDrafts(serverChains)
  }

  useEffect(() => {
    const turn = state?.turn ?? null
    if (turn === null || lastTurn.current === null) {
      lastTurn.current = turn
      return
    }
    if (turn !== lastTurn.current) {
      setChainDrafts({})
      setWinterDraft('')
      setServerSubmission(null)
      setSubmittedOrders(null)
      setSpecialDraft('')
      setActionError(null)
      lastTurn.current = turn
    }
  }, [state?.turn])

  const selectedTerritory = map?.territories.find(
    (territory) => territory.id === selectedId,
  )
  const selectedState =
    state?.territories.find((territory) => territory.id === selectedId) ?? null
  const selectedRegion = map?.regions?.find((region) =>
    region.territories.includes(selectedId ?? ''),
  )

  const handleAuthError = useCallback(() => {
    void signOut().catch(() => undefined)
    navigate('/signin', { replace: true })
  }, [navigate, signOut])

  const supplyFetcher = useCallback(
    <T,>(path: string, signal: AbortSignal) =>
      apiRequest<T>({ getIdToken }, path, { signal }),
    [getIdToken],
  )

  const ordersBody = useMemo(
    () =>
      state && playerID
        ? buildOrdersBody(state, playerID, { chainDrafts, winterDraft, specialDraft })
        : null,
    [chainDrafts, playerID, specialDraft, state, winterDraft],
  )
  const previewRequest = useMemo<OrdersPreviewRequest | null>(() => {
    if (!gameId || !playerID) return null
    const path = `/api/games/${encodeURIComponent(gameId)}/orders/preview?lang=${language}`
    return (body, signal) =>
      apiRequest<OrdersPreview>({ getIdToken }, path, {
        method: 'POST',
        body: JSON.stringify(body),
        signal,
      })
  }, [gameId, getIdToken, language, playerID])
  const preview = useOrdersPreview(ordersBody, previewRequest)
  const draftOrders = useMemo(() => draftOrdersByNoble(preview?.chains), [preview])

  const {
    selectedSupplyLine,
    sourceTerritoryId,
    supplyLoading,
    supplyError,
    transferLine,
    transferLoading,
    transferError,
    transferTargets,
    transferTarget,
    setTransferTarget: setSelectedTransferTarget,
  } = useSupplyAndTransfer({
    selectedId,
    state,
    selectedState,
    draftOrders,
    ownerId: playerID,
    basePath: gameId ? `/api/games/${encodeURIComponent(gameId)}` : '/api',
    fetcher: supplyFetcher,
    networkErrorMessage: t('error.network'),
    onAuthError: handleAuthError,
  })
  const sourceTerritory = sourceTerritoryId
    ? (map?.territories.find((territory) => territory.id === sourceTerritoryId) ?? null)
    : null

  const { intentions, winterIntentions, intentionsColor } = useGameIntentions({
    state,
    map,
    playerID,
    preview,
    winterText: ordersBody?.winter[0]?.lines ?? '',
    spectator,
    submittedOrders,
  })

  useEffect(() => {
    if (!gameId || !hasSummary || !user) return
    let active = true
    setReportLoading(true)
    void apiRequest<ReportSummary[]>(
      { getIdToken },
      `/api/games/${encodeURIComponent(gameId)}/reports`,
    )
      .then(async (summaries) => {
        if (!active) return
        setReportSummaries(summaries)
        const latest = summaries.at(-1)
        if (!latest) {
          setReport(null)
          return
        }
        const latestReport = await apiRequest<TurnReport>(
          { getIdToken },
          `/api/games/${encodeURIComponent(gameId)}/reports/${latest.index}`,
        )
        if (active) setReport(latestReport)
      })
      .catch((reportFailure: unknown) => {
        if (!active) return
        if (reportFailure instanceof ApiError && reportFailure.status === 401) {
          void signOut().catch(() => undefined)
          navigate('/signin', { replace: true })
          return
        }
        setReportError(errorText(reportFailure, t('error.network')))
      })
      .finally(() => {
        if (active) setReportLoading(false)
      })
    return () => {
      active = false
    }
  }, [gameId, getIdToken, hasSummary, navigate, signOut, summaryRevision, t, user])

  const loadReport = async (index: number) => {
    if (!gameId) return
    setReportLoading(true)
    setReportError(null)
    try {
      const nextReport = await apiRequest<TurnReport>(
        { getIdToken },
        `/api/games/${encodeURIComponent(gameId)}/reports/${index}`,
      )
      setReport(nextReport)
      setActivePanel('report')
    } catch (reportFailure) {
      if (reportFailure instanceof ApiError && reportFailure.status === 401) {
        await signOut().catch(() => undefined)
        navigate('/signin', { replace: true })
        return
      }
      setReportError(errorText(reportFailure, t('error.network')))
    } finally {
      setReportLoading(false)
    }
  }

  const applyOrdersResponse = (response: OrdersResponse) => {
    if (!gameId || !user) return
    const nextState = normalizeStateData(response.state)
    if (!nextState) throw new Error('the server returned an invalid private state')
    setRestView(
      createView(gameId, user.uid, nextState, response.revision ?? view?.revision ?? 0),
    )
    setSummaryFromAPI((current) => {
      if (!current) return current
      const submitted = new Set(response.submitted)
      const remaining = new Set(response.remaining)
      return {
        ...current,
        turn: nextState.turn,
        season: nextState.season,
        status: nextState.finished ? 'finished' : current.status,
        winner: nextState.finished ? (nextState.winner ?? null) : current.winner,
        revision: response.revision ?? current.revision,
        scores: nextState.scores ?? current.scores,
        players: current.players.map((player) => ({
          ...player,
          submitted: submitted.has(player.id),
          // A player absent from both lists has nothing left to submit this
          // turn (see turn.Progress on the server).
          required: submitted.has(player.id) || remaining.has(player.id),
        })),
      }
    })
    if (response.report) {
      setReport(response.report)
      setActivePanel('report')
    }
    if (response.status === 'resolved' || response.resolved) {
      setChainDrafts({})
      setWinterDraft('')
      setServerSubmission(null)
      setSpecialDraft('')
      if (!response.report) {
        setReport(null)
        setActivePanel('report')
      }
    }
  }

  const submitOrders = async (force = false) => {
    if (
      !gameId ||
      !state ||
      (!playerID && !force) ||
      state.finished ||
      summary?.status === 'finished'
    )
      return
    setSubmitting(true)
    setActionError(null)
    try {
      let response: OrdersResponse
      if (force) {
        response = await apiRequest<OrdersResponse>(
          { getIdToken },
          `/api/games/${encodeURIComponent(gameId)}/resolve`,
          { method: 'POST' },
        )
      } else {
        const body = buildOrdersBody(state, playerID!, {
          chainDrafts,
          winterDraft,
          specialDraft,
        })
        response = await apiRequest<OrdersResponse>(
          { getIdToken },
          `/api/games/${encodeURIComponent(gameId)}/orders?lang=${language}`,
          {
            method: 'POST',
            body: JSON.stringify({
              ...body,
              revision: summary?.revision ?? view?.revision ?? 0,
            }),
          },
        )
        setServerSubmission({
          turn: state.turn,
          season: state.season,
          submitted: true,
          chains: body.chains,
          winter: body.winter[0],
        })
      }
      applyOrdersResponse(response)
      setConfirmResolve(false)
    } catch (submitFailure) {
      if (submitFailure instanceof ApiError && submitFailure.status === 401) {
        await signOut().catch(() => undefined)
        navigate('/signin', { replace: true })
        return
      } else if (
        submitFailure instanceof ApiError &&
        submitFailure.code === 'revision_conflict'
      ) {
        setActionError(t('error.revisionConflict'))
      } else {
        setActionError(errorText(submitFailure, t('error.resolutionFailed')))
      }
    } finally {
      setSubmitting(false)
    }
  }

  const createInvitation = async () => {
    if (!gameId) return
    setInviting(true)
    try {
      const nextInvitation = await apiRequest<Invitation>(
        { getIdToken },
        `/api/games/${encodeURIComponent(gameId)}/invite`,
      )
      setInvitation(nextInvitation)
    } catch (inviteFailure) {
      if (inviteFailure instanceof ApiError && inviteFailure.status === 401) {
        void signOut().catch(() => undefined)
        navigate('/signin', { replace: true })
        return
      }
      setActionError(errorText(inviteFailure, t('error.serverUnavailable')))
    } finally {
      setInviting(false)
    }
  }

  const openRules = (section: RulesSection) => {
    setRulesNavigation((current) => ({ section, key: (current?.key ?? 0) + 1 }))
    setActivePanel('rules')
  }

  const handleTerritorySelect = (id: string | null) => {
    setSelectedId(id)
    if (id) {
      setMapFocusSignal((signal) => signal + 1)
    }
  }

  if (loadError) {
    return (
      <div className="space-y-4">
        <Link
          to="/"
          className="inline-flex items-center gap-2 text-sm font-semibold text-[#a84632]"
        >
          <IconArrowLeft aria-hidden="true" className="size-4" /> {t('online.backHome')}
        </Link>
        <p
          role="alert"
          className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-4 py-3 text-sm text-[#8d321e]"
        >
          {loadError}
        </p>
      </div>
    )
  }

  if (
    subscription.error?.code === 'permission-denied' ||
    subscription.error?.code === 'not-member'
  ) {
    return (
      <div className="space-y-4">
        <Link
          to="/"
          className="inline-flex items-center gap-2 text-sm font-semibold text-[#a84632]"
        >
          <IconArrowLeft aria-hidden="true" className="size-4" /> {t('online.backHome')}
        </Link>
        <p
          role="alert"
          className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-4 py-3 text-sm text-[#8d321e]"
        >
          {t('online.accessRevoked')}
        </p>
      </div>
    )
  }

  if (!summary || !map || !state) {
    return (
      <p className="py-20 text-center font-serif text-lg italic text-[#806f57]">
        {t('online.loading')}
      </p>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 sm:gap-4">
      <div className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-1">
        <Link
          to="/"
          aria-label={t('online.backHome')}
          className="inline-flex shrink-0 items-center gap-1.5 text-sm font-semibold text-[#a84632]"
        >
          <IconArrowLeft aria-hidden="true" className="size-4" />
          <span className="hidden min-[420px]:inline">{t('online.backHome')}</span>
        </Link>
        <h1 className="min-w-0 flex-1 truncate font-serif text-lg font-semibold text-[#30291f] sm:flex-none sm:text-2xl">
          {summary.name}
        </h1>
        <p className="text-xs font-semibold text-[#30291f]">
          {t('online.currentTurn', {
            turn: state.turn,
            season: t(SEASON_LABEL_KEYS[state.season]),
          })}
        </p>
        <div className="ml-auto flex items-center gap-2">
          <HeaderPopover
            label={t('app.scores')}
            icon={<IconTrophy aria-hidden="true" className="size-4" />}
            hint={String(
              Math.max(
                0,
                ...Object.values(state.scores ?? summary.scores ?? {}).map(
                  (score) => score.total ?? 0,
                ),
              ),
            )}
          >
            <Scoreboard players={state.players} scores={state.scores ?? summary.scores} />
          </HeaderPopover>
          <HeaderPopover
            label={t('online.lobby')}
            icon={<IconUsersGroup aria-hidden="true" className="size-4" />}
            hint={`${summary.players.filter((player) => player.required && player.submitted).length}/${summary.players.filter((player) => player.required).length}`}
          >
            <Lobby
              summary={summary}
              uid={user?.uid ?? ''}
              currentPlayer={summary.currentPlayer}
              invitation={invitation}
              onInvite={() => void createInvitation()}
              inviting={inviting}
              scores={state.scores ?? summary.scores}
            />
          </HeaderPopover>
          <SubmissionDots
            players={summary.players.map((player) => ({
              id: player.id,
              name: player.name || player.id,
              color: player.color,
              submitted: player.submitted,
              required: player.required,
              isYou: player.id === playerID,
            }))}
          />
          <span
            className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs font-semibold ${offline ? 'border-[#a84632]/30 bg-[#f8e5dd] text-[#8d321e]' : 'border-[#376341]/30 bg-[#e8f1e3] text-[#376341]'}`}
          >
            {offline ? (
              <IconWifiOff aria-hidden="true" className="size-3.5" />
            ) : (
              <IconWifi aria-hidden="true" className="size-3.5" />
            )}
            {offline ? t('online.networkOffline') : t('online.realtime')}
          </span>
          {summary.status === 'finished' && (
            <span className="rounded-full border border-[#815f1e]/40 bg-[#f8e8ae]/60 px-2 py-1 text-xs font-semibold text-[#6d5118]">
              {t('online.finished')}
            </span>
          )}
        </div>
      </div>
      <p className="shrink-0 text-[11px] text-[#806f57]">
        {t('app.year', { year: 1000 + internalYear(state) })} ·{' '}
        {t('app.remainingYearsTurns', {
          years: remainingYears(state, summary.yearCount),
          turns: remainingTurns(state, summary.yearCount),
        })}
      </p>

      {subscription.error &&
        subscription.error.code !== 'permission-denied' &&
        subscription.error.code !== 'not-member' && (
          <p
            role="status"
            className="rounded-lg border border-[#815f1e]/40 bg-[#f8e8ae]/50 px-3 py-2 text-sm text-[#6d5118]"
          >
            {t('online.realtimeError')}
          </p>
        )}
      {actionError && (
        <p
          role="alert"
          className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-sm text-[#8d321e]"
        >
          {actionError}
        </p>
      )}
      {spectator && (
        <div className="rounded-xl border border-[#815f1e]/50 bg-[#f8e8ae]/60 px-4 py-3 text-sm text-[#6d5118]">
          <p className="font-semibold">{t('online.spectatorBanner')}</p>
          <p className="mt-1">{t('online.spectatorDescription')}</p>
        </div>
      )}
      {summary.winner && (
        <div className="rounded-xl border border-[#815f1e]/50 bg-[#f8e8ae]/60 px-4 py-3 text-center text-sm font-semibold text-[#6d5118]">
          {t('online.victory')}:{' '}
          {ownerName(summary.winner, state, summary.players, summary.winner)}
        </div>
      )}

      <GameLayout
        map={
          <MapViewer
            map={map}
            state={state}
            supply={selectedSupplyLine}
            onSelect={handleTerritorySelect}
            intentions={intentions}
            winterIntentions={winterIntentions}
            showIntentions={showIntentions}
            intentionsColor={intentionsColor}
            onToggleIntentions={setShowIntentions}
            showOwnership={showOwnership}
            onToggleOwnership={setShowOwnership}
            showRegions={showRegions}
            onToggleRegions={setShowRegions}
            showCalamities={showCalamities}
            onToggleCalamities={setShowCalamities}
            showCards={showCards}
            onToggleCards={setShowCards}
            specialOrders={
              specialDraft.trim() !== '' && playerID
                ? [{ player: playerID, text: specialDraft }]
                : []
            }
          />
        }
        focusSignal={mapFocusSignal}
      >
        <GamePanelCard
          activePanel={activePanel}
          onPanelChange={setActivePanel}
          subtitle={
            currentSlot
              ? `${currentSlot.name} · ${t('online.you')}`
              : spectator
                ? t('online.spectator')
                : t('online.accessRevoked')
          }
          reportLabelExtra={
            report ? (
              <span className="ml-1 text-[#806f57]">· {report.header.turn}</span>
            ) : null
          }
          command={
            <>
              <SelectedTerritoryDetails
                state={state}
                selectedTerritory={selectedTerritory}
                selectedState={selectedState}
                selectedRegion={selectedRegion}
                preferredPlayers={summary.players}
                mapTerritories={map.territories}
                selectedSupplyLine={selectedSupplyLine}
                sourceTerritory={sourceTerritory}
                supplyLoading={supplyLoading}
                supplyError={supplyError}
                transferTargets={transferTargets}
                selectedTransferTarget={transferTarget}
                onTransferTargetChange={setSelectedTransferTarget}
                transferLine={transferLine}
                transferLoading={transferLoading}
                transferError={transferError}
              />
              {playerID ? (
                <OrdersPanel
                  state={state}
                  player={playerID}
                  chainDrafts={chainDrafts}
                  winterDraft={winterDraft}
                  specialDraft={specialDraft}
                  preview={preview}
                  submitted={Boolean(currentSlot?.submitted)}
                  submitting={submitting}
                  error={actionError}
                  draftDiffers={draftDiffers}
                  onChainChange={(noble, text) =>
                    setChainDrafts((current) => ({ ...current, [noble]: text }))
                  }
                  onWinterChange={setWinterDraft}
                  onSpecialChange={setSpecialDraft}
                  onSubmit={() => void submitOrders()}
                  onOpenRules={openRules}
                  onRestoreFromServer={restoreFromServer}
                />
              ) : spectator ? (
                <p className="rounded-lg border border-[#815f1e]/40 bg-[#f8e8ae]/50 px-3 py-2 text-sm text-[#6d5118]">
                  {t('online.spectatorReadOnly')}
                </p>
              ) : (
                <p
                  role="alert"
                  className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-sm text-[#8d321e]"
                >
                  {t('online.accessRevoked')}
                </p>
              )}
              {summary.status !== 'finished' && summary.canInvite && (
                <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                  {!confirmResolve ? (
                    <Button
                      type="button"
                      variant="outline"
                      className="w-full"
                      disabled={submitting}
                      onClick={() => setConfirmResolve(true)}
                    >
                      {t('online.forceResolve')}
                    </Button>
                  ) : (
                    <div className="rounded-lg border border-[#815f1e]/40 bg-[#f8e8ae]/50 p-3 text-sm text-[#6d5118]">
                      <p>{t('online.forceResolveConfirm')}</p>
                      <div className="mt-3 flex gap-2">
                        <Button
                          type="button"
                          size="sm"
                          disabled={submitting}
                          onClick={() => void submitOrders(true)}
                        >
                          {t('online.forceResolve')}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => setConfirmResolve(false)}
                        >
                          {t('online.cancel')}
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </>
          }
          report={
            <ReportPane
              report={report}
              map={map}
              players={state.players}
              summaries={reportSummaries}
              loading={reportLoading}
              error={reportError}
              onSelectReport={(index) => void loadReport(index)}
            />
          }
          rules={
            <RulesPanel
              gameId={gameId}
              tokenProvider={tokenProvider}
              targetSection={rulesNavigation?.section}
              navigationKey={rulesNavigation?.key}
            />
          }
        />
      </GameLayout>
    </div>
  )
}
