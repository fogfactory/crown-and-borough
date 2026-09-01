import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { formatOrderLabel } from '@/lib/order-label'
import { playerDisplayName, type PlayerName } from '@/lib/player-label'
import { hasSupplySource } from '@/lib/supply'
import type { MapData, PlayerId, StateData, SupplyLine } from '@/types'

const TERRAIN_LABEL_KEYS: Record<MapData['territories'][number]['terrain'], MessageKey> =
  {
    plain: 'terrain.plain',
    forest: 'terrain.forest',
    hill: 'terrain.hill',
    mountain: 'terrain.mountain',
    swamp: 'terrain.swamp',
  }

const INFRASTRUCTURE_LABEL_KEYS: Record<
  StateData['territories'][number]['infrastructures'][number]['type'],
  MessageKey
> = {
  mill: 'infrastructure.mill',
  supply_depot: 'infrastructure.supply_depot',
  castle: 'infrastructure.castle',
  village: 'infrastructure.village',
}

type MapTerritory = MapData['territories'][number]
type TerritoryState = StateData['territories'][number]

interface SelectedTerritoryDetailsProps {
  state: StateData | null
  selectedTerritory: MapTerritory | null | undefined
  selectedState: TerritoryState | null | undefined
  preferredPlayers?: readonly PlayerName[]
  selectedSupplyLine: SupplyLine | null
  sourceTerritory: MapTerritory | null | undefined
  supplyLoading: boolean
  supplyError: string | null
}

export function SelectedTerritoryDetails({
  state,
  selectedTerritory,
  selectedState,
  preferredPlayers,
  selectedSupplyLine,
  sourceTerritory,
  supplyLoading,
  supplyError,
}: SelectedTerritoryDetailsProps) {
  const { t } = useLanguage()

  if (!state || !selectedTerritory) {
    return (
      <div className="flex min-h-36 items-center justify-center rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] px-6 text-center">
        <p className="font-serif text-lg italic text-[#806f57]">
          {t('app.selectTerritory')}
        </p>
      </div>
    )
  }

  const playerLists: readonly (readonly PlayerName[])[] = preferredPlayers
    ? [preferredPlayers, state.players]
    : [state.players]
  const displayOwner = (playerID: PlayerId | null | undefined, fallback: string) =>
    playerDisplayName(playerID, playerLists, fallback)
  const playerColor = (playerID: PlayerId) =>
    state.players.find((player) => player.id === playerID)?.color || '#b7a786'
  const selectedCapitalPlayer = state.players.find(
    (player) => player.capitalTerritory === selectedTerritory.id,
  )
  const selectedChain = selectedState?.army?.chain ?? null
  const presentNobles = state.nobles.filter(
    (noble) => noble.location === selectedTerritory.id,
  )

  return (
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
            {t('app.capitalOf', {
              player: displayOwner(selectedCapitalPlayer.id, selectedCapitalPlayer.name),
            })}
          </p>
        )}
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm">
        <dt className="text-[#806f57]">{t('app.terrain')}</dt>
        <dd className="font-medium">
          {t(TERRAIN_LABEL_KEYS[selectedTerritory.terrain])}
        </dd>
        {selectedState && (
          <>
            <dt className="text-[#806f57]">{t('app.control')}</dt>
            <dd className="font-medium">
              {displayOwner(selectedState.owner, t('app.noOwner'))}
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
              const holder =
                noble.status !== 'free' ? (selectedState?.army?.owner ?? null) : null
              return (
                <li key={noble.id} className="rounded-md bg-[#f3ead9] px-3 py-2 text-sm">
                  <div className="flex items-center justify-between gap-3">
                    <span className="min-w-0">
                      <span
                        className="mr-2 inline-block size-3 shrink-0 rounded-full border border-[#30291f]/30 align-[-1px]"
                        style={{ backgroundColor: playerColor(noble.owner) }}
                        aria-label={t('app.colorOf', {
                          player: displayOwner(noble.owner, noble.owner),
                        })}
                      />
                      <strong>{noble.code}</strong> · {noble.name}
                    </span>
                    <span
                      className={`shrink-0 text-xs ${noble.status === 'dungeon' ? 'text-[#a84632]' : 'text-[#376341]'}`}
                    >
                      {t(`orders.nobleStatus.${noble.status}` as MessageKey)}
                    </span>
                  </div>
                  <dl className="mt-1 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 border-t border-[#b7a786]/40 pt-1 text-xs text-[#806f57]">
                    <dt>{t('app.owner')}</dt>
                    <dd className="font-medium text-[#594b3c]">
                      {displayOwner(noble.owner, noble.owner)}
                    </dd>
                    {holder && (
                      <>
                        <dt>{t('app.holder')}</dt>
                        <dd className="font-medium text-[#594b3c]">
                          {noble.status === 'hostage'
                            ? t('app.hostageBy', {
                                player: displayOwner(holder, holder),
                              })
                            : t('app.dungeonBy', {
                                player: displayOwner(holder, holder),
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
          <p className="text-sm italic text-[#806f57]">{t('app.noNoble')}</p>
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
                    {displayOwner(selectedState.army.owner, selectedState.army.owner)}
                  </span>
                  <span className="shrink-0 text-xs text-[#806f57]">
                    {t(selectedState.army.size === 1 ? 'app.troop' : 'app.troops', {
                      count: selectedState.army.size,
                    })}
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
                        {t('app.nobleEmitter')}: <strong>{selectedChain.noble}</strong>
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
                          {(selectedChain.orders ?? []).map((order, index) => {
                            const current = index === (selectedChain.currentIndex ?? 0)
                            return (
                              <li
                                key={`${order.type}-${order.position}-${index}`}
                                aria-current={current ? 'step' : undefined}
                                className={`rounded-md border px-2 py-1.5 ${current ? 'border-[#a84632]/60 bg-[#f8e5dd] text-[#8d321e]' : 'border-[#b7a786]/40 bg-[#fffaf0]'}`}
                              >
                                <span className="mr-1.5 font-semibold">{index + 1}.</span>
                                <span>{formatOrderLabel(order)}</span>
                                {current && (
                                  <span className="ml-1.5 font-semibold">
                                    · {t('app.current')}
                                  </span>
                                )}
                              </li>
                            )
                          })}
                        </ol>
                      </div>
                    </>
                  ) : (
                    <p>{t('app.noOrders')}</p>
                  )}
                </div>
              </div>
            ) : (
              <p className="text-sm italic text-[#806f57]">{t('app.noArmy')}</p>
            )}
            {selectedState.army && state.season === 'winter' && (
              <p className="rounded-md border border-[#b7a786]/50 bg-[#fffaf0] px-3 py-2 text-xs text-[#806f57]">
                {t('app.noSupplyWinter')}
              </p>
            )}
            {selectedState.army && state.season !== 'winter' && (
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
                  <p className="mt-1 text-[#376341]">{t('app.localRationsSufficient')}</p>
                ) : selectedSupplyLine?.source ? (
                  <>
                    <p className="mt-1">
                      {t('app.sourceLabel')}{' '}
                      <strong>
                        {sourceTerritory?.id ?? selectedSupplyLine.source}
                        {sourceTerritory ? ` · ${sourceTerritory.name}` : ''}
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
                  <p className="mt-1 text-[#8d321e]">{t('app.noAccessibleSource')}</p>
                ) : null}
                {selectedSupplyLine && (
                  <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 border-t border-[#b7a786]/40 pt-2">
                    <dt className="text-[#806f57]">{t('app.localRations')}</dt>
                    <dd className="font-medium">{selectedSupplyLine.rations}</dd>
                    <dt className="text-[#806f57]">{t('app.demandToCover')}</dt>
                    <dd className="font-medium">{selectedSupplyLine.demand}</dd>
                  </dl>
                )}
              </div>
            )}
            {!selectedState.army && hasSupplySource(selectedState) && (
              <div className="rounded-md border border-[#b7a786]/50 bg-[#fffaf0] px-3 py-2 text-xs text-[#594b3c]">
                <p className="font-semibold uppercase tracking-[0.12em] text-[#806f57]">
                  {t('app.supplyZone')}
                </p>
                {state.season === 'winter' ? (
                  <p className="mt-1 italic text-[#806f57]">{t('app.noSupplyWinter')}</p>
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
                {selectedState.infrastructures.map((infrastructure, index) => (
                  <li
                    key={`${infrastructure.type}-${index}`}
                    className="flex items-center justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
                  >
                    <span className="flex min-w-0 items-center gap-2 font-medium">
                      <span>{t(INFRASTRUCTURE_LABEL_KEYS[infrastructure.type])}</span>
                      {infrastructure.type === 'castle' && selectedCapitalPlayer && (
                        <span className="shrink-0 rounded-full bg-[#f8e8ae] px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-[#6d5118]">
                          {t('app.capital')}
                        </span>
                      )}
                    </span>
                    <span className="shrink-0 text-xs text-[#806f57]">
                      {t('app.level', { level: infrastructure.level })}
                    </span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm italic text-[#806f57]">{t('app.noInfrastructure')}</p>
            )}
          </div>
        </>
      )}
    </div>
  )
}
