import { useEffect, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { BookOpen } from 'lucide-react'

import { MapLegend } from '@/components/MapLegend'
import { MapViewer } from '@/components/MapViewer'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { OrdersPanel } from '@/components/OrdersPanel'
import { ReportPanel } from '@/components/ReportPanel'
import { RulesPanel, type RulesSection } from '@/components/RulesPanel'
import { formatOrderLabel } from '@/lib/order-label'
import { hasSupplySource } from '@/lib/supply'
import { LanguageProvider, useLanguage } from '@/i18n/LanguageContext'
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
  InfraType,
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

const TERRAIN_KEYS: Record<MapData['territories'][number]['terrain'], MessageKey> = {
  plain: 'terrain.plain',
  forest: 'terrain.forest',
  hill: 'terrain.hill',
  mountain: 'terrain.mountain',
  swamp: 'terrain.swamp',
}

const INFRASTRUCTURE_KEYS: Record<InfraType, MessageKey> = {
  mill: 'infrastructure.mill',
  supply_depot: 'infrastructure.supply_depot',
  castle: 'infrastructure.castle',
  village: 'infrastructure.village',
}

function ownerLabel(
  owner: PlayerId | null,
  state: StateData | null,
  t: Translate,
): string {
  if (!owner) return t('app.noOwner')
  return state?.players.find((player) => player.id === owner)?.name ?? owner
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function addNobleHeader(nobleCode: string, text: string): string {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const firstContentIndex = lines.findIndex((line) => line.split('#', 1)[0].trim() !== '')
  if (firstContentIndex >= 0) {
    const headerPattern = new RegExp(`^${escapeRegExp(nobleCode)}(?:\\s+#.*)?$`, 'i')
    if (headerPattern.test(lines[firstContentIndex].trim())) {
      lines.splice(firstContentIndex, 1)
    }
  }
  return `${nobleCode}\n${lines.join('\n')}`.trimEnd()
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
  const selectedCapitalPlayer = state?.players.find(
    (player) => player.capitalTerritory === selectedId,
  )
  const selectedChain = selectedState?.army?.chain ?? null
  const presentNobles =
    state?.nobles.filter((noble) => noble.location === selectedId) ?? []
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
            .filter(
              (submission) =>
                submission.text
                  .replace(new RegExp(`^${escapeRegExp(submission.noble)}\\s*`), '')
                  .trim() !== '',
            )
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
        body: JSON.stringify({ seed: seed.trim(), players: playerCount }),
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
                {selectedTerritory ? (
                  <div className="space-y-5">
                    <div>
                      <p className="text-xs font-bold uppercase tracking-[0.2em] text-[#a84632]">
                        {selectedTerritory.id}
                      </p>
                      <h2 className="mt-1 font-serif text-2xl font-semibold leading-tight">
                        {selectedTerritory.name}
                      </h2>
                      {selectedCapitalPlayer && (
                        <p className="mt-2 inline-flex items-center rounded-full border border-[#815f1e]/40 bg-[#f8e8ae]/60 px-2.5 py-1 text-xs font-semibold text-[#6d5118]">
                          {t('app.capitalOf', { player: selectedCapitalPlayer.name })}
                        </p>
                      )}
                    </div>
                    <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm">
                      <dt className="text-[#806f57]">{t('app.terrain')}</dt>
                      <dd className="font-medium">
                        {t(TERRAIN_KEYS[selectedTerritory.terrain])}
                      </dd>
                      {selectedState && (
                        <>
                          <dt className="text-[#806f57]">{t('app.control')}</dt>
                          <dd className="font-medium">
                            {ownerLabel(selectedState.owner, state, t)}
                          </dd>
                          <dt className="text-[#806f57]">{t('app.resources')}</dt>
                          <dd className="font-medium">{selectedState.resources} R</dd>
                        </>
                      )}
                    </dl>

                    <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                      <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
                        {t('app.noblesPresent')}
                      </h3>
                      {presentNobles.length > 0 ? (
                        <ul className="space-y-1.5 text-sm">
                          {presentNobles.map((noble) => {
                            const owner = state?.players.find(
                              (player) => player.id === noble.owner,
                            )
                            const holder =
                              noble.status !== 'free'
                                ? (selectedState?.army?.owner ?? null)
                                : null
                            return (
                              <li
                                key={noble.id}
                                className="rounded-md bg-[#f3ead9] px-3 py-2 text-sm"
                              >
                                <div className="flex items-center justify-between gap-3">
                                  <span className="min-w-0">
                                    <span
                                      className="mr-2 inline-block size-3 shrink-0 rounded-full border border-[#30291f]/30 align-[-1px]"
                                      style={{
                                        backgroundColor: owner?.color ?? '#b7a786',
                                      }}
                                      aria-label={t('app.colorOf', {
                                        player: ownerLabel(noble.owner, state, t),
                                      })}
                                    />
                                    <strong>{noble.code}</strong> · {noble.name}
                                  </span>
                                  <span
                                    className={`shrink-0 text-xs ${noble.status === 'dungeon' ? 'text-[#a84632]' : 'text-[#376341]'}`}
                                  >
                                    {t(
                                      `orders.nobleStatus.${noble.status}` as MessageKey,
                                    )}
                                  </span>
                                </div>
                                <dl className="mt-1 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 border-t border-[#b7a786]/40 pt-1 text-xs text-[#806f57]">
                                  <dt>{t('app.owner')}</dt>
                                  <dd className="font-medium text-[#594b3c]">
                                    {ownerLabel(noble.owner, state, t)}
                                  </dd>
                                  {holder && (
                                    <>
                                      <dt>{t('app.holder')}</dt>
                                      <dd className="font-medium text-[#594b3c]">
                                        {noble.status === 'hostage'
                                          ? t('app.hostageBy', {
                                              player: ownerLabel(holder, state, t),
                                            })
                                          : t('app.dungeonBy', {
                                              player: ownerLabel(holder, state, t),
                                            })}
                                      </dd>
                                    </>
                                  )}
                                </dl>
                              </li>
                            )
                          })}
                        </ul>
                      ) : (
                        <p className="text-sm italic text-[#806f57]">
                          {t('app.noNoble')}
                        </p>
                      )}
                    </div>

                    {selectedState && (
                      <>
                        <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                          <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
                            {t('app.army')}
                          </h3>
                          {selectedState.army ? (
                            <div className="rounded-md bg-[#f3ead9] px-3 py-2 text-sm">
                              <div className="flex items-center justify-between gap-3">
                                <span className="font-semibold">
                                  {ownerLabel(selectedState.army.owner, state, t)}
                                </span>
                                <span className="shrink-0 text-xs text-[#806f57]">
                                  {t(
                                    selectedState.army.size === 1
                                      ? 'app.troop'
                                      : 'app.troops',
                                    { count: selectedState.army.size },
                                  )}
                                </span>
                              </div>
                              <div className="mt-2 border-t border-[#b7a786]/40 pt-2 text-xs text-[#806f57]">
                                {selectedChain?.visibility === 'hidden' ? (
                                  <p className="rounded-md border border-[#b7a786]/50 bg-[#fffaf0] px-2 py-1.5 italic">
                                    {t('app.hiddenChain')}
                                  </p>
                                ) : selectedChain ? (
                                  <>
                                    <p>
                                      {t('app.nobleEmitter')}:{' '}
                                      <strong>{selectedChain.noble}</strong>
                                    </p>
                                    <p>
                                      {t('app.currentIndex')}:{' '}
                                      <strong>
                                        {(selectedChain.currentIndex ?? 0) <
                                        (selectedChain.orders?.length ?? 0)
                                          ? (selectedChain.currentIndex ?? 0) + 1
                                          : t('app.finished')}
                                      </strong>
                                    </p>
                                    <div className="mt-2 space-y-1.5">
                                      <p className="font-semibold uppercase tracking-[0.12em] text-[#806f57]">
                                        {t('app.orderStack')}
                                      </p>
                                      <ol className="space-y-1.5">
                                        {(selectedChain.orders ?? []).map(
                                          (order, index) => {
                                            const current =
                                              index === (selectedChain.currentIndex ?? 0)
                                            return (
                                              <li
                                                key={`${order.type}-${order.position}-${index}`}
                                                aria-current={
                                                  current ? 'step' : undefined
                                                }
                                                className={`rounded-md border px-2 py-1.5 ${current ? 'border-[#a84632]/60 bg-[#f8e5dd] text-[#8d321e]' : 'border-[#b7a786]/40 bg-[#fffaf0]'}`}
                                              >
                                                <span className="mr-1.5 font-semibold">
                                                  {index + 1}.
                                                </span>
                                                <span>{formatOrderLabel(order)}</span>
                                                {current && (
                                                  <span className="ml-1.5 font-semibold">
                                                    · {t('app.current')}
                                                  </span>
                                                )}
                                              </li>
                                            )
                                          },
                                        )}
                                      </ol>
                                    </div>
                                  </>
                                ) : (
                                  <p>{t('app.noOrders')}</p>
                                )}
                              </div>
                            </div>
                          ) : (
                            <p className="text-sm italic text-[#806f57]">
                              {t('app.noArmy')}
                            </p>
                          )}
                          {selectedState.army && state?.season === 'winter' && (
                            <p className="rounded-md border border-[#b7a786]/50 bg-[#fffaf0] px-3 py-2 text-xs text-[#806f57]">
                              {t('app.noSupplyWinter')}
                            </p>
                          )}
                          {selectedState.army && state?.season !== 'winter' && (
                            <div className="rounded-md border border-[#b7a786]/50 bg-[#fffaf0] px-3 py-2 text-xs text-[#594b3c]">
                              <p className="font-semibold uppercase tracking-[0.12em] text-[#806f57]">
                                {t('app.supply')}
                              </p>
                              {supplyLoading ? (
                                <p className="mt-1 italic text-[#806f57]">
                                  {t('app.supplyCalculating')}
                                </p>
                              ) : supplyError ? (
                                <p className="mt-1 text-[#8d321e]">{supplyError}</p>
                              ) : selectedSupplyLine?.selfSupplied ? (
                                <p className="mt-1 text-[#376341]">
                                  {t('app.localRationsSufficient')}
                                </p>
                              ) : selectedSupplyLine?.source ? (
                                <>
                                  <p className="mt-1">
                                    {t('app.sourceLabel')}{' '}
                                    <strong>
                                      {supplySourceTerritory?.id ??
                                        selectedSupplyLine.source}
                                      {supplySourceTerritory
                                        ? ` · ${supplySourceTerritory.name}`
                                        : ''}
                                    </strong>
                                  </p>
                                  <p className="mt-1">
                                    {t(
                                      selectedSupplyLine.distance > 1
                                        ? 'app.distances'
                                        : 'app.distance',
                                      { distance: selectedSupplyLine.distance },
                                    )}
                                  </p>
                                </>
                              ) : selectedSupplyLine ? (
                                <p className="mt-1 text-[#8d321e]">
                                  {t('app.noAccessibleSource')}
                                </p>
                              ) : null}
                              {selectedSupplyLine && (
                                <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 border-t border-[#b7a786]/40 pt-2">
                                  <dt className="text-[#806f57]">
                                    {t('app.localRations')}
                                  </dt>
                                  <dd className="font-medium">
                                    {selectedSupplyLine.rations}
                                  </dd>
                                  <dt className="text-[#806f57]">
                                    {t('app.demandToCover')}
                                  </dt>
                                  <dd className="font-medium">
                                    {selectedSupplyLine.demand}
                                  </dd>
                                </dl>
                              )}
                            </div>
                          )}
                          {!selectedState.army && hasSupplySource(selectedState) && (
                            <div className="rounded-md border border-[#b7a786]/50 bg-[#fffaf0] px-3 py-2 text-xs text-[#594b3c]">
                              <p className="font-semibold uppercase tracking-[0.12em] text-[#806f57]">
                                {t('app.supplyZone')}
                              </p>
                              {state?.season === 'winter' ? (
                                <p className="mt-1 italic text-[#806f57]">
                                  {t('app.noSupplyWinter')}
                                </p>
                              ) : supplyLoading ? (
                                <p className="mt-1 italic text-[#806f57]">
                                  {t('app.supplyCalculating')}
                                </p>
                              ) : supplyError ? (
                                <p className="mt-1 text-[#8d321e]">{supplyError}</p>
                              ) : selectedSupplyLine ? (
                                <p className="mt-1 text-[#376341]">
                                  {t(
                                    selectedSupplyLine.reachable.length > 1
                                      ? 'app.reachablePlural'
                                      : 'app.reachable',
                                    { count: selectedSupplyLine.reachable.length },
                                  )}
                                </p>
                              ) : null}
                            </div>
                          )}
                        </div>

                        <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                          <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
                            {t('app.infrastructure')}
                          </h3>
                          {selectedState.infrastructures.length > 0 ? (
                            <ul className="space-y-1.5 text-sm">
                              {selectedState.infrastructures.map(
                                (infrastructure, index) => (
                                  <li
                                    key={`${infrastructure.type}-${index}`}
                                    className="flex items-center justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
                                  >
                                    <span className="flex min-w-0 items-center gap-2 font-medium">
                                      <span>
                                        {t(INFRASTRUCTURE_KEYS[infrastructure.type])}
                                      </span>
                                      {infrastructure.type === 'castle' &&
                                        selectedCapitalPlayer && (
                                          <span className="shrink-0 rounded-full bg-[#f8e8ae] px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-[#6d5118]">
                                            {t('app.capital')}
                                          </span>
                                        )}
                                    </span>
                                    <span className="shrink-0 text-xs text-[#806f57]">
                                      {t('app.level', { level: infrastructure.level })}
                                    </span>
                                  </li>
                                ),
                              )}
                            </ul>
                          ) : (
                            <p className="text-sm italic text-[#806f57]">
                              {t('app.noInfrastructure')}
                            </p>
                          )}
                        </div>
                      </>
                    )}
                  </div>
                ) : (
                  <div className="flex min-h-36 items-center justify-center rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] px-6 text-center">
                    <p className="font-serif text-lg italic text-[#806f57]">
                      {t('app.selectTerritory')}
                    </p>
                  </div>
                )}

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
      <AppContent />
    </LanguageProvider>
  )
}

export default App
