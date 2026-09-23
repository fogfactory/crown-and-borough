import { useCallback, useEffect, useMemo, useState } from 'react'
import { IconTrophy } from '@tabler/icons-react'

import { GameLayout } from '@/components/GameLayout'
import { BrandMark } from '@/components/BrandMark'
import { GameSetupMenu } from '@/components/GameSetupMenu'
import { MapViewer } from '@/components/MapViewer'
import { SelectedTerritoryDetails } from '@/components/SelectedTerritoryDetails'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { OrdersPanel } from '@/components/OrdersPanel'
import { ReportPane } from '@/components/ReportPane'
import { Scoreboard } from '@/components/Scoreboard'
import { SubmissionDots } from '@/components/SubmissionDots'
import { RulesPanel, type RulesSection } from '@/components/RulesPanel'
import { InfoPage } from '@/components/InfoPage'
import { GamePanelCard } from '@/components/GamePanelCard'
import type { Panel } from '@/components/CommandReportRulesTabs'
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
import { useLocalStorageState, useLocalStorageText } from '@/lib/storage'
import { VersionBadge } from '@/components/VersionBadge'
import { LanguageProvider, useLanguage } from '@/i18n/LanguageContext'
import { firebaseConfigured } from '@/lib/firebase'
import { OnlineApp } from '@/online/OnlineApp'
import type { Language, Translate } from '@/i18n/messages'
import { Button } from '@/components/ui/button'
import { HeaderPopover } from '@/components/ui/header-popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type {
  MapData,
  PlayerId,
  StateData,
  SupplyLine,
  TransferLine,
  TurnReport,
  OrdersResponse,
  OrdersPreview,
} from '@/types'

type HotseatView = 'game' | 'rules' | 'faq'

/**
 * The hotseat runs on the development games API: the server trusts the
 * `player` query parameter, and HOTSEAT_HOST creates games and forces their
 * resolution.
 */
const HOTSEAT_HOST: PlayerId = 'P1'

function hotseatGamePath(gameId: string): string {
  return `/api/games/${encodeURIComponent(gameId)}`
}

function asPlayer(path: string, player: PlayerId, language?: Language): string {
  const query = new URLSearchParams({ player })
  if (language) query.set('lang', language)
  return `${path}${path.includes('?') ? '&' : '?'}${query.toString()}`
}

function postJSON(path: string, body: unknown): Promise<Response> {
  return fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

const MIN_PLAYERS = 2
const MAX_PLAYERS = 16
const PLAYER_COUNT_OPTIONS = Array.from(
  { length: MAX_PLAYERS - MIN_PLAYERS + 1 },
  (_, index) => MIN_PLAYERS + index,
)

function ownerLabel(
  owner: PlayerId | null,
  state: StateData | null,
  t: Translate,
): string {
  if (!owner) return t('app.noOwner')
  return state ? ownerName(owner, state) : owner
}

async function responseError(response: Response, t: Translate): Promise<string> {
  const payload = (await response.json().catch(() => null)) as {
    message?: string
    errors?: Array<{ line?: number; message?: string }>
  } | null
  const first = payload?.errors?.[0]
  if (first?.line) {
    return t('error.line', {
      line: first.line,
      message: first.message ?? t('error.invalidOrder'),
    })
  }
  return payload?.message ?? t('error.requestFailed', { status: response.status })
}

function AppContent() {
  const { language, t } = useLanguage()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [selectedPlayer, setSelectedPlayer] = useState<PlayerId>('P1')
  const [gameId, setGameId] = useLocalStorageText('cb.hotseatGame')
  const [gameReady, setGameReady] = useState(false)
  const [map, setMap] = useState<MapData | null>(null)
  const [state, setState] = useState<StateData | null>(null)
  const [report, setReport] = useState<TurnReport | null>(null)

  const [chainDrafts, setChainDrafts] = useState<
    Record<PlayerId, Record<string, string>>
  >({})
  const [winterDrafts, setWinterDrafts] = useState<Record<PlayerId, string>>({})
  const [specialDrafts, setSpecialDrafts] = useState<Record<PlayerId, string>>({})
  const [submittedPlayers, setSubmittedPlayers] = useState<PlayerId[]>([])
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [createError, setCreateError] = useState<string | null>(null)
  const [resolving, setResolving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [playerCount, setPlayerCount] = useState(4)
  const [years, setYears] = useState(10)
  const [seed, setSeed] = useState('')
  const [view, setView] = useState<HotseatView>('game')
  const [activePanel, setActivePanel] = useState<Panel>('command')
  const [showOwnership, setShowOwnership] = useLocalStorageState(
    'cb.ownershipOverlay',
    true,
  )
  const [showRegions, setShowRegions] = useLocalStorageState('cb.regionsOverlay', true)
  const [viewedReportTurn, setViewedReportTurn] = useState<number | null>(null)
  const [mapFocusSignal, setMapFocusSignal] = useState(0)
  const [rulesNavigation, setRulesNavigation] = useState<{
    section: RulesSection
    key: number
  } | null>(null)
  const [showIntentions, setShowIntentions] = useLocalStorageState(
    'cb.intentionsOverlay',
    true,
  )
  const [showCalamities, setShowCalamities] = useLocalStorageState(
    'cb.calamitiesOverlay',
    true,
  )
  const [showCards, setShowCards] = useLocalStorageState('cb.cardsOverlay', true)

  useEffect(() => {
    const controller = new AbortController()
    const selectGame = async () => {
      try {
        const response = await fetch(asPlayer('/api/games', HOTSEAT_HOST), {
          signal: controller.signal,
        })
        if (!response.ok) {
          throw new Error(`${t('error.loadGameFailed')} (${response.status})`)
        }
        const games = (await response.json()) as Array<{ id: string }>
        if (controller.signal.aborted) return
        if (!games.some((game) => game.id === gameId)) {
          const fallback = games[0]?.id ?? null
          if (fallback === null) {
            throw new Error(t('error.loadGameFailed'))
          }
          setGameId(fallback)
        }
        setGameReady(true)
      } catch (error) {
        if (!controller.signal.aborted) {
          setLoadError(error instanceof Error ? error.message : t('error.loadGameFailed'))
        }
      }
    }
    void selectGame()
    return () => controller.abort()
    // The remembered game is only validated once, when the hotseat opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [t])

  useEffect(() => {
    if (!gameReady || !gameId) return
    const controller = new AbortController()
    const loadMap = async () => {
      try {
        const mapResponse = await fetch(
          asPlayer(`${hotseatGamePath(gameId)}/map`, HOTSEAT_HOST),
          { signal: controller.signal },
        )
        if (!mapResponse.ok) {
          throw new Error(`${t('error.loadGameFailed')} (${mapResponse.status})`)
        }
        const mapData = (await mapResponse.json()) as MapData
        if (!controller.signal.aborted) {
          setMap(mapData)
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setLoadError(error instanceof Error ? error.message : t('error.loadGameFailed'))
        }
      }
    }
    void loadMap()
    return () => controller.abort()
  }, [gameId, gameReady, t])

  useEffect(() => {
    if (!gameReady || !gameId) return
    const controller = new AbortController()
    const loadPrivateState = async () => {
      try {
        const response = await fetch(
          asPlayer(`${hotseatGamePath(gameId)}/state`, selectedPlayer),
          { signal: controller.signal },
        )
        if (!response.ok) {
          throw new Error(`${t('error.loadGameFailed')} (${response.status})`)
        }
        const stateData = (await response.json()) as StateData
        if (!controller.signal.aborted) {
          setState(stateData)
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setLoadError(error instanceof Error ? error.message : t('error.loadGameFailed'))
        }
      }
    }
    setReport(null)
    void loadPrivateState()
    return () => controller.abort()
  }, [gameId, gameReady, selectedPlayer, t])

  useEffect(() => {
    if (state && !state.players.some((player) => player.id === selectedPlayer)) {
      setSelectedPlayer(state.players[0]?.id ?? 'P1')
    }
  }, [selectedPlayer, state])

  useEffect(() => {
    if (activePanel === 'report' && report) {
      setViewedReportTurn(report.header.turn)
    }
  }, [activePanel, report])

  const selectedTerritory = map?.territories.find(
    (territory) => territory.id === selectedId,
  )
  const selectedState = state?.territories.find(
    (territory) => territory.id === selectedId,
  )
  const selectedRegion = map?.regions?.find((region) =>
    region.territories.includes(selectedId ?? ''),
  )

  const supplyFetcher = useCallback(
    async <T,>(path: string, signal: AbortSignal): Promise<T> => {
      const response = await fetch(path, { signal })
      if (!response.ok) {
        throw new Error(t('error.requestFailed', { status: response.status }))
      }
      const payload = (await response.json()) as T
      if (path.includes('target=')) {
        const transfer = payload as unknown as TransferLine
        if (
          transfer.kind !== 'transfer' ||
          !Array.isArray(transfer.path) ||
          !Array.isArray(transfer.reachableTerritories)
        ) {
          throw new Error('Invalid transfer response')
        }
      } else {
        const supply = payload as unknown as SupplyLine
        if (!Array.isArray(supply.path) || !Array.isArray(supply.reachable)) {
          throw new Error('Invalid supply response')
        }
      }
      return payload
    },
    [t],
  )

  const ordersBody = useMemo(
    () =>
      state
        ? buildOrdersBody(state, selectedPlayer, {
            chainDrafts: chainDrafts[selectedPlayer] ?? {},
            winterDraft: winterDrafts[selectedPlayer] ?? '',
            specialDraft: specialDrafts[selectedPlayer] ?? '',
          })
        : null,
    [chainDrafts, selectedPlayer, specialDrafts, state, winterDrafts],
  )
  const previewRequest = useMemo<OrdersPreviewRequest | null>(() => {
    if (!gameId) return null
    const path = asPlayer(
      `${hotseatGamePath(gameId)}/orders/preview`,
      selectedPlayer,
      language,
    )
    return async (body, signal) => {
      const response = await fetch(path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal,
      })
      if (!response.ok)
        throw new Error(t('error.requestFailed', { status: response.status }))
      return (await response.json()) as OrdersPreview
    }
  }, [gameId, language, selectedPlayer, t])
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
    ownerId: selectedPlayer,
    basePath: gameId ? hotseatGamePath(gameId) : '/api',
    fetcher: supplyFetcher,
    networkErrorMessage: t('error.requestFailed', { status: 500 }),
  })
  const supplySourceTerritory = sourceTerritoryId
    ? (map?.territories.find((territory) => territory.id === sourceTerritoryId) ?? null)
    : null

  const { intentions, winterIntentions, intentionsColor } = useGameIntentions({
    state,
    map,
    playerID: selectedPlayer,
    preview,
    winterText: ordersBody?.winter[0]?.lines ?? '',
    spectator: false,
  })
  /**
   * Card intentions are private: only the drafting player's own map shows
   * them, so the hotseat view exposes the selected player's draft only.
   */
  const specialOrders = useMemo(() => {
    const text = specialDrafts[selectedPlayer] ?? ''
    return text.trim() !== '' ? [{ player: selectedPlayer, text }] : []
  }, [specialDrafts, selectedPlayer])

  const updateChainDraft = (noble: string, text: string) => {
    setChainDrafts((drafts) => ({
      ...drafts,
      [selectedPlayer]: { ...(drafts[selectedPlayer] ?? {}), [noble]: text },
    }))
  }

  const handleTerritorySelect = (id: string | null) => {
    setSelectedId(id)
    if (id) {
      setMapFocusSignal((signal) => signal + 1)
    }
  }

  const updateWinterDraft = (text: string) => {
    setWinterDrafts((drafts) => ({ ...drafts, [selectedPlayer]: text }))
  }

  const updateSpecialDraft = (text: string) => {
    setSpecialDrafts((drafts) => ({ ...drafts, [selectedPlayer]: text }))
  }

  const openRules = (section: RulesSection) => {
    setRulesNavigation((current) => ({
      section,
      key: (current?.key ?? 0) + 1,
    }))
    setActivePanel('rules')
  }

  const renderMap = () => {
    if (loadError) {
      return (
        <div
          role="alert"
          className="flex h-full items-center justify-center px-6 text-center"
        >
          <p className="font-serif text-lg text-[#a84632]">
            {t('app.mapLoadFailed', { message: loadError })}
          </p>
        </div>
      )
    }
    if (map && state) {
      return (
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
          specialOrders={specialOrders}
        />
      )
    }
    return (
      <div className="flex h-full items-center justify-center px-6 text-center">
        <p className="font-serif text-lg italic text-[#806f57]">{t('app.mapLoading')}</p>
      </div>
    )
  }

  /**
   * Reloads the selected player's state and latest report after a forced
   * resolution, which the server answers from the host's point of view.
   */
  const reloadForSelectedPlayer = async (currentGameId: string) => {
    const base = hotseatGamePath(currentGameId)
    const [stateResponse, reportsResponse] = await Promise.all([
      fetch(asPlayer(`${base}/state`, selectedPlayer)),
      fetch(asPlayer(`${base}/reports`, selectedPlayer)),
    ])
    if (!stateResponse.ok) throw new Error(await responseError(stateResponse, t))
    if (!reportsResponse.ok) throw new Error(await responseError(reportsResponse, t))
    const nextState = (await stateResponse.json()) as StateData
    const reports = (await reportsResponse.json()) as Array<{ index: number }>
    const latest = reports.at(-1)
    let nextReport: TurnReport | null = null
    if (latest) {
      const reportResponse = await fetch(
        asPlayer(`${base}/reports/${latest.index}`, selectedPlayer),
      )
      if (!reportResponse.ok) throw new Error(await responseError(reportResponse, t))
      nextReport = (await reportResponse.json()) as TurnReport
    }
    return { state: nextState, report: nextReport }
  }

  const submitOrders = async (force = false) => {
    if (!state || !gameId) return
    setResolving(true)
    setActionError(null)
    const base = hotseatGamePath(gameId)
    try {
      const response = await postJSON(
        asPlayer(`${base}/orders`, selectedPlayer, language),
        buildOrdersBody(state, selectedPlayer, {
          chainDrafts: chainDrafts[selectedPlayer] ?? {},
          winterDraft: winterDrafts[selectedPlayer] ?? '',
          specialDraft: specialDrafts[selectedPlayer] ?? '',
        }),
      )
      if (!response.ok) throw new Error(await responseError(response, t))
      let payload = (await response.json()) as OrdersResponse
      let resolvedReport = payload.report ?? null
      if (force && payload.status === 'pending') {
        const forced = await postJSON(
          asPlayer(`${base}/resolve`, HOTSEAT_HOST, language),
          {},
        )
        if (!forced.ok) throw new Error(await responseError(forced, t))
        payload = (await forced.json()) as OrdersResponse
        const reloaded = await reloadForSelectedPlayer(gameId)
        payload = { ...payload, state: reloaded.state }
        resolvedReport = reloaded.report
      }
      setState(payload.state)
      setSubmittedPlayers(payload.submitted)
      if (payload.status === 'resolved' && resolvedReport) {
        setReport(resolvedReport)
        setActivePanel('report')
        setChainDrafts({})
        setWinterDrafts({})
        setSpecialDrafts({})
        setSubmittedPlayers([])
      }
    } catch (error) {
      setActionError(error instanceof Error ? error.message : t('error.resolutionFailed'))
    } finally {
      setResolving(false)
    }
  }

  const startNewGame = async () => {
    if (!state) return
    setCreating(true)
    setCreateError(null)
    try {
      const response = await postJSON(asPlayer('/api/games', HOTSEAT_HOST, language), {
        name: 'Hotseat',
        seed: seed.trim(),
        players: playerCount,
        years,
      })
      if (!response.ok) throw new Error(await responseError(response, t))
      const created = (await response.json()) as { id: string }
      setMap(null)
      setState(null)
      setGameId(created.id)
      setReport(null)
      setChainDrafts({})
      setWinterDrafts({})
      setSpecialDrafts({})
      setSubmittedPlayers([])
      setSelectedId(null)
      setSelectedPlayer(HOTSEAT_HOST)
      setView('game')
      setActivePanel('command')
    } catch (error) {
      setCreateError(
        error instanceof Error ? error.message : t('error.gameCreationFailed'),
      )
    } finally {
      setCreating(false)
    }
  }

  return (
    <div
      lang={language}
      className={`flex flex-col bg-[#efe7d8] text-[#30291f] ${
        view === 'game' ? 'h-dvh overflow-hidden' : 'min-h-screen'
      }`}
    >
      <header className="z-30 shrink-0 border-b border-[#b7a786]/60 bg-[#fffaf0]/95 px-3 py-2 shadow-sm backdrop-blur-sm sm:px-6">
        <div className="mx-auto flex max-w-[1800px] flex-wrap items-center gap-x-3 gap-y-2">
          <div className="flex items-center gap-2.5">
            <BrandMark className="size-9 sm:size-11" />
            <div className="min-w-0">
              <h1 className="truncate font-serif text-base font-semibold tracking-tight sm:text-xl">
                Crown &amp; Borough
              </h1>
              <p className="hidden text-[10px] uppercase tracking-[0.18em] text-[#806f57] min-[420px]:block">
                {t('app.tagline')}
              </p>
            </div>
            <VersionBadge />
          </div>

          <nav
            aria-label={t('nav.primary')}
            className="order-3 flex w-full items-center justify-center gap-1 min-[420px]:order-none min-[420px]:w-auto"
          >
            {(['game', 'rules', 'faq'] as const).map((nextView) => (
              <button
                key={nextView}
                type="button"
                aria-current={view === nextView ? 'page' : undefined}
                className={`rounded-md px-2.5 py-1.5 text-sm font-semibold transition ${view === nextView ? 'bg-[#f3ead9] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:bg-[#f3ead9] hover:text-[#30291f]'}`}
                onClick={() => setView(nextView)}
              >
                {t(`nav.${nextView}`)}
              </button>
            ))}
          </nav>

          <div className="ml-auto flex items-center gap-2 sm:gap-3">
            <div className="hidden text-right min-[420px]:block">
              <p className="text-xs font-semibold leading-tight">
                {state
                  ? t('app.turn', {
                      turn: state.turn,
                      season: t(SEASON_LABEL_KEYS[state.season]),
                    })
                  : t('app.loading')}
              </p>
              {state && (
                <p className="text-[10px] leading-tight text-[#806f57]">
                  {t('app.year', { year: 1000 + internalYear(state) })} ·{' '}
                  {t('app.remainingYearsTurns', {
                    years: remainingYears(state),
                    turns: remainingTurns(state),
                  })}
                </p>
              )}
            </div>
            {state && (
              <HeaderPopover
                label={t('app.scores')}
                icon={<IconTrophy aria-hidden="true" className="size-4" />}
                hint={String(
                  Math.max(
                    0,
                    ...Object.values(state.scores ?? {}).map((score) => score.total ?? 0),
                  ),
                )}
              >
                <Scoreboard players={state.players} scores={state.scores} />
              </HeaderPopover>
            )}
            <SubmissionDots
              players={(state?.players ?? []).map((player) => ({
                id: player.id,
                name: player.name || player.id,
                color: player.color,
                submitted: submittedPlayers.includes(player.id),
                isYou: player.id === selectedPlayer,
              }))}
            />
            <div className="flex items-center gap-1.5">
              <span
                role="img"
                className="size-3 shrink-0 rounded-full border border-[#30291f]/30 shadow-inner"
                style={{
                  backgroundColor:
                    state?.players.find((player) => player.id === selectedPlayer)
                      ?.color ?? '#b7a786',
                }}
                aria-label={t('app.colorOf', {
                  player: ownerLabel(selectedPlayer, state, t),
                })}
              />
              <label htmlFor="player-view" className="sr-only">
                {t('app.activePlayer')}
              </label>
              <Select value={selectedPlayer} onValueChange={setSelectedPlayer}>
                <SelectTrigger
                  id="player-view"
                  className="w-32 border-[#b7a786] bg-[#fffaf0] text-[#30291f] sm:w-44"
                >
                  <SelectValue placeholder={t('app.choosePlayer')} />
                </SelectTrigger>
                <SelectContent>
                  {state?.players.map((player) => (
                    <SelectItem key={player.id} value={player.id}>
                      {player.id} · {player.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={resolving || !state}
              title={t('app.resolveTitle')}
              onClick={() => void submitOrders(true)}
            >
              {t('app.resolve')}
            </Button>
            <LanguageSwitcher />

            <div className="hidden items-end gap-3 md:flex">
              <div>
                <label
                  htmlFor="player-count"
                  className="block text-[10px] font-semibold uppercase tracking-[0.2em] text-[#806f57]"
                >
                  {t('app.players')}
                </label>
                <Select
                  value={String(playerCount)}
                  onValueChange={(value) => setPlayerCount(Number(value))}
                >
                  <SelectTrigger
                    id="player-count"
                    className="w-[76px] border-[#b7a786] bg-[#fffaf0] text-[#30291f]"
                  >
                    <SelectValue placeholder={t('app.players')} />
                  </SelectTrigger>
                  <SelectContent>
                    {PLAYER_COUNT_OPTIONS.map((count) => (
                      <SelectItem key={count} value={String(count)}>
                        {count}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label
                  htmlFor="game-seed"
                  className="block text-[10px] font-semibold uppercase tracking-[0.2em] text-[#806f57]"
                >
                  {t('app.seed')}
                </label>
                <input
                  id="game-seed"
                  type="text"
                  value={seed}
                  onChange={(event) => setSeed(event.target.value)}
                  placeholder={t('app.seedPlaceholder')}
                  className="h-8 w-44 rounded-lg border border-[#b7a786] bg-[#fffaf0] px-2.5 text-sm text-[#30291f] outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20 sm:w-52"
                />
              </div>
              <div>
                <label
                  htmlFor="game-years"
                  className="block text-[10px] font-semibold uppercase tracking-[0.2em] text-[#806f57]"
                >
                  {t('home.gameYears')}
                </label>
                <input
                  id="game-years"
                  type="number"
                  min={1}
                  max={50}
                  value={years}
                  onChange={(event) => setYears(Number(event.target.value))}
                  className="h-8 w-20 rounded-lg border border-[#b7a786] bg-[#fffaf0] px-2.5 text-sm text-[#30291f] outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20"
                />
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={creating || !state || seed.trim() === ''}
                title={t('app.newGameTitle')}
                onClick={() => void startNewGame()}
              >
                {creating ? t('app.creating') : t('app.newGame')}
              </Button>
            </div>
            <div className="md:hidden">
              <GameSetupMenu
                playerCount={playerCount}
                years={years}
                seed={seed}
                creating={creating}
                createError={createError}
                canCreate={Boolean(state) && seed.trim() !== ''}
                onPlayerCountChange={setPlayerCount}
                onYearsChange={setYears}
                onSeedChange={setSeed}
                onCreate={startNewGame}
              />
            </div>
          </div>
        </div>
      </header>

      {view === 'game' && state?.finished && (
        <div className="mx-auto mt-2 w-full max-w-[1800px] shrink-0 rounded-xl border border-[#815f1e]/50 bg-[#f8e8ae]/60 px-4 py-2 text-center text-sm font-semibold text-[#6d5118] sm:mx-6">
          {state.winner
            ? `${t('online.victory')}: ${ownerLabel(state.winner, state, t)}`
            : t('app.finished')}
        </div>
      )}

      {view === 'game' ? (
        <main className="mx-auto flex min-h-0 w-full max-w-[1800px] flex-1 flex-col p-3 sm:p-4 lg:p-6">
          <GameLayout map={renderMap()} focusSignal={mapFocusSignal}>
            <GamePanelCard
              activePanel={activePanel}
              onPanelChange={setActivePanel}
              subtitle={t('app.selectedPlayer', { player: selectedPlayer })}
              reportLabelExtra={
                report && viewedReportTurn !== report.header.turn ? (
                  <span className="ml-1 rounded-full bg-[#a84632] px-1.5 py-0.5 text-[10px] text-[#fffaf0]">
                    {t('app.reportNew')}
                  </span>
                ) : null
              }
              command={
                <>
                  <SelectedTerritoryDetails
                    state={state}
                    selectedTerritory={selectedTerritory}
                    selectedState={selectedState}
                    mapTerritories={map?.territories ?? []}
                    selectedRegion={selectedRegion}
                    selectedSupplyLine={selectedSupplyLine}
                    sourceTerritory={supplySourceTerritory}
                    supplyLoading={supplyLoading}
                    supplyError={supplyError}
                    transferTargets={transferTargets}
                    selectedTransferTarget={transferTarget}
                    onTransferTargetChange={setSelectedTransferTarget}
                    transferLine={transferLine}
                    transferLoading={transferLoading}
                    transferError={transferError}
                  />
                  {state && (
                    <OrdersPanel
                      state={state}
                      player={selectedPlayer}
                      chainDrafts={chainDrafts[selectedPlayer] ?? {}}
                      winterDraft={winterDrafts[selectedPlayer] ?? ''}
                      preview={preview}
                      specialDraft={specialDrafts[selectedPlayer] ?? ''}
                      submitted={submittedPlayers.includes(selectedPlayer)}
                      submitting={resolving}
                      error={actionError}
                      onChainChange={updateChainDraft}
                      onWinterChange={updateWinterDraft}
                      onSpecialChange={updateSpecialDraft}
                      onSubmit={() => void submitOrders()}
                      onOpenRules={openRules}
                    />
                  )}
                </>
              }
              report={
                <ReportPane report={report} map={map} players={state?.players ?? []} />
              }
              rules={
                <RulesPanel
                  targetSection={rulesNavigation?.section}
                  navigationKey={rulesNavigation?.key}
                />
              }
            />
          </GameLayout>
        </main>
      ) : (
        <main className="mx-auto w-full max-w-[1200px] flex-1 p-4 sm:p-6">
          <InfoPage kind={view} />
        </main>
      )}
    </div>
  )
}

function App({ initialLanguage }: { initialLanguage?: Language }) {
  return (
    <LanguageProvider initialLanguage={initialLanguage}>
      {firebaseConfigured ? <OnlineApp /> : <AppContent />}
    </LanguageProvider>
  )
}

export default App
