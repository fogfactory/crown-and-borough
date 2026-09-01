import { useEffect, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { BookOpen } from 'lucide-react'

import { MapLegend } from '@/components/MapLegend'
import { MapViewer } from '@/components/MapViewer'
import { SelectedTerritoryDetails } from '@/components/SelectedTerritoryDetails'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { OrdersPanel } from '@/components/OrdersPanel'
import { ReportPanel } from '@/components/ReportPanel'
import { Scoreboard } from '@/components/Scoreboard'
import { RulesPanel, type RulesSection } from '@/components/RulesPanel'
import { addNobleHeader, hasChainContent } from '@/lib/order-text'
import { hasSupplySource } from '@/lib/supply'
import { VersionBadge } from '@/components/VersionBadge'
import { LanguageProvider, useLanguage } from '@/i18n/LanguageContext'
import { firebaseConfigured } from '@/lib/firebase'
import { OnlineApp } from '@/online/OnlineApp'
import type { Language, MessageKey, Translate } from '@/i18n/messages'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
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
  Season,
  StateData,
  SupplyLine,
  TurnReport,
  OrdersResponse,
} from '@/types'

const PANEL_ORDER = ['command', 'report', 'rules'] as const
type Panel = (typeof PANEL_ORDER)[number]

const MIN_PLAYERS = 2
const MAX_PLAYERS = 16
const PLAYER_COUNT_OPTIONS = Array.from(
  { length: MAX_PLAYERS - MIN_PLAYERS + 1 },
  (_, index) => MIN_PLAYERS + index,
)

const SEASON_KEYS: Record<Season, MessageKey> = {
  spring: 'season.spring',
  summer: 'season.summer',
  autumn: 'season.autumn',
  winter: 'season.winter',
}

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
  const [report, setReport] = useState<TurnReport | null>(null)
  const [supplyLine, setSupplyLine] = useState<SupplyLine | null>(null)
  const [supplyLoading, setSupplyLoading] = useState(false)
  const [supplyError, setSupplyError] = useState<string | null>(null)
  const [chainDrafts, setChainDrafts] = useState<
    Record<PlayerId, Record<string, string>>
  >({})
  const [winterDrafts, setWinterDrafts] = useState<Record<PlayerId, string>>({})
  const [submittedPlayers, setSubmittedPlayers] = useState<PlayerId[]>([])
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [createError, setCreateError] = useState<string | null>(null)
  const [resolving, setResolving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [playerCount, setPlayerCount] = useState(4)
  const [years, setYears] = useState(10)
  const [seed, setSeed] = useState('')
  const [activePanel, setActivePanel] = useState<Panel>('command')
  const [viewedReportTurn, setViewedReportTurn] = useState<number | null>(null)
  const [rulesNavigation, setRulesNavigation] = useState<{
    section: RulesSection
    key: number
  } | null>(null)

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

  const updateChainDraft = (noble: string, text: string) => {
    setChainDrafts((drafts) => ({
      ...drafts,
      [selectedPlayer]: { ...(drafts[selectedPlayer] ?? {}), [noble]: text },
    }))
  }

  const updateWinterDraft = (text: string) => {
    setWinterDrafts((drafts) => ({ ...drafts, [selectedPlayer]: text }))
  }

  const openRules = (section: RulesSection) => {
    setRulesNavigation((current) => ({
      section,
      key: (current?.key ?? 0) + 1,
    }))
    setActivePanel('rules')
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
    const winter =
      state.season === 'winter' && (winterDrafts[selectedPlayer] ?? '').trim() !== ''
        ? [{ player: selectedPlayer, lines: winterDrafts[selectedPlayer] ?? '' }]
        : []

    try {
      const response = await fetch(
        `/api/orders?lang=${language}&player=${encodeURIComponent(selectedPlayer)}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ player: selectedPlayer, chains, winter, force }),
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
    <div lang={language} className="min-h-screen bg-[#efe7d8] text-[#30291f]">
      <header className="border-b border-[#b7a786]/60 bg-[#fffaf0]/90 px-4 py-4 shadow-sm backdrop-blur-sm sm:px-6">
        <div className="mx-auto flex max-w-[1800px] flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-full border-2 border-[#a84632] bg-[#f6dfc6] font-serif text-sm font-bold text-[#a84632] shadow-inner">
              C&amp;B
            </div>
            <div>
              <h1 className="font-serif text-xl font-semibold tracking-tight sm:text-2xl">
                Crown &amp; Borough
              </h1>
              <p className="text-xs uppercase tracking-[0.18em] text-[#806f57]">
                {t('app.tagline')}
              </p>
              <VersionBadge />
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-2 sm:gap-x-4">
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
            {createError && (
              <p
                role="alert"
                className="w-full rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-2 py-1 text-xs text-[#8d321e]"
              >
                {createError}
              </p>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-3 sm:gap-5">
            <div className="border-l border-[#b7a786]/60 pl-3 sm:pl-5">
              <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-[#806f57]">
                {t('app.season')}
              </p>
              <p className="font-serif text-base font-semibold">
                {state
                  ? t('app.turn', {
                      turn: state.turn,
                      season: t(SEASON_KEYS[state.season]),
                    })
                  : t('app.loading')}
              </p>
              {state && (
                <>
                  <p className="text-xs font-semibold uppercase tracking-[0.12em] text-[#a84632]">
                    {t('app.year', { year: 1000 + internalYear(state) })}
                  </p>
                  <p className="text-[11px] text-[#806f57]">
                    {t('app.remainingYearsTurns', {
                      years: remainingYears(state),
                      turns: remainingTurns(state),
                    })}
                  </p>
                </>
              )}
            </div>
            <div className="flex items-center gap-2">
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
              <label htmlFor="player-view" className="text-xs font-medium text-[#806f57]">
                {t('app.activePlayer')}
              </label>
              <Select value={selectedPlayer} onValueChange={setSelectedPlayer}>
                <SelectTrigger
                  id="player-view"
                  className="w-[172px] border-[#b7a786] bg-[#fffaf0] text-[#30291f]"
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
              <span className="rounded-full border border-[#376341]/30 bg-[#e8f1e3] px-2 py-1 text-[10px] font-semibold uppercase tracking-[0.12em] text-[#376341]">
                {t('app.privateView')}
              </span>
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
            </div>
            <LanguageSwitcher />
          </div>
        </div>
      </header>

      {state?.finished && (
        <div className="mx-auto mt-4 max-w-[1800px] rounded-xl border border-[#815f1e]/50 bg-[#f8e8ae]/60 px-4 py-3 text-center text-sm font-semibold text-[#6d5118] sm:mx-6">
          {state.winner
            ? `${t('online.victory')}: ${ownerLabel(state.winner, state, t)}`
            : t('app.finished')}
        </div>
      )}

      <main className="mx-auto flex min-h-[calc(100vh-6.5rem)] max-w-[1800px] flex-col gap-4 p-4 sm:p-6 lg:flex-row">
        <section className="relative h-[620px] min-h-0 flex-1 overflow-hidden rounded-2xl border border-[#b7a786] bg-[#e6d8bb] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)] lg:h-[calc(100vh-8.5rem)]">
          <div className="pointer-events-none absolute left-5 top-5 z-10">
            <p className="text-[10px] font-semibold uppercase tracking-[0.22em] text-[#806f57]">
              {t('app.mapPublic')}
            </p>
            <p className="mt-1 text-xs text-[#594b3c]">{t('app.mapInstructions')}</p>
          </div>
          {loadError ? (
            <div
              role="alert"
              className="flex h-full items-center justify-center px-6 text-center"
            >
              <p className="font-serif text-lg text-[#a84632]">
                {t('app.mapLoadFailed', { message: loadError })}
              </p>
            </div>
          ) : map && state ? (
            <MapViewer
              map={map}
              state={state}
              supply={selectedSupplyLine}
              onSelect={setSelectedId}
            />
          ) : (
            <div className="flex h-full items-center justify-center px-6 text-center">
              <p className="font-serif text-lg italic text-[#806f57]">
                {t('app.mapLoading')}
              </p>
            </div>
          )}
        </section>

        <aside className="w-full shrink-0 space-y-4 lg:w-96 xl:w-[27rem]">
          <Scoreboard players={state?.players ?? []} scores={state?.scores} />
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
                {t('app.selectedPlayer', { player: selectedPlayer })}
              </CardDescription>
              <div
                role="tablist"
                aria-label={t('app.panelViews')}
                className="mt-3 grid grid-cols-3 gap-1 rounded-lg bg-[#f3ead9] p-1"
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={activePanel === 'command'}
                  aria-controls="command-panel"
                  tabIndex={activePanel === 'command' ? 0 : -1}
                  data-panel-tab="command"
                  className={`rounded-md px-2 py-2 text-xs font-semibold transition ${activePanel === 'command' ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
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
                  className={`rounded-md px-2 py-2 text-xs font-semibold transition ${activePanel === 'report' ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
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
                  className={`rounded-md px-2 py-2 text-xs font-semibold transition ${activePanel === 'rules' ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
                  onClick={() => setActivePanel('rules')}
                  onKeyDown={handlePanelKeyDown}
                >
                  <span className="inline-flex items-center gap-1.5">
                    <BookOpen aria-hidden="true" className="size-3.5" />
                    {t('app.rules')}
                  </span>
                </button>
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
                  selectedSupplyLine={selectedSupplyLine}
                  sourceTerritory={supplySourceTerritory}
                  supplyLoading={supplyLoading}
                  supplyError={supplyError}
                />

                {state && (
                  <OrdersPanel
                    state={state}
                    player={selectedPlayer}
                    chainDrafts={chainDrafts[selectedPlayer] ?? {}}
                    winterDraft={winterDrafts[selectedPlayer] ?? ''}
                    submitted={submittedPlayers.includes(selectedPlayer)}
                    submitting={resolving}
                    error={actionError}
                    onChainChange={updateChainDraft}
                    onWinterChange={updateWinterDraft}
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
                {report ? (
                  <ReportPanel report={report} map={map} players={state?.players ?? []} />
                ) : (
                  <div className="flex min-h-36 items-center justify-center rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] px-6 text-center">
                    <p className="font-serif text-lg italic text-[#806f57]">
                      {t('app.noReport')}
                    </p>
                  </div>
                )}
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
          <MapLegend />
        </aside>
      </main>
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
