import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from 'react'
import { ArrowLeft, BookOpen, Wifi, WifiOff } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { MapLegend } from '@/components/MapLegend'
import { MapViewer } from '@/components/MapViewer'
import { SelectedTerritoryDetails } from '@/components/SelectedTerritoryDetails'
import { OrdersPanel } from '@/components/OrdersPanel'
import { ReportPane, type ReportSummary } from '@/components/ReportPane'
import { RulesPanel, type RulesSection } from '@/components/RulesPanel'
import { Scoreboard } from '@/components/Scoreboard'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ApiError, apiRequest, type TokenProvider } from '@/lib/api'
import { buildIntentions } from '@/lib/intent-overlay'
import { hasSupplySource } from '@/lib/supply'
import { addNobleHeader, hasChainContent } from '@/lib/order-text'
import { playerDisplayName, type PlayerName } from '@/lib/player-label'
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
  OrdersResponse,
  PlayerId,
  StateData,
  SupplyLine,
  TurnReport,
} from '@/types'

type Panel = 'command' | 'report' | 'rules'
const panelOrder: Panel[] = ['command', 'report', 'rules']

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
  }
}

function ownerName(
  owner: PlayerId | null,
  state: StateData,
  preferredPlayers: readonly PlayerName[],
  fallback: string,
): string {
  return playerDisplayName(owner, [preferredPlayers, state.players], fallback)
}

function internalYear(state: StateData): number {
  const year = state.year ?? Math.floor((state.turn - 1) / 4) + 1
  if (state.finished && state.yearCount && state.turn > state.yearCount * 4) {
    return state.yearCount
  }
  return year
}

function remainingYears(state: StateData, fallbackYearCount = 10): number {
  if (state.finished) return 0
  const yearCount = state.yearCount ?? fallbackYearCount
  return Math.max(0, yearCount - internalYear(state) + 1)
}

function remainingTurns(state: StateData, fallbackYearCount = 10): number {
  const yearCount = state.yearCount ?? fallbackYearCount
  return Math.max(0, yearCount * 4 - state.turn + 1)
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

function panelKeyDown(
  event: ReactKeyboardEvent<HTMLButtonElement>,
  activePanel: Panel,
  setActivePanel: (panel: Panel) => void,
) {
  const currentIndex = panelOrder.indexOf(activePanel)
  let nextIndex: number | null = null
  if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    nextIndex = (currentIndex + 1) % panelOrder.length
  } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
    nextIndex = (currentIndex - 1 + panelOrder.length) % panelOrder.length
  } else if (event.key === 'Home') {
    nextIndex = 0
  } else if (event.key === 'End') {
    nextIndex = panelOrder.length - 1
  }
  if (nextIndex === null) return
  event.preventDefault()
  const nextPanel = panelOrder[nextIndex]
  setActivePanel(nextPanel)
  event.currentTarget.parentElement
    ?.querySelector<HTMLButtonElement>(`[data-online-panel-tab="${nextPanel}"]`)
    ?.focus()
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
        <ul className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
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
  const { t } = useLanguage()
  const navigate = useNavigate()
  const [summaryFromAPI, setSummaryFromAPI] = useState<GameSummary | null>(null)
  const [restView, setRestView] = useState<GameViewDocument | null>(null)
  const [map, setMap] = useState<MapData | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [supplyLine, setSupplyLine] = useState<SupplyLine | null>(null)
  const [supplyError, setSupplyError] = useState<string | null>(null)
  const [supplyLoading, setSupplyLoading] = useState(false)
  const [chainDrafts, setChainDrafts] = useState<Record<string, string>>({})
  const [winterDraft, setWinterDraft] = useState('')
  const [specialDraft, setSpecialDraft] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [confirmResolve, setConfirmResolve] = useState(false)
  const [activePanel, setActivePanel] = useState<Panel>('command')
  const [showRegions, setShowRegions] = useState(false)
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
  const lastTurn = useRef<number | null>(null)
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
    setSupplyLine(null)
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
    ])
      .then(([detail, mapData, stateResponse]) => {
        if (controller.signal.aborted) return
        const summary = normalizeGameSummary(detail as Record<string, unknown>, gameId)
        const state = normalizeStateData(stateResponse)
        if (!state) throw new Error('the private state is invalid')
        setSummaryFromAPI(summary)
        setMap(mapData)
        setRestView(createView(gameId, user.uid, state, stateResponse.revision))
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

  const intentions = useMemo(
    () =>
      state && playerID
        ? buildIntentions(map ?? { territories: [] }, state, playerID, chainDrafts)
        : [],
    [chainDrafts, map, playerID, state],
  )
  const intentionsColor =
    state?.players.find((player) => player.id === playerID)?.color ?? '#a84632'

  useEffect(() => {
    const turn = state?.turn ?? null
    if (turn === null || lastTurn.current === null) {
      lastTurn.current = turn
      return
    }
    if (turn !== lastTurn.current) {
      setChainDrafts({})
      setWinterDraft('')
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

  useEffect(() => {
    if (
      !gameId ||
      !selectedId ||
      !state ||
      (!selectedState?.army && !hasSupplySource(selectedState ?? undefined))
    ) {
      setSupplyLine(null)
      setSupplyError(null)
      setSupplyLoading(false)
      return
    }
    if (state.season === 'winter') {
      setSupplyLine(null)
      setSupplyLoading(false)
      return
    }
    const controller = new AbortController()
    setSupplyLoading(true)
    setSupplyError(null)
    void apiRequest<SupplyLine>(
      { getIdToken },
      `/api/games/${encodeURIComponent(gameId)}/supply?territory=${encodeURIComponent(selectedId)}`,
    )
      .then((line) => {
        if (!controller.signal.aborted) setSupplyLine(line)
      })
      .catch((supplyFailure: unknown) => {
        if (controller.signal.aborted) return
        if (supplyFailure instanceof ApiError && supplyFailure.status === 401) {
          void signOut().catch(() => undefined)
          navigate('/signin', { replace: true })
          return
        }
        setSupplyError(errorText(supplyFailure, t('error.network')))
      })
      .finally(() => {
        if (!controller.signal.aborted) setSupplyLoading(false)
      })
    return () => controller.abort()
  }, [gameId, getIdToken, navigate, selectedId, selectedState, signOut, state, t])

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
      !playerID ||
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
        const chains =
          state.season === 'winter'
            ? []
            : state.nobles
                .filter((noble) => noble.owner === playerID && noble.status !== 'dungeon')
                .map((noble) => ({
                  noble: noble.code,
                  text: addNobleHeader(noble.code, chainDrafts[noble.code] ?? ''),
                }))
                .filter((chain) => hasChainContent(chain.noble, chain.text))
        const winter =
          state.season === 'winter' && winterDraft.trim() !== ''
            ? [{ lines: winterDraft }]
            : []
        const special = specialDraft.trim() !== '' ? [{ text: specialDraft }] : []
        response = await apiRequest<OrdersResponse>(
          { getIdToken },
          `/api/games/${encodeURIComponent(gameId)}/orders`,
          {
            method: 'POST',
            body: JSON.stringify({
              chains,
              winter,
              special,
              revision: summary?.revision ?? view?.revision ?? 0,
            }),
          },
        )
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

  if (loadError) {
    return (
      <div className="space-y-4">
        <Link
          to="/"
          className="inline-flex items-center gap-2 text-sm font-semibold text-[#a84632]"
        >
          <ArrowLeft aria-hidden="true" className="size-4" /> {t('online.backHome')}
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
          <ArrowLeft aria-hidden="true" className="size-4" /> {t('online.backHome')}
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

  const supplySelectionAllowed =
    (supplyLine?.kind === 'army' && Boolean(selectedState?.army)) ||
    (supplyLine?.kind === 'source' &&
      !selectedState?.army &&
      hasSupplySource(selectedState ?? undefined))
  const selectedSupplyLine =
    supplySelectionAllowed && supplyLine?.territory === selectedId ? supplyLine : null
  const sourceTerritory = selectedSupplyLine?.source
    ? map.territories.find((territory) => territory.id === selectedSupplyLine.source)
    : null

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link
            to="/"
            className="mb-2 inline-flex items-center gap-2 text-sm font-semibold text-[#a84632]"
          >
            <ArrowLeft aria-hidden="true" className="size-4" /> {t('online.backHome')}
          </Link>
          <h1 className="font-serif text-3xl font-semibold text-[#30291f]">
            {summary.name}
          </h1>
          <p className="mt-1 text-sm text-[#806f57]">
            {t('online.currentTurn', {
              turn: state.turn,
              season: t(SEASON_LABEL_KEYS[state.season]),
            })}
          </p>
          <p className="mt-1 text-sm font-semibold uppercase tracking-[0.12em] text-[#a84632]">
            {t('app.year', { year: 1000 + internalYear(state) })}
          </p>
          <p className="mt-1 text-xs text-[#806f57]">
            {t('app.remainingYearsTurns', {
              years: remainingYears(state, summary.yearCount),
              turns: remainingTurns(state, summary.yearCount),
            })}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold ${offline ? 'border-[#a84632]/30 bg-[#f8e5dd] text-[#8d321e]' : 'border-[#376341]/30 bg-[#e8f1e3] text-[#376341]'}`}
          >
            {offline ? (
              <WifiOff aria-hidden="true" className="size-3.5" />
            ) : (
              <Wifi aria-hidden="true" className="size-3.5" />
            )}
            {offline ? t('online.networkOffline') : t('online.realtime')}
          </span>
          {summary.status === 'finished' && (
            <span className="rounded-full border border-[#815f1e]/40 bg-[#f8e8ae]/60 px-2.5 py-1 text-xs font-semibold text-[#6d5118]">
              {t('online.finished')}
            </span>
          )}
        </div>
      </div>

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
      {summary.winner && (
        <div className="rounded-xl border border-[#815f1e]/50 bg-[#f8e8ae]/60 px-4 py-3 text-center text-sm font-semibold text-[#6d5118]">
          {t('online.victory')}:{' '}
          {ownerName(summary.winner, state, summary.players, summary.winner)}
        </div>
      )}

      <main className="flex flex-col gap-4 lg:flex-row">
        <section className="relative h-[560px] min-h-0 flex-1 overflow-hidden rounded-2xl border border-[#b7a786] bg-[#e6d8bb] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)] lg:h-[calc(100vh-19rem)]">
          <div className="pointer-events-none absolute left-5 top-5 z-10">
            <p className="text-[10px] font-semibold uppercase tracking-[0.22em] text-[#806f57]">
              {t('app.mapPublic')}
            </p>
            <p className="mt-1 text-xs text-[#594b3c]">{t('app.mapInstructions')}</p>
          </div>
            <MapViewer
              map={map}
              state={state}
              supply={selectedSupplyLine}
              onSelect={setSelectedId}
              intentions={intentions}
              showIntentions={showIntentions}
              intentionsColor={intentionsColor}
              showRegions={showRegions}
            />
        </section>

        <aside className="w-full shrink-0 space-y-4 lg:w-96 xl:w-[27rem]">
          <Card className="border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
            <CardHeader className="border-b border-[#b7a786]/50 pb-4">
              <CardTitle className="font-serif text-xl text-[#30291f]">
                {activePanel === 'command'
                  ? t('app.commandPost')
                  : activePanel === 'report'
                    ? t('app.turnReport')
                    : t('app.rules')}
              </CardTitle>
              <CardDescription className="text-[#806f57]">
                {currentSlot
                  ? `${currentSlot.name} · ${t('online.you')}`
                  : t('online.accessRevoked')}
              </CardDescription>
              <div
                role="tablist"
                aria-label={t('app.panelViews')}
                className="mt-3 grid grid-cols-3 gap-1 rounded-lg bg-[#f3ead9] p-1"
              >
                {(['command', 'report', 'rules'] as const).map((panel) => (
                  <button
                    key={panel}
                    type="button"
                    role="tab"
                    aria-selected={activePanel === panel}
                    aria-controls={`${panel}-panel`}
                    tabIndex={activePanel === panel ? 0 : -1}
                    data-online-panel-tab={panel}
                    className={`rounded-md px-2 py-2 text-xs font-semibold transition ${activePanel === panel ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
                    onClick={() => setActivePanel(panel)}
                    onKeyDown={(event) =>
                      panelKeyDown(event, activePanel, setActivePanel)
                    }
                  >
                    {panel === 'rules' && (
                      <BookOpen aria-hidden="true" className="mr-1 inline size-3.5" />
                    )}
                    {panel === 'command'
                      ? t('app.commandPost')
                      : panel === 'report'
                        ? `${t('app.turnReport')} ${report ? `· ${report.header.turn}` : ''}`
                        : t('app.rules')}
                  </button>
                ))}
              </div>
            </CardHeader>
            <CardContent className="min-w-0 space-y-5 pt-5">
              <div
                id="command-panel"
                role="tabpanel"
                aria-label={t('app.commandPost')}
                hidden={activePanel !== 'command'}
                className="space-y-5"
              >
                <SelectedTerritoryDetails
                  state={state}
                  selectedTerritory={selectedTerritory}
                  selectedState={selectedState}
                  preferredPlayers={summary.players}
                  selectedSupplyLine={selectedSupplyLine}
                  sourceTerritory={sourceTerritory}
                  supplyLoading={supplyLoading}
                  supplyError={supplyError}
                />
                {playerID ? (
                  <OrdersPanel
                    state={state}
                    player={playerID}
                    chainDrafts={chainDrafts}
                    winterDraft={winterDraft}
                    specialDraft={specialDraft}
                    submitted={Boolean(currentSlot?.submitted)}
                    submitting={submitting}
                    error={actionError}
                    onChainChange={(noble, text) =>
                      setChainDrafts((current) => ({ ...current, [noble]: text }))
                    }
                    onWinterChange={setWinterDraft}
                    onSpecialChange={setSpecialDraft}
                    onSubmit={() => void submitOrders()}
                    onOpenRules={openRules}
                  />
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
              </div>
              <div
                id="report-panel"
                role="tabpanel"
                aria-label={t('app.turnReport')}
                hidden={activePanel !== 'report'}
                className="space-y-4"
              >
                <ReportPane
                  report={report}
                  map={map}
                  players={state.players}
                  summaries={reportSummaries}
                  loading={reportLoading}
                  error={reportError}
                  onSelectReport={(index) => void loadReport(index)}
                />
              </div>
              <div
                id="rules-panel"
                role="tabpanel"
                aria-label={t('app.rules')}
                hidden={activePanel !== 'rules'}
              >
                <RulesPanel
                  gameId={gameId}
                  tokenProvider={tokenProvider}
                  targetSection={rulesNavigation?.section}
                  navigationKey={rulesNavigation?.key}
                />
              </div>
            </CardContent>
          </Card>
          <MapLegend
            showIntentions={showIntentions}
            onToggleIntentions={setShowIntentions}
            showRegions={showRegions}
            onToggleRegions={setShowRegions}
          />
        </aside>
      </main>
      <Lobby
        summary={summary}
        uid={user?.uid ?? ''}
        currentPlayer={summary.currentPlayer}
        invitation={invitation}
        onInvite={() => void createInvitation()}
        inviting={inviting}
        scores={state.scores ?? summary.scores}
      />
    </div>
  )
}
