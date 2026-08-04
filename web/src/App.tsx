import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
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
import { MapViewer } from '@/components/MapViewer'
import { stateFixture } from '@/fixtures/state'
import type { InfraType, MapData, PlayerId, Season, Terrain } from '@/types'

const TERRAIN_LABELS: Record<Terrain, string> = {
  plain: 'Plaine',
  forest: 'Forêt',
  hill: 'Colline',
  mountain: 'Montagne',
  swamp: 'Marécage',
}

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

function ownerLabel(owner: PlayerId | null): string {
  return owner ? `Joueur ${owner}` : 'Personne'
}

function App() {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [map, setMap] = useState<MapData | null>(null)
  const [mapError, setMapError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    const loadMap = async () => {
      try {
        const response = await fetch('/api/map', { signal: controller.signal })
        if (!response.ok) {
          throw new Error(`Map request failed with status ${response.status}`)
        }

        const data = (await response.json()) as MapData
        if (!controller.signal.aborted) {
          setMap(data)
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setMapError(error instanceof Error ? error.message : 'Unable to load the map')
        }
      }
    }

    void loadMap()
    return () => controller.abort()
  }, [])

  const selectedTerritory = map?.territories.find(
    (territory) => territory.id === selectedId,
  )
  const selectedState = stateFixture.territories.find(
    (territory) => territory.id === selectedId,
  )
  const observedTurn = selectedState
    ? (stateFixture.asOf[selectedState.id] ?? stateFixture.turn)
    : stateFixture.turn
  const isStale = observedTurn < stateFixture.turn

  return (
    <div className="min-h-screen bg-[#efe7d8] text-[#30291f]">
      <header className="border-b border-[#b7a786]/60 bg-[#fffaf0]/90 px-4 py-4 shadow-sm backdrop-blur-sm sm:px-6">
        <div className="mx-auto flex max-w-[1800px] flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-full border-2 border-[#a84632] bg-[#f6dfc6] font-serif text-sm font-bold text-[#a84632] shadow-inner">
              C&B
            </div>
            <div>
              <h1 className="font-serif text-xl font-semibold tracking-tight sm:text-2xl">
                Crown &amp; Borough
              </h1>
              <p className="text-xs uppercase tracking-[0.18em] text-[#806f57]">
                Chroniques du royaume
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 sm:gap-5">
            <div className="border-l border-[#b7a786]/60 pl-3 sm:pl-5">
              <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-[#806f57]">
                Saison
              </p>
              <p className="font-serif text-base font-semibold">
                Tour {stateFixture.turn} · {SEASON_LABELS[stateFixture.season]}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <label htmlFor="player-view" className="text-xs font-medium text-[#806f57]">
                Vue du joueur
              </label>
              <Select defaultValue="P1">
                <SelectTrigger
                  id="player-view"
                  className="w-[152px] border-[#b7a786] bg-[#fffaf0] text-[#30291f]"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="P1">P1 · Actif</SelectItem>
                  <SelectItem value="P2" disabled>
                    P2 · Bientôt
                  </SelectItem>
                  <SelectItem value="P3" disabled>
                    P3 · Bientôt
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
      </header>

      <main className="mx-auto flex min-h-[calc(100vh-6.5rem)] max-w-[1800px] flex-col gap-4 p-4 sm:p-6 lg:flex-row">
        <section className="relative h-[620px] min-h-0 flex-1 overflow-hidden rounded-2xl border border-[#b7a786] bg-[#e6d8bb] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)] lg:h-[calc(100vh-8.5rem)]">
          <div className="pointer-events-none absolute left-5 top-5 z-10">
            <p className="text-[10px] font-semibold uppercase tracking-[0.22em] text-[#806f57]">
              Carte publique
            </p>
            <p className="mt-1 text-xs text-[#594b3c]">
              Clic milieu pour sélectionner · clic gauche pour déplacer · molette pour
              zoomer
            </p>
          </div>
          {mapError ? (
            <div
              role="alert"
              className="flex h-full items-center justify-center px-6 text-center"
            >
              <p className="font-serif text-lg text-[#a84632]">
                Impossible de charger la carte : {mapError}
              </p>
            </div>
          ) : map ? (
            <MapViewer map={map} state={stateFixture} onSelect={setSelectedId} />
          ) : (
            <div className="flex h-full items-center justify-center px-6 text-center">
              <p className="font-serif text-lg italic text-[#806f57]">
                Chargement de la carte…
              </p>
            </div>
          )}
        </section>

        <aside className="w-full shrink-0 lg:w-80 xl:w-96">
          <Card className="h-full border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
            <CardHeader className="border-b border-[#b7a786]/50 pb-4">
              <CardTitle className="font-serif text-xl text-[#30291f]">
                Détail du territoire
              </CardTitle>
              <CardDescription className="text-[#806f57]">
                {selectedTerritory
                  ? `${selectedTerritory.code} · ${selectedTerritory.name}`
                  : 'Sélectionnez un territoire sur la carte'}
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-5">
              {selectedTerritory ? (
                <div className="space-y-5">
                  <div>
                    <p className="text-xs font-bold uppercase tracking-[0.2em] text-[#a84632]">
                      {selectedTerritory.code}
                    </p>
                    <h2 className="mt-1 font-serif text-2xl font-semibold leading-tight">
                      {selectedTerritory.name}
                    </h2>
                  </div>

                  <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm">
                    <dt className="text-[#806f57]">Terrain</dt>
                    <dd className="font-medium">
                      {TERRAIN_LABELS[selectedTerritory.terrain]}
                    </dd>
                  </dl>

                  {selectedState && (
                    <>
                      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm">
                        <dt className="text-[#806f57]">Contrôle</dt>
                        <dd className="font-medium">{ownerLabel(selectedState.owner)}</dd>
                        <dt className="text-[#806f57]">Ressources</dt>
                        <dd className="font-medium">{selectedState.resources} R</dd>
                      </dl>

                      <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                        <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
                          Troupes
                        </h3>
                        {selectedState.troops.length > 0 ? (
                          <ul className="space-y-1.5 text-sm">
                            {selectedState.troops.map((troop) => (
                              <li
                                key={troop.id}
                                className="flex items-center justify-between rounded-md bg-[#f3ead9] px-3 py-2"
                              >
                                <span className="font-semibold">{troop.id}</span>
                                <span className="text-xs text-[#806f57]">
                                  {troop.owner}
                                </span>
                              </li>
                            ))}
                          </ul>
                        ) : (
                          <p className="text-sm italic text-[#806f57]">Aucune troupe</p>
                        )}
                      </div>

                      <div className="space-y-2 border-t border-[#b7a786]/50 pt-4">
                        <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
                          Infrastructures
                        </h3>
                        {selectedState.infrastructures.length > 0 ? (
                          <ul className="space-y-1.5 text-sm">
                            {selectedState.infrastructures.map(
                              (infrastructure, index) => (
                                <li
                                  key={`${infrastructure.type}-${index}`}
                                  className="flex items-center justify-between rounded-md bg-[#f3ead9] px-3 py-2"
                                >
                                  <span className="font-medium">
                                    {INFRASTRUCTURE_LABELS[infrastructure.type]}
                                  </span>
                                  <span className="text-xs text-[#806f57]">
                                    Niveau {infrastructure.level}
                                  </span>
                                </li>
                              ),
                            )}
                          </ul>
                        ) : (
                          <p className="text-sm italic text-[#806f57]">
                            Aucune infrastructure
                          </p>
                        )}
                      </div>

                      <div
                        className={`rounded-lg border px-3 py-3 text-sm ${
                          isStale
                            ? 'border-[#c98d45]/50 bg-[#fbefd9] text-[#805521]'
                            : 'border-[#6d9b73]/50 bg-[#e8f1e3] text-[#376341]'
                        }`}
                      >
                        <p className="font-semibold">Fraîcheur</p>
                        <p className="mt-1 text-xs leading-relaxed">
                          {isStale
                            ? `Observé au tour ${observedTurn} — il y a ${stateFixture.turn - observedTurn} tours`
                            : 'À jour'}
                        </p>
                      </div>

                      <Button disabled className="w-full">
                        Donner un ordre
                      </Button>
                      <p className="text-center text-xs text-[#806f57]">
                        Bientôt disponible
                      </p>
                    </>
                  )}
                </div>
              ) : (
                <div className="flex min-h-72 items-center justify-center rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] px-6 text-center">
                  <p className="font-serif text-lg italic text-[#806f57]">
                    Sélectionnez un territoire
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </aside>
      </main>
    </div>
  )
}

export default App
