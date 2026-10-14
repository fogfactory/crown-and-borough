import { useCallback, useEffect, useMemo, useState } from 'react'
import { IconTrophy } from '@tabler/icons-react'

import { GameLayout } from '@/components/GameLayout'
import { BrandMark } from '@/components/BrandMark'
import { GameMenu } from '@/components/GameMenu'
import { MapViewer } from '@/components/MapViewer'
import { ownTerritoriesFrame } from '@/lib/map-frame'
import { SelectedTerritoryDetails } from '@/components/SelectedTerritoryDetails'
import { OrdersPanel, SealOrders } from '@/components/OrdersPanel'
import { RealmPanel } from '@/components/RealmPanel'
import { ProjectedConsumptionSummary } from '@/components/ProjectedConsumptionSummary'
import { ProjectedIncomeSummary } from '@/components/ProjectedIncomeSummary'
import { ReportPane } from '@/components/ReportPane'
import { LineageDialog } from '@/components/LineageDialog'
import { Scoreboard } from '@/components/Scoreboard'
import {
  SimulationRejectedError,
  type VictorySimulationRequest,
} from '@/lib/use-victory-simulation'
import { SubmissionDots } from '@/components/SubmissionDots'
import { RulesPanel, type RulesSection } from '@/components/RulesPanel'
import { InfoPage } from '@/components/InfoPage'
import { GameDrawer, type Panel } from '@/components/GameDrawer'
import { useIsDesktop } from '@/lib/use-media-query'
import { buildOrdersBody } from '@/lib/orders-body'
import { draftOrdersByNoble } from '@/lib/transfer-preview'
import { useOrdersPreview, type OrdersPreviewRequest } from '@/lib/use-orders-preview'
import { internalYear, ownerName, totalTurns } from '@/lib/game-progress'
import { useGameIntentions } from '@/lib/use-game-intentions'
import { useSupplyAndTransfer } from '@/lib/use-supply-and-transfer'
import { SEASON_LABEL_KEYS, SEASON_TINT_BG } from '@/lib/season'
import { useLocalStorageState, useLocalStorageText } from '@/lib/storage'
import { LanguageProvider, useLanguage } from '@/i18n/LanguageContext'
import { firebaseConfigured } from '@/lib/firebase'
import { OnlineApp } from '@/online/OnlineApp'
import type { Language, Translate } from '@/i18n/messages'
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
  VictorySimulation,
  TurnReport,
  OrdersResponse,
  OrdersPreview,
} from '@/types'
import { DEFAULT_PLAYER_COLOR } from '@/lib/map-palette'

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
  const desktop = useIsDesktop()
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
  const [requiredPlayers, setRequiredPlayers] = useState<PlayerId[]>([])
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [createError, setCreateError] = useState<string | null>(null)
  const [resolving, setResolving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [playerCount, setPlayerCount] = useState(4)
  const [years, setYears] = useState(10)
  const [seed, setSeed] = useState('')
  const [view, setView] = useState<HotseatView>('game')
  const [activePanel, setActivePanel] = useState<Panel>(desktop ? 'orders' : 'map')
  const [showOwnership, setShowOwnership] = useLocalStorageState(
    'cb.ownershipOverlay',
    true,
  )
  const [showRegions, setShowRegions] = useLocalStorageState('cb.regionsOverlay', true)
  const [viewedReportTurn, setViewedReportTurn] = useState<number | null>(null)
  const [lineageOpen, setLineageOpen] = useState(false)
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
    const loadRequiredPlayers = async () => {
      try {
        const response = await fetch(asPlayer(hotseatGamePath(gameId), HOTSEAT_HOST), {
          signal: controller.signal,
        })
        if (!response.ok) return
        const detail = (await response.json()) as {
          players: Array<{ id: PlayerId; required: boolean }>
        }
        if (!controller.signal.aborted) {
          setRequiredPlayers(
            detail.players.filter((player) => player.required).map((player) => player.id),
          )
        }
      } catch {
        // A stale required-players roster only affects the submission dots;
        // the next turn change retries this fetch.
      }
    }
    void loadRequiredPlayers()
    return () => controller.abort()
    // Refetched once per turn: required-ness only changes when the turn
    // advances (elimination, a captured noble reaching the dungeon, etc.),
    // never while players are still submitting for the current one.
  }, [gameId, gameReady, state?.turn])

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

  const simulationRequest = useMemo<VictorySimulationRequest | null>(() => {
    if (!gameId) return null
    const path = asPlayer(`${hotseatGamePath(gameId)}/victory/simulate`, selectedPlayer)
    return async (actions, signal) => {
      const response = await fetch(path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ actions }),
        signal,
      })
      if (response.status === 422) {
        const payload = (await response.json().catch(() => null)) as {
          message?: string
        } | null
        throw new SimulationRejectedError(payload?.message ?? 'rejected')
      }
      if (!response.ok) throw new Error(`simulation failed (${response.status})`)
      return (await response.json()) as VictorySimulation
    }
  }, [gameId, selectedPlayer])

  const supplyFetcher = useCallback(
    async <T,>(path: string, signal: AbortSignal): Promise<T> => {
      const response = await fetch(asPlayer(path, selectedPlayer), { signal })
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
    [selectedPlayer, t],
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
    specialDraft: specialDrafts[selectedPlayer] ?? '',
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
  }

  const mapFrame =
    desktop || !state
      ? undefined
      : ownTerritoriesFrame(state, selectedPlayer, `${gameId}:${selectedPlayer}`)
  // On the phone map view the details show in a card over the map instead.
  const detailsOnMap = !desktop && activePanel === 'map' && Boolean(selectedTerritory)

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
          <p className="font-display text-lg text-brass-700">
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
          selectedId={selectedId}
          frame={mapFrame}
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
        <p className="font-display text-lg italic text-ink-500">{t('app.mapLoading')}</p>
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
        setRequiredPlayers([])
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
      setRequiredPlayers([])
      setSelectedId(null)
      setSelectedPlayer(HOTSEAT_HOST)
      setView('game')
      setActivePanel(desktop ? 'orders' : 'map')
    } catch (error) {
      setCreateError(
        error instanceof Error ? error.message : t('error.gameCreationFailed'),
      )
    } finally {
      setCreating(false)
    }
  }

  const territoryDetails = (
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
  )

  const playerPicker = (
    <div className="flex items-center gap-1.5">
      <span
        role="img"
        className="size-3 shrink-0 rounded-full border border-ink-900/30 shadow-inner"
        style={{
          backgroundColor:
            state?.players.find((player) => player.id === selectedPlayer)?.color ??
            DEFAULT_PLAYER_COLOR,
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
          className="h-8 w-28 border-line bg-paper-50 text-ink-900 sm:w-40"
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
  )

  const hotseatNav = (['game', 'rules', 'faq'] as const).map((nextView) => (
    <button
      key={nextView}
      type="button"
      aria-current={view === nextView ? 'page' : undefined}
      className={`rounded-md px-2 py-1 text-sm font-semibold transition ${view === nextView ? 'bg-paper-100 text-brass-700' : 'text-ink-500 hover:bg-paper-100 hover:text-ink-900'}`}
      onClick={() => setView(nextView)}
    >
      {t(`nav.${nextView}`)}
    </button>
  ))

  return (
    <div
      lang={language}
      className={`flex flex-col bg-paper-200 text-ink-900 ${
        view === 'game' ? 'h-dvh overflow-hidden' : 'min-h-screen'
      }`}
    >
      <header className="z-30 shrink-0 bg-paper-50">
        <div
          className={`flex items-center gap-2 px-3 sm:gap-3 sm:px-4 ${desktop ? 'h-12' : 'h-14'}`}
        >
          <div className="flex shrink-0 items-center gap-2">
            <BrandMark className="size-8" />
            <h1 className="hidden font-display text-base font-semibold tracking-tight lg:block">
              Crown &amp; Borough
            </h1>
          </div>
          <p
            data-game-progress
            className="min-w-0 truncate text-sm font-semibold text-ink-900"
          >
            {state
              ? t(desktop ? 'app.progress' : 'app.progressShort', {
                  season: t(SEASON_LABEL_KEYS[state.season]),
                  year: 1000 + internalYear(state),
                  turn: state.turn,
                  total: totalTurns(state),
                })
              : t('app.loading')}
          </p>

          <div className="ml-auto flex shrink-0 items-center gap-2">
            {state && (
              <>
                <HeaderPopover
                  label={t('app.scores')}
                  icon={<IconTrophy aria-hidden="true" className="size-4" />}
                  hint={String(
                    Math.max(
                      0,
                      ...Object.values(state.scores ?? {}).map(
                        (score) => score.total ?? 0,
                      ),
                    ),
                  )}
                >
                  <Scoreboard
                    players={state.players}
                    scores={state.scores}
                    victory={state.victory}
                    finished={state.finished}
                    winners={state.winners}
                    minorWinner={state.minorWinner}
                  />
                </HeaderPopover>
                {/* On phones the lineage opens from the Realm tab only. */}
                <span className={desktop ? 'contents' : 'hidden'}>
                  <LineageDialog
                    open={lineageOpen}
                    onOpenChange={setLineageOpen}
                    players={state.players}
                    nobles={state.nobles}
                    deceased={state.deceased}
                    claims={state.claims}
                    defaultFocus={selectedPlayer}
                    fiefs={state.fiefs}
                    bishoprics={state.bishoprics}
                    cardinals={state.cardinals}
                    pope={state.pope}
                    king={state.king}
                    queen={state.queen}
                    queenDowager={state.queenDowager}
                    excommunicated={state.excommunicated}
                    marriages={state.marriages}
                    scores={state.scores}
                    victory={state.victory}
                    simulationRequest={simulationRequest}
                    simulationKey={`${state.turn}-${state.season}`}
                  />
                </span>
              </>
            )}
            {desktop && (
              <SubmissionDots
                players={(state?.players ?? []).map((player) => ({
                  id: player.id,
                  name: player.name || player.id,
                  color: player.color,
                  submitted: submittedPlayers.includes(player.id),
                  required: requiredPlayers.includes(player.id),
                  isYou: player.id === selectedPlayer,
                }))}
              />
            )}
            {desktop && playerPicker}
            {desktop && (
              <nav aria-label={t('nav.primary')} className="flex items-center gap-1">
                {hotseatNav}
              </nav>
            )}
            <GameMenu
              language
              compact={
                <>
                  <nav aria-label={t('nav.primary')} className="flex gap-1">
                    {hotseatNav}
                  </nav>
                  {playerPicker}
                </>
              }
              resolution={{
                disabled: resolving || !state,
                onResolve: () => submitOrders(true),
              }}
              creation={{
                playerCount,
                years,
                seed,
                creating,
                error: createError,
                canCreate: Boolean(state) && seed.trim() !== '',
                onPlayerCountChange: setPlayerCount,
                onYearsChange: setYears,
                onSeedChange: setSeed,
                onCreate: startNewGame,
              }}
            />
          </div>
        </div>
        {state ? (
          <div
            aria-hidden="true"
            data-season-rule={state.season}
            className={`h-[3px] ${SEASON_TINT_BG[state.season]}`}
          />
        ) : (
          <div aria-hidden="true" className="h-px bg-line" />
        )}
      </header>

      {view === 'game' && state?.finished && (
        <div className="shrink-0 border-b border-brass-700/50 bg-brass-300/30 px-4 py-1.5 text-center text-sm font-semibold text-brass-700">
          {state.winner
            ? `${t('online.victory')}: ${ownerLabel(state.winner, state, t)}`
            : t('app.finished')}
        </div>
      )}

      {view === 'game' ? (
        <main className="flex min-h-0 w-full flex-1 flex-col">
          <GameLayout
            map={renderMap()}
            view={activePanel}
            mapOverlay={detailsOnMap ? territoryDetails : undefined}
            onCloseOverlay={() => setSelectedId(null)}
          >
            <GameDrawer
              activePanel={activePanel}
              onPanelChange={setActivePanel}
              season={state?.season}
              reportLabelExtra={
                report && viewedReportTurn !== report.header.turn ? (
                  <span className="rounded-full bg-paper-50 px-1.5 py-0.5 text-2xs text-brass-700">
                    {t('app.reportNew')}
                  </span>
                ) : null
              }
              footer={
                state ? (
                  <SealOrders
                    state={state}
                    player={selectedPlayer}
                    winterDraft={winterDrafts[selectedPlayer] ?? ''}
                    specialDraft={specialDrafts[selectedPlayer] ?? ''}
                    submitted={submittedPlayers.includes(selectedPlayer)}
                    submitting={resolving}
                    error={actionError}
                    onSubmit={() => void submitOrders()}
                  />
                ) : null
              }
              panels={{
                orders: (
                  <>
                    <ProjectedIncomeSummary
                      state={state}
                      playerId={selectedPlayer}
                      specialDraft={specialDrafts[selectedPlayer]}
                    />
                    <ProjectedConsumptionSummary
                      state={state}
                      playerId={selectedPlayer}
                    />
                    {!detailsOnMap && territoryDetails}
                    {state && (
                      <OrdersPanel
                        regions={map?.regions}
                        mapTerritories={map?.territories}
                        state={state}
                        player={selectedPlayer}
                        chainDrafts={chainDrafts[selectedPlayer] ?? {}}
                        winterDraft={winterDrafts[selectedPlayer] ?? ''}
                        preview={preview}
                        specialDraft={specialDrafts[selectedPlayer] ?? ''}
                        onChainChange={updateChainDraft}
                        onWinterChange={updateWinterDraft}
                        onSpecialChange={updateSpecialDraft}
                        onOpenRules={openRules}
                      />
                    )}
                  </>
                ),
                report: (
                  <ReportPane report={report} map={map} players={state?.players ?? []} />
                ),
                realm: state ? (
                  <RealmPanel
                    state={state}
                    territories={map?.territories}
                    onOpenLineage={() => setLineageOpen(true)}
                  />
                ) : null,
                rules: (
                  <RulesPanel
                    targetSection={rulesNavigation?.section}
                    navigationKey={rulesNavigation?.key}
                  />
                ),
              }}
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
