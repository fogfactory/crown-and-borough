import { useEffect, useState } from 'react'

import { MapViewer } from '@/components/MapViewer'
import { OrdersPanel } from '@/components/OrdersPanel'
import { ReportPanel } from '@/components/ReportPanel'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
  Order,
  PlayerId,
  Season,
  StateData,
  TurnReport,
  OrdersResponse,
} from '@/types'

const TERRAIN_NAMES = {
  plain: 'Plaine',
  forest: 'Forêt',
  hill: 'Colline',
  mountain: 'Montagne',
  swamp: 'Marécage',
} as const

const SEASON_LABELS: Record<Season, string> = {
  spring: 'Printemps',
  summer: 'Été',
  autumn: 'Automne',
  winter: 'Hiver',
}

const INFRASTRUCTURE_LABELS: Record<InfraType, string> = {
  mill: 'Moulin',
  post_relay: 'Relais de poste',
  watchtower: 'Tour de guet',
  supply_depot: 'Dépôt de vivres',
  castle: 'Château',
  village: 'Village',
}

function ownerLabel(owner: PlayerId | null, state: StateData | null): string {
  if (!owner) return 'Personne'
  return state?.players.find((player) => player.id === owner)?.name ?? `Joueur ${owner}`
}

function orderLabel(order: Order): string {
  const targets = order.targets?.join(' → ')
  return [order.type, order.position, targets].filter(Boolean).join(' ')
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

async function responseError(response: Response): Promise<string> {
  const payload = (await response.json().catch(() => null)) as
    | { message?: string; errors?: Array<{ line?: number; message?: string }> }
    | null
  const first = payload?.errors?.[0]
  if (first?.line) return `Ligne ${first.line} : ${first.message ?? 'ordre invalide'}`
  return payload?.message ?? `La requête a échoué (${response.status})`
}

function App() {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [selectedPlayer, setSelectedPlayer] = useState<PlayerId>('P1')
  const [map, setMap] = useState<MapData | null>(null)
  const [state, setState] = useState<StateData | null>(null)
  const [report, setReport] = useState<TurnReport | null>(null)
  const [chainDrafts, setChainDrafts] = useState<Record<PlayerId, Record<string, string>>>({})
  const [winterDrafts, setWinterDrafts] = useState<Record<PlayerId, string>>({})
  const [submittedPlayers, setSubmittedPlayers] = useState<PlayerId[]>([])
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [resolving, setResolving] = useState(false)

  useEffect(() => {
    const controller = new AbortController()
    const loadData = async () => {
      try {
        const [mapResponse, stateResponse] = await Promise.all([
          fetch('/api/map', { signal: controller.signal }),
          fetch('/api/state', { signal: controller.signal }),
        ])
        if (!mapResponse.ok) throw new Error(`Map request failed with status ${mapResponse.status}`)
        if (!stateResponse.ok) throw new Error(`State request failed with status ${stateResponse.status}`)
        const [mapData, stateData] = (await Promise.all([
          mapResponse.json(),
          stateResponse.json(),
        ])) as [MapData, StateData]
        if (!controller.signal.aborted) {
          setMap(mapData)
          setState(stateData)
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setLoadError(error instanceof Error ? error.message : 'Impossible de charger la partie')
        }
      }
    }
    void loadData()
    return () => controller.abort()
  }, [])

  useEffect(() => {
    if (state && !state.players.some((player) => player.id === selectedPlayer)) {
      setSelectedPlayer(state.players[0]?.id ?? 'P1')
    }
  }, [selectedPlayer, state])

  const selectedTerritory = map?.territories.find((territory) => territory.id === selectedId)
  const selectedState = state?.territories.find((territory) => territory.id === selectedId)
  const observedTurn = state && selectedState ? (state.asOf[selectedState.id] ?? state.turn) : null
  const isStale = state !== null && observedTurn !== null && observedTurn < state.turn
  const selectedChain = selectedState?.army?.chain ?? null
  const currentOrder = selectedChain?.orders[selectedChain.currentIndex]

  const updateChainDraft = (noble: string, text: string) => {
    setChainDrafts((drafts) => ({
      ...drafts,
      [selectedPlayer]: { ...(drafts[selectedPlayer] ?? {}), [noble]: text },
    }))
  }

  const updateWinterDraft = (text: string) => {
    setWinterDrafts((drafts) => ({ ...drafts, [selectedPlayer]: text }))
  }

  const submitOrders = async () => {
    if (!state) return
    setResolving(true)
    setActionError(null)
    const chains = state.season === 'winter'
      ? []
      : state.nobles
          .filter((noble) => noble.owner === selectedPlayer && noble.status === 'free')
          .map((noble) => ({
            player: selectedPlayer,
            noble: noble.code,
            text: addNobleHeader(noble.code, chainDrafts[selectedPlayer]?.[noble.code] ?? ''),
          }))
          .filter((submission) => submission.text.replace(new RegExp(`^${escapeRegExp(submission.noble)}\\s*`), '').trim() !== '')
    const winter = state.season === 'winter' && (winterDrafts[selectedPlayer] ?? '').trim() !== ''
      ? [{ player: selectedPlayer, lines: winterDrafts[selectedPlayer] ?? '' }]
      : []

    try {
      const response = await fetch('/api/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ player: selectedPlayer, chains, winter }),
      })
      if (!response.ok) throw new Error(await responseError(response))
      const payload = (await response.json()) as OrdersResponse
      setState(payload.state)
      setSubmittedPlayers(payload.submitted)
      if (payload.status === 'resolved' && payload.report) {
        setReport(payload.report)
        setChainDrafts({})
        setWinterDrafts({})
        setSubmittedPlayers([])
      }
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'La résolution a échoué')
    } finally {
      setResolving(false)
    }
  }

  return (
    <div className="min-h-screen bg-[#efe7d8] text-[#30291f]">
      <header className="border-b border-[#b7a786]/60 bg-[#fffaf0]/90 px-4 py-4 shadow-sm backdrop-blur-sm sm:px-6">
        <div className="mx-auto flex max-w-[1800px] flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-full border-2 border-[#a84632] bg-[#f6dfc6] font-serif text-sm font-bold text-[#a84632] shadow-inner">
              C&amp;B
            </div>
            <div>
              <h1 className="font-serif text-xl font-semibold tracking-tight sm:text-2xl">Crown &amp; Borough</h1>
              <p className="text-xs uppercase tracking-[0.18em] text-[#806f57]">Chroniques du royaume</p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 sm:gap-5">
            <div className="border-l border-[#b7a786]/60 pl-3 sm:pl-5">
              <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-[#806f57]">Saison</p>
              <p className="font-serif text-base font-semibold">
                {state ? `Tour ${state.turn} · ${SEASON_LABELS[state.season]}` : 'Chargement…'}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <label htmlFor="player-view" className="text-xs font-medium text-[#806f57]">Joueur actif</label>
              <Select value={selectedPlayer} onValueChange={setSelectedPlayer}>
                <SelectTrigger id="player-view" className="w-[172px] border-[#b7a786] bg-[#fffaf0] text-[#30291f]">
                  <SelectValue placeholder="Choisir un joueur" />
                </SelectTrigger>
                <SelectContent>
                  {state?.players.map((player) => (
                    <SelectItem key={player.id} value={player.id}>{player.id} · {player.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
      </header>

      <main className="mx-auto flex min-h-[calc(100vh-6.5rem)] max-w-[1800px] flex-col gap-4 p-4 sm:p-6 lg:flex-row">
        <section className="relative h-[620px] min-h-0 flex-1 overflow-hidden rounded-2xl border border-[#b7a786] bg-[#e6d8bb] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)] lg:h-[calc(100vh-8.5rem)]">
          <div className="pointer-events-none absolute left-5 top-5 z-10">
            <p className="text-[10px] font-semibold uppercase tracking-[0.22em] text-[#806f57]">Carte publique · vision T0</p>
            <p className="mt-1 text-xs text-[#594b3c]">Clic gauche pour sélectionner · maintenir puis glisser pour déplacer la carte · molette pour zoomer</p>
          </div>
          {loadError ? (
            <div role="alert" className="flex h-full items-center justify-center px-6 text-center">
              <p className="font-serif text-lg text-[#a84632]">Impossible de charger la partie : {loadError}</p>
            </div>
          ) : map && state ? (
            <MapViewer map={map} state={state} onSelect={setSelectedId} />
          ) : (
            <div className="flex h-full items-center justify-center px-6 text-center">
              <p className="font-serif text-lg italic text-[#806f57]">Chargement de la carte et de l&apos;état…</p>
            </div>
          )}
        </section>

        <aside className="w-full shrink-0 space-y-4 lg:w-96 xl:w-[27rem]">
          <Card className="border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
            <CardHeader className="border-b border-[#b7a786]/50 pb-4">
              <CardTitle className="font-serif text-xl text-[#30291f]">Poste de commandement</CardTitle>
              <CardDescription className="text-[#806f57]">Joueur sélectionné : {selectedPlayer}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5 pt-5">
              {selectedTerritory ? (
                <div className="space-y-5">
                  <div>
                    <p className="text-xs font-bold uppercase tracking-[0.2em] text-[#a84632]">{selectedTerritory.code}</p>
                    <h2 className="mt-1 font-serif text-2xl font-semibold leading-tight">{selectedTerritory.name}</h2>
                  </div>
                  <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm">
                    <dt className="text-[#806f57]">Terrain</dt>
                    <dd className="font-medium">{TERRAIN_NAMES[selectedTerritory.terrain]}</dd>
                    {selectedState && (
                      <>
                        <dt className="text-[#806f57]">Contrôle</dt>
                        <dd className="font-medium">{ownerLabel(selectedState.owner, state)}</dd>
                        <dt className="text-[#806f57]">Ressources</dt>
                        <dd className="font-medium">{selectedState.resources} R</dd>
                      </>
                    )}
                  </dl>

                  {selectedState && (
                    <>
                      <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                        <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">Armée</h3>
                        {selectedState.army ? (
                          <div className="rounded-md bg-[#f3ead9] px-3 py-2 text-sm">
                            <div className="flex items-center justify-between">
                              <span className="font-semibold">{ownerLabel(selectedState.army.owner, state)}</span>
                              <span className="text-xs text-[#806f57]">{selectedState.army.size} troupe{selectedState.army.size > 1 ? 's' : ''}</span>
                            </div>
                            <div className="mt-2 border-t border-[#b7a786]/40 pt-2 text-xs text-[#806f57]">
                              {selectedChain ? (
                                <>
                                  <p>Noble émetteur : <strong>{selectedChain.noble}</strong></p>
                                  <p>Index : <strong>{selectedChain.currentIndex}</strong></p>
                                  <p>Ordre courant : <strong>{currentOrder ? orderLabel(currentOrder) : 'Terminé'}</strong></p>
                                </>
                              ) : <p>Sans Ordre</p>}
                            </div>
                          </div>
                        ) : <p className="text-sm italic text-[#806f57]">Aucune armée</p>}
                      </div>

                      <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                        <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">Infrastructures</h3>
                        {selectedState.infrastructures.length > 0 ? (
                          <ul className="space-y-1.5 text-sm">
                            {selectedState.infrastructures.map((infrastructure, index) => (
                              <li key={`${infrastructure.type}-${index}`} className="flex items-center justify-between rounded-md bg-[#f3ead9] px-3 py-2">
                                <span className="font-medium">{INFRASTRUCTURE_LABELS[infrastructure.type]}</span>
                                <span className="text-xs text-[#806f57]">Niveau {infrastructure.level}</span>
                              </li>
                            ))}
                          </ul>
                        ) : <p className="text-sm italic text-[#806f57]">Aucune infrastructure</p>}
                      </div>

                      {state && observedTurn !== null && (
                        <div className={`rounded-lg border px-3 py-3 text-sm ${isStale ? 'border-[#c98d45]/50 bg-[#fbefd9] text-[#805521]' : 'border-[#6d9b73]/50 bg-[#e8f1e3] text-[#376341]'}`}>
                          <p className="font-semibold">Fraîcheur</p>
                          <p className="mt-1 text-xs leading-relaxed">{isStale ? `Observé au tour ${observedTurn} — il y a ${state.turn - observedTurn} tours` : 'À jour'}</p>
                        </div>
                      )}
                    </>
                  )}
                </div>
              ) : <div className="flex min-h-36 items-center justify-center rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] px-6 text-center"><p className="font-serif text-lg italic text-[#806f57]">Sélectionnez un territoire</p></div>}

              {state && (
                <OrdersPanel
                  state={state}
                  player={selectedPlayer}
                  chainDrafts={chainDrafts[selectedPlayer] ?? {}}
                  winterDraft={winterDrafts[selectedPlayer] ?? ''}
                  submitted={submittedPlayers.includes(selectedPlayer)}
                  submitting={resolving}
                  onChainChange={updateChainDraft}
                  onWinterChange={updateWinterDraft}
                  onSubmit={() => void submitOrders()}
                />
              )}
              {actionError && <p role="alert" className="rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-xs text-[#8d321e]">{actionError}</p>}
            </CardContent>
          </Card>

          {report && (
            <Card className="border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
              <CardContent className="pt-5"><ReportPanel report={report} map={map} /></CardContent>
            </Card>
          )}
        </aside>
      </main>
    </div>
  )
}

export default App
