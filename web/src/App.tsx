import {
  useEffect,
  useMemo,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from 'react'
import { IconBook, IconTrophy } from '@tabler/icons-react'

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
import { addNobleHeader, hasChainContent } from '@/lib/order-text'
import { buildIntentions } from '@/lib/intent-overlay'
import { hasSupplySource } from '@/lib/supply'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { useLocalStorageState } from '@/lib/storage'
import { transferTargetsForTerritory } from '@/lib/transfer-preview'
import { isWinterCosts } from '@/lib/winter-cost'
import { VersionBadge } from '@/components/VersionBadge'
import { LanguageProvider, useLanguage } from '@/i18n/LanguageContext'
import { firebaseConfigured } from '@/lib/firebase'
import { OnlineApp } from '@/online/OnlineApp'
import type { Language, Translate } from '@/i18n/messages'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
  WinterCosts,
} from '@/types'

const PANEL_ORDER = ['command', 'report', 'rules'] as const
type Panel = (typeof PANEL_ORDER)[number]
type HotseatView = 'game' | 'rules' | 'faq'

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
  return state?.players.find((player) => player.id === owner)?.name ?? owner
}

function internalYear(state: StateData): number {
  const year = state.year ?? Math.floor((state.turn - 1) / 4) + 1
  if (state.finished && state.yearCount && state.turn > state.yearCount * 4) {
    return state.yearCount
  }
  return year
}

function remainingYears(state: StateData): number {
  if (state.finished) return 0
  const yearCount = state.yearCount ?? 10
  return Math.max(0, yearCount - internalYear(state) + 1)
}

function remainingTurns(state: StateData): number {
  const yearCount = state.yearCount ?? 10
  return Math.max(0, yearCount * 4 - state.turn + 1)
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
  const [map, setMap] = useState<MapData | null>(null)
  const [state, setState] = useState<StateData | null>(null)
  const [winterCosts, setWinterCosts] = useState<WinterCosts | null>(null)
  const [report, setReport] = useState<TurnReport | null>(null)
  const [supplyLine, setSupplyLine] = useState<SupplyLine | null>(null)
  const [supplyLoading, setSupplyLoading] = useState(false)
  const [supplyError, setSupplyError] = useState<string | null>(null)
  const [transferLine, setTransferLine] = useState<TransferLine | null>(null)
  const [transferLoading, setTransferLoading] = useState(false)
  const [transferError, setTransferError] = useState<string | null>(null)
  const [selectedTransferTarget, setSelectedTransferTarget] = useState<string | null>(
    null,
  )
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
  const [showRegions, setShowRegions] = useState(false)
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

  useEffect(() => {
    const controller = new AbortController()
    const loadMap = async () => {
      try {
        const mapResponse = await fetch('/api/map', { signal: controller.signal })
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
  }, [t])

  useEffect(() => {
    const controller = new AbortController()
    const loadWinterCosts = async () => {
      try {
        const response = await fetch('/api/balance', { signal: controller.signal })
        if (!response.ok) return
        const payload: unknown = await response.json()
        if (!controller.signal.aborted && isWinterCosts(payload)) {
          setWinterCosts(payload)
        }
      } catch {
        // The cost preview is optional; game loading should remain available.
      }
    }
    void loadWinterCosts()
    return () => controller.abort()
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    const loadPrivateState = async () => {
      try {
        const response = await fetch(
          `/api/state?player=${encodeURIComponent(selectedPlayer)}`,
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
  }, [selectedPlayer, t])

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
  const supplySelectionAllowed =
    (supplyLine?.kind === 'army' && Boolean(selectedState?.army)) ||
    (supplyLine?.kind === 'source' &&
      !selectedState?.army &&
      hasSupplySource(selectedState))
  const selectedSupplyLine =
    supplySelectionAllowed && supplyLine?.territory === selectedId ? supplyLine : null
  const supplySourceTerritory = map?.territories.find(
    (territory) => territory.id === selectedSupplyLine?.source,
  )
  const transferTargets =
    selectedState?.army?.owner === selectedPlayer
      ? transferTargetsForTerritory(chainDrafts[selectedPlayer] ?? {}, selectedId)
      : []
  const transferTarget = transferTargets.includes(selectedTransferTarget ?? '')
    ? selectedTransferTarget
    : (transferTargets[0] ?? null)
  const intentions = useMemo(
    () =>
      state && map
        ? buildIntentions(map, state, selectedPlayer, chainDrafts[selectedPlayer] ?? {})
        : [],
    [chainDrafts, map, selectedPlayer, state],
  )
  const intentionsColor =
    state?.players.find((player) => player.id === selectedPlayer)?.color ?? '#a84632'

  useEffect(() => {
    const controller = new AbortController()
    const armySelected = Boolean(selectedState?.army)
    const sourceSelected = hasSupplySource(selectedState)

    setSupplyLine(null)
    setSupplyError(null)
    if (
      !state ||
      !selectedId ||
      (!armySelected && !sourceSelected) ||
      state.season === 'winter'
    ) {
      setSupplyLoading(false)
      return () => controller.abort()
    }

    setSupplyLoading(true)
    const loadSupplyLine = async () => {
      try {
        const response = await fetch(
          `/api/supply?territory=${encodeURIComponent(selectedId)}`,
          { signal: controller.signal },
        )
        if (!response.ok) {
          throw new Error(`${t('error.requestFailed', { status: response.status })}`)
        }
        const payload = (await response.json()) as SupplyLine
        if (!Array.isArray(payload.path) || !Array.isArray(payload.reachable)) {
          throw new Error('Invalid supply response')
        }
        if (!controller.signal.aborted) {
          setSupplyLine(payload)
          setSupplyLoading(false)
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setSupplyError(
            error instanceof Error
              ? error.message
              : t('error.requestFailed', { status: 500 }),
          )
          setSupplyLoading(false)
        }
      }
    }

    void loadSupplyLine()
    return () => controller.abort()
  }, [selectedId, selectedState, state, t])

  useEffect(() => {
    const controller = new AbortController()
    setTransferLine(null)
    setTransferError(null)
    if (
      !state ||
      !selectedId ||
      !selectedState?.army ||
      !transferTarget ||
      state.season === 'winter'
    ) {
      setTransferLoading(false)
      return () => controller.abort()
    }

    setTransferLoading(true)
    const loadTransferLine = async () => {
      try {
        const response = await fetch(
          `/api/supply?territory=${encodeURIComponent(selectedId)}&target=${encodeURIComponent(transferTarget)}`,
          { signal: controller.signal },
        )
        if (!response.ok) {
          throw new Error(`${t('error.requestFailed', { status: response.status })}`)
        }
        const payload = (await response.json()) as TransferLine
        if (
          payload.kind !== 'transfer' ||
          !Array.isArray(payload.path) ||
          !Array.isArray(payload.reachableTerritories)
        ) {
          throw new Error('Invalid transfer response')
        }
        if (!controller.signal.aborted) {
          setTransferLine(payload)
          setTransferLoading(false)
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setTransferError(
            error instanceof Error
              ? error.message
              : t('error.requestFailed', { status: 500 }),
          )
          setTransferLoading(false)
        }
      }
    }

    void loadTransferLine()
    return () => controller.abort()
  }, [selectedId, selectedState, state, t, transferTarget])

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
          showIntentions={showIntentions}
          intentionsColor={intentionsColor}
          onToggleIntentions={setShowIntentions}
          showRegions={showRegions}
          onToggleRegions={setShowRegions}
        />
      )
    }
    return (
      <div className="flex h-full items-center justify-center px-6 text-center">
        <p className="font-serif text-lg italic text-[#806f57]">{t('app.mapLoading')}</p>
      </div>
    )
  }

  const handlePanelKeyDown = (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    const currentIndex = PANEL_ORDER.indexOf(activePanel)
    let nextIndex: number | null = null
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
      nextIndex = (currentIndex + 1) % PANEL_ORDER.length
    } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
      nextIndex = (currentIndex - 1 + PANEL_ORDER.length) % PANEL_ORDER.length
    } else if (event.key === 'Home') {
      nextIndex = 0
    } else if (event.key === 'End') {
      nextIndex = PANEL_ORDER.length - 1
    }
    if (nextIndex === null) return

    event.preventDefault()
    const nextPanel = PANEL_ORDER[nextIndex]
    setActivePanel(nextPanel)
    event.currentTarget.parentElement
      ?.querySelector<HTMLButtonElement>(`[data-panel-tab="${nextPanel}"]`)
      ?.focus()
  }

  const submitOrders = async (force = false) => {
    if (!state) return
    setResolving(true)
    setActionError(null)
    const chains =
      state.season === 'winter'
        ? []
        : state.nobles
            .filter(
              (noble) => noble.owner === selectedPlayer && noble.status !== 'dungeon',
            )
            .map((noble) => ({
              player: selectedPlayer,
              noble: noble.code,
              text: addNobleHeader(
                noble.code,
                chainDrafts[selectedPlayer]?.[noble.code] ?? '',
              ),
            }))
            .filter((submission) => hasChainContent(submission.noble, submission.text))
    const winterLines = [
      winterDrafts[selectedPlayer] ?? '',
      state.season === 'winter' ? (specialDrafts[selectedPlayer] ?? '') : '',
    ]
      .filter((text) => text.trim() !== '')
      .join('\n')
    const winter =
      state.season === 'winter' && winterLines !== ''
        ? [{ player: selectedPlayer, lines: winterLines }]
        : []
    const special =
      state.season !== 'winter' && (specialDrafts[selectedPlayer] ?? '').trim() !== ''
        ? [{ player: selectedPlayer, text: specialDrafts[selectedPlayer] ?? '' }]
        : []

    try {
      const response = await fetch(
        `/api/orders?lang=${language}&player=${encodeURIComponent(selectedPlayer)}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            player: selectedPlayer,
            chains,
            winter,
            special,
            force,
          }),
        },
      )
      if (!response.ok) throw new Error(await responseError(response, t))
      const payload = (await response.json()) as OrdersResponse
      setState(payload.state)
      setSubmittedPlayers(payload.submitted)
      if (payload.status === 'resolved' && payload.report) {
        setReport(payload.report)
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
      const response = await fetch('/api/game', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ seed: seed.trim(), players: playerCount, years }),
      })
      if (!response.ok) throw new Error(await responseError(response, t))
      const payload = (await response.json()) as { map: MapData; state: StateData }
      setMap(payload.map)
      setState(payload.state)
      setReport(null)
      setChainDrafts({})
      setWinterDrafts({})
      setSubmittedPlayers([])
      setSelectedId(null)
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
            <Card className="border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
              <CardHeader className="border-b border-[#b7a786]/50 pb-3">
                <CardTitle className="font-serif text-lg text-[#30291f] sm:text-xl">
                  {activePanel === 'command'
                    ? t('app.commandPost')
                    : activePanel === 'report'
                      ? t('app.turnReport')
                      : t('app.rules')}
                </CardTitle>
                <CardDescription className="text-[#806f57]">
                  {t('app.selectedPlayer', { player: selectedPlayer })}
                </CardDescription>
                <div
                  role="tablist"
                  aria-label={t('app.panelViews')}
                  className="mt-2 grid grid-cols-3 gap-1 rounded-lg bg-[#f3ead9] p-1"
                >
                  <button
                    type="button"
                    role="tab"
                    aria-selected={activePanel === 'command'}
                    aria-controls="command-panel"
                    tabIndex={activePanel === 'command' ? 0 : -1}
                    data-panel-tab="command"
                    className={`rounded-md px-2 py-1.5 text-xs font-semibold transition ${activePanel === 'command' ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
                    onClick={() => setActivePanel('command')}
                    onKeyDown={handlePanelKeyDown}
                  >
                    {t('app.commandPost')}
                  </button>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={activePanel === 'report'}
                    aria-controls="report-panel"
                    tabIndex={activePanel === 'report' ? 0 : -1}
                    data-panel-tab="report"
                    className={`rounded-md px-2 py-1.5 text-xs font-semibold transition ${activePanel === 'report' ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
                    onClick={() => setActivePanel('report')}
                    onKeyDown={handlePanelKeyDown}
                  >
                    {t('app.turnReport')}{' '}
                    {report && viewedReportTurn !== report.header.turn ? (
                      <span className="ml-1 rounded-full bg-[#a84632] px-1.5 py-0.5 text-[10px] text-[#fffaf0]">
                        {t('app.reportNew')}
                      </span>
                    ) : null}
                  </button>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={activePanel === 'rules'}
                    aria-controls="rules-panel"
                    tabIndex={activePanel === 'rules' ? 0 : -1}
                    data-panel-tab="rules"
                    className={`rounded-md px-2 py-1.5 text-xs font-semibold transition ${activePanel === 'rules' ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
                    onClick={() => setActivePanel('rules')}
                    onKeyDown={handlePanelKeyDown}
                  >
                    <span className="inline-flex items-center gap-1.5">
                      <IconBook aria-hidden="true" className="size-3.5" />
                      {t('app.rules')}
                    </span>
                  </button>
                </div>
              </CardHeader>
              <CardContent className="min-w-0 space-y-4 pt-4">
                <div
                  id="command-panel"
                  role="tabpanel"
                  aria-label={t('app.commandPost')}
                  hidden={activePanel !== 'command'}
                  className="space-y-4"
                >
                  <SelectedTerritoryDetails
                    state={state}
                    selectedTerritory={selectedTerritory}
                    selectedState={selectedState}
                    mapTerritories={map?.territories ?? []}
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
                      winterCosts={winterCosts}
                      map={map ?? undefined}
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
                </div>
                <div
                  id="report-panel"
                  role="tabpanel"
                  aria-label={t('app.turnReport')}
                  hidden={activePanel !== 'report'}
                  className="min-w-0"
                >
                  <ReportPane report={report} map={map} players={state?.players ?? []} />
                </div>
                <div
                  id="rules-panel"
                  role="tabpanel"
                  aria-label={t('app.rules')}
                  hidden={activePanel !== 'rules'}
                  className="min-w-0"
                >
                  <RulesPanel
                    targetSection={rulesNavigation?.section}
                    navigationKey={rulesNavigation?.key}
                  />
                </div>
              </CardContent>
            </Card>
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
