import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { formatOrderLabel } from '@/lib/order-label'
import { formatCardLabel } from '@/lib/card-hand'
import { religiousTitlesByHolder } from '@/lib/lineage'
import { playerDisplayName, type PlayerName } from '@/lib/player-label'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { isOccupiedAgainstController } from '@/lib/occupation'
import { hasSupplySource } from '@/lib/supply'
import type {
  MapData,
  PlayerId,
  Region,
  StateData,
  SupplyLine,
  TransferLine,
} from '@/types'
import { DEFAULT_PLAYER_COLOR } from '@/lib/map-palette'

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

const FIEF_TITLE_KEYS: Record<
  NonNullable<StateData['fiefs']>[number]['title'],
  MessageKey
> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

type MapTerritory = MapData['territories'][number]
type TerritoryState = StateData['territories'][number]

interface SelectedTerritoryDetailsProps {
  state: StateData | null
  selectedTerritory: MapTerritory | null | undefined
  selectedState: TerritoryState | null | undefined
  selectedRegion?: Region | null
  preferredPlayers?: readonly PlayerName[]
  mapTerritories?: readonly MapTerritory[]
  selectedSupplyLine: SupplyLine | null
  sourceTerritory: MapTerritory | null | undefined
  supplyLoading: boolean
  supplyError: string | null
  transferTargets?: readonly string[]
  selectedTransferTarget?: string | null
  onTransferTargetChange?: (target: string) => void
  transferLine?: TransferLine | null
  transferLoading?: boolean
  transferError?: string | null
}

export function SelectedTerritoryDetails({
  state,
  selectedTerritory,
  selectedState,
  selectedRegion,
  preferredPlayers,
  mapTerritories = [],
  selectedSupplyLine,
  sourceTerritory,
  supplyLoading,
  supplyError,
  transferTargets = [],
  selectedTransferTarget = null,
  onTransferTargetChange = () => undefined,
  transferLine = null,
  transferLoading = false,
  transferError = null,
}: SelectedTerritoryDetailsProps) {
  const { t } = useLanguage()

  if (!state || !selectedTerritory) {
    return (
      <div className="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-line bg-paper-100 px-6 text-center">
        <p className="font-display text-base italic text-ink-500">
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
    state.players.find((player) => player.id === playerID)?.color || DEFAULT_PLAYER_COLOR
  const selectedCapitalPlayer = state.players.find(
    (player) => player.capitalTerritory === selectedTerritory.id,
  )
  const selectedFief = (state.fiefs ?? []).find((fief) =>
    fief.territories.includes(selectedTerritory.id),
  )
  const selectedFiefHolder = selectedFief?.holder
    ? state.nobles.find((noble) => noble.code === selectedFief.holder)
    : null
  const isFiefCapital = selectedFief?.capital === selectedTerritory.id
  const occupied = isOccupiedAgainstController(selectedState)
  const selectedChain = selectedState?.army?.chain ?? null
  const religiousTitles = religiousTitlesByHolder({
    bishoprics: state.bishoprics,
    cardinals: state.cardinals,
    pope: state.pope,
    king: state.king,
    queen: state.queen,
    queenDowager: state.queenDowager,
  })
  const excommunicatedCodes = new Set((state.excommunicated ?? []).map((entry) => entry.noble))
  const presentNobles = state.nobles.filter(
    (noble) => noble.location === selectedTerritory.id,
  )
  const territoryLabel = (territoryID: string) => {
    const territory = mapTerritories.find((candidate) => candidate.id === territoryID)
    return territory ? `${territory.name} (${territory.id})` : territoryID
  }
  const activeRegionEffects = selectedRegion
    ? (state.activeRegionEffects ?? []).filter(
        (effect) => effect.regionSeed === selectedRegion.seed,
      )
    : []

  return (
    <div className="space-y-5">
      <div>
        <p className="text-xs font-bold uppercase tracking-[0.2em] text-brass-700">
          {selectedTerritory.id}
        </p>
        <h2 className="mt-1 font-display text-2xl font-semibold leading-tight">
          {selectedTerritory.name}
        </h2>
        {selectedCapitalPlayer && (
          <p className="mt-2 inline-flex items-center rounded-full border border-brass-700/40 bg-brass-300/30 px-2.5 py-1 text-xs font-semibold text-brass-700">
            {t('app.capitalOf', {
              player: displayOwner(selectedCapitalPlayer.id, selectedCapitalPlayer.name),
            })}
          </p>
        )}
        {isFiefCapital && (
          <p className="mt-2 ml-2 inline-flex items-center rounded-full border border-vermilion-700/40 bg-vermilion-100/60 px-2.5 py-1 text-xs font-semibold text-vermilion-700">
            {t('app.cityBonus')}
          </p>
        )}
        {occupied && (
          <p className="mt-2 ml-2 inline-flex items-center rounded-full border border-vermilion-700/40 bg-vermilion-100/60 px-2.5 py-1 text-xs font-semibold text-vermilion-700">
            {t('app.occupiedBy', {
              player: displayOwner(
                selectedState?.army?.owner,
                selectedState?.army?.owner ?? '',
              ),
            })}
          </p>
        )}
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm">
        <dt className="text-ink-500">{t('app.terrain')}</dt>
        <dd className="font-medium">
          {t(TERRAIN_LABEL_KEYS[selectedTerritory.terrain])}
        </dd>
        {selectedRegion && (
          <>
            <dt className="text-ink-500">{t('app.region')}</dt>
            <dd className="font-medium">
              <span>{selectedRegion.seed}</span>
              <span className="ml-2 text-xs font-semibold text-ink-700">
                {t('app.regionSeedLabel', { seed: selectedRegion.seed })}
              </span>
            </dd>
          </>
        )}
        {selectedFief && (
          <>
            <dt className="text-ink-500">{t('app.fiefTitle')}</dt>
            <dd className="font-medium">
              {t(FIEF_TITLE_KEYS[selectedFief.title])} —{' '}
              {territoryLabel(selectedFief.capital)}
            </dd>
            <dt className="text-ink-500">{t('app.fiefHolder')}</dt>
            <dd className="font-medium">
              {selectedFiefHolder
                ? `${selectedFiefHolder.name} (${selectedFiefHolder.code})`
                : t('app.fiefVacant')}
            </dd>
            {selectedFief.projectedIncome !== undefined && (
              <>
                <dt className="text-ink-500">{t('app.fiefProjectedIncome')}</dt>
                <dd className="font-medium">
                  {t('app.fiefProjectedIncomeAmount', {
                    amount: selectedFief.projectedIncome,
                  })}
                </dd>
              </>
            )}
          </>
        )}
        {selectedState && (
          <>
            <dt className="text-ink-500">{t('app.control')}</dt>
            <dd className="font-medium">
              {displayOwner(selectedState.owner, t('app.noOwner'))}
            </dd>
            <dt className="text-ink-500">{t('app.resources')}</dt>
            <dd className="font-medium">{selectedState.resources} R</dd>
            {selectedState.owner && selectedState.projectedIncome !== undefined && (
              <>
                <dt className="text-ink-500">{t('app.projectedIncome')}</dt>
                <dd className="font-medium">
                  {selectedState.incomeDestination
                    ? t('app.territoryIncome', {
                        amount: selectedState.projectedIncome,
                        destination: territoryLabel(selectedState.incomeDestination),
                      })
                    : t('app.territoryIncomeLost')}
                </dd>
              </>
            )}
          </>
        )}
      </dl>

      {activeRegionEffects.length > 0 && (
        <div className="space-y-2 border-t border-line/50 pt-4">
          <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
            {t('app.activeEffects')}
          </h3>
          <ul className="space-y-1.5 text-sm">
            {activeRegionEffects.map((effect, index) => {
              const current = effect.season === state.season
              return (
                <li
                  key={`${effect.kind}-${effect.season}-${index}`}
                  className="rounded-md bg-paper-100 px-3 py-2"
                >
                  <span className={current ? 'font-bold' : 'font-medium'}>
                    {formatCardLabel(effect.kind, t)}
                  </span>
                  <span className="mt-1 block text-xs text-ink-500">
                    {t('app.effectSeason', {
                      season: t(SEASON_LABEL_KEYS[effect.season]),
                    })}
                  </span>
                </li>
              )
            })}
          </ul>
        </div>
      )}

      <div className="space-y-2 border-t border-line/50 pt-4">
        <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
          {t('app.noblesPresent')}
        </h3>
        {presentNobles.length > 0 ? (
          <ul className="space-y-1.5 text-sm">
            {presentNobles.map((noble) => {
              const holder =
                noble.status !== 'free' ? (selectedState?.army?.owner ?? null) : null
              return (
                <li key={noble.id} className="rounded-md bg-paper-100 px-3 py-2 text-sm">
                  <div className="flex items-center justify-between gap-3">
                    <span className="min-w-0">
                      <span
                        className="mr-2 inline-block size-3 shrink-0 rounded-full border border-ink-900/30 align-[-1px]"
                        style={{ backgroundColor: playerColor(noble.owner) }}
                        aria-label={t('app.colorOf', {
                          player: displayOwner(noble.owner, noble.owner),
                        })}
                      />
                      <strong>{noble.name}</strong> ({noble.code})
                    </span>
                    <span
                      className={`shrink-0 text-xs ${noble.status === 'dungeon' ? 'text-brass-700' : 'text-success-700'}`}
                    >
                      {t(`orders.nobleStatus.${noble.status}` as MessageKey)}
                    </span>
                  </div>
                  <dl className="mt-1 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 border-t border-line/40 pt-1 text-xs text-ink-500">
                    <dt>{t('app.owner')}</dt>
                    <dd className="font-medium text-ink-700">
                      {displayOwner(noble.owner, noble.owner)}
                    </dd>
                    {(() => {
                      const fiefTitles = (state.fiefs ?? []).filter(
                        (fief) => fief.holder === noble.code,
                      )
                      const religious = religiousTitles.get(noble.code) ?? []
                      const excommunicated = excommunicatedCodes.has(noble.code)
                      if (fiefTitles.length === 0 && religious.length === 0 && !excommunicated) return null
                      return (
                        <>
                          <dt>{t('app.titles')}</dt>
                          <dd className="flex flex-wrap gap-1 font-medium text-ink-700">
                            {religious.map((held) => (
                              <span
                                key={`${held.title}-${held.seat ?? ''}`}
                                className="rounded bg-paper-100 px-1.5 py-0.5 text-2xs uppercase tracking-wide"
                              >
                                {t(`lineage.religious.${held.title}` as MessageKey, {
                                  seat: held.seat ?? '',
                                })}
                              </span>
                            ))}
                            {excommunicated && (
                              <span className="rounded bg-vermilion-700/30 px-1.5 py-0.5 text-2xs uppercase tracking-wide text-vermilion-700">
                                {t('lineage.excommunicated')}
                              </span>
                            )}
                            {fiefTitles.map((fief) => (
                              <span
                                key={fief.capital}
                                className="rounded bg-paper-300 px-1.5 py-0.5 text-2xs uppercase tracking-wide"
                              >
                                {t(FIEF_TITLE_KEYS[fief.title])} · {fief.capital}
                              </span>
                            ))}
                          </dd>
                        </>
                      )
                    })()}
                    {noble.dignities && noble.dignities.length > 0 && (
                      <>
                        <dt>{t('app.dignity')}</dt>
                        <dd className="flex flex-wrap gap-1 font-medium text-ink-700">
                          {noble.dignities.map((dignity) => (
                            <span
                              key={dignity}
                              className="rounded bg-paper-300 px-1.5 py-0.5 text-2xs uppercase tracking-wide"
                            >
                              {t(`dignity.${dignity}` as MessageKey)}
                            </span>
                          ))}
                        </dd>
                      </>
                    )}
                    {noble.secret && (
                      <>
                        <dt>{t('app.secretIdentity')}</dt>
                        <dd className="font-medium text-ink-700">
                          {noble.secret.name} ({noble.secret.code})
                        </dd>
                      </>
                    )}
                    {noble.commissioned && noble.commissioned.length > 0 && (
                      <>
                        <dt>{t('app.commissionedKilling')}</dt>
                        <dd
                          className="font-medium text-vermilion-700"
                          data-testid="assassination-victim"
                        >
                          {noble.commissioned
                            .map((code) => {
                              const name =
                                state.nobles.find((other) => other.code === code)?.name ??
                                state.deceased?.find((other) => other.code === code)?.name
                              return name ? `${name} (${code})` : code
                            })
                            .join(', ')}
                        </dd>
                      </>
                    )}
                    {noble.spouse && (
                      <>
                        <dt>{t('app.marriage')}</dt>
                        <dd className="font-medium text-ink-700">
                          {t(noble.sex === 'female' ? 'app.wifeOf' : 'app.husbandOf', {
                            spouse: noble.spouse,
                          })}
                        </dd>
                      </>
                    )}
                    {holder && (
                      <>
                        <dt>{t('app.holder')}</dt>
                        <dd className="font-medium text-ink-700">
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
          <p className="text-sm italic text-ink-500">{t('app.noNoble')}</p>
        )}
      </div>

      {selectedState && (
        <>
          <div className="space-y-2 border-t border-line/50 pt-4">
            <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
              {t('app.army')}
            </h3>
            {selectedState.army ? (
              <div className="rounded-md bg-paper-100 px-3 py-2 text-sm">
                <div className="flex items-center justify-between gap-3">
                  <span className="font-semibold">
                    {displayOwner(selectedState.army.owner, selectedState.army.owner)}
                  </span>
                  <span className="shrink-0 text-xs text-ink-500">
                    {t(selectedState.army.size === 1 ? 'app.troop' : 'app.troops', {
                      count: selectedState.army.size,
                    })}
                  </span>
                </div>
                {selectedState.army.starving && (
                  <p className="mt-2 text-xs font-semibold text-vermilion-700">
                    {t('app.armyStarving')}
                  </p>
                )}
                <div className="mt-2 border-t border-line/40 pt-2 text-xs text-ink-500">
                  {selectedChain?.visibility === 'hidden' ? (
                    <p className="rounded-md border border-line/50 bg-paper-50 px-2 py-1.5 italic">
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
                        <p className="font-semibold uppercase tracking-[0.12em] text-ink-500">
                          {t('app.orderStack')}
                        </p>
                        <ol className="space-y-1.5">
                          {(selectedChain.orders ?? []).map((order, index) => {
                            const current = index === (selectedChain.currentIndex ?? 0)
                            return (
                              <li
                                key={`${order.type}-${order.position}-${index}`}
                                aria-current={current ? 'step' : undefined}
                                className={`rounded-md border px-2 py-1.5 ${current ? 'border-vermilion-700/60 bg-vermilion-100 text-vermilion-700' : 'border-line/40 bg-paper-50'}`}
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
              <p className="text-sm italic text-ink-500">{t('app.noArmy')}</p>
            )}
            {selectedState.army && state.season === 'winter' && (
              <p className="rounded-md border border-line/50 bg-paper-50 px-3 py-2 text-xs text-ink-500">
                {t('app.noSupplyWinter')}
              </p>
            )}
            {selectedState.army && state.season !== 'winter' && (
              <div className="rounded-md border border-line/50 bg-paper-50 px-3 py-2 text-xs text-ink-700">
                <p className="font-semibold uppercase tracking-[0.12em] text-ink-500">
                  {t('app.supply')}
                </p>
                {supplyLoading ? (
                  <p className="mt-1 italic text-ink-500">
                    {t('app.supplyCalculating')}
                  </p>
                ) : supplyError ? (
                  <p className="mt-1 text-vermilion-700">{supplyError}</p>
                ) : selectedSupplyLine?.selfSupplied ? (
                  <p className="mt-1 text-success-700">{t('app.localRationsSufficient')}</p>
                ) : selectedSupplyLine?.source ? (
                  <>
                    <p className="mt-1">
                      {t('app.sourceLabel')}{' '}
                      <strong>
                        {sourceTerritory
                          ? `${sourceTerritory.name} (${sourceTerritory.id})`
                          : selectedSupplyLine.source}
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
                  <p className="mt-1 text-vermilion-700">{t('app.noAccessibleSource')}</p>
                ) : null}
                {selectedSupplyLine && (
                  <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 border-t border-line/40 pt-2">
                    <dt className="text-ink-500">{t('app.localProduction')}</dt>
                    <dd className="font-medium">
                      {selectedSupplyLine.localProduction}{' '}
                      <span className="text-xs font-normal text-ink-500">
                        ({t(TERRAIN_LABEL_KEYS[selectedTerritory.terrain])}{' '}
                        {selectedSupplyLine.terrainProduction}
                        {(selectedSupplyLine.famineRations ?? 0) > 0
                          ? ` − ${t('app.localFamineRations')} ${selectedSupplyLine.famineRations}`
                          : ''}
                        {(selectedSupplyLine.bonusRations ?? 0) > 0
                          ? ` + ${t('app.localBonusRations')} ${selectedSupplyLine.bonusRations}`
                          : ''}
                        )
                      </span>
                    </dd>
                    <dt className="text-ink-500">{t('app.demand')}</dt>
                    <dd className="font-medium">{selectedSupplyLine.totalDemand}</dd>
                    <dt className="text-ink-500">{t('app.toCover')}</dt>
                    <dd
                      className={
                        selectedSupplyLine.demand > 0
                          ? 'font-semibold text-vermilion-700'
                          : 'font-medium'
                      }
                    >
                      {selectedSupplyLine.demand}
                    </dd>
                  </dl>
                )}
              </div>
            )}
            {!selectedState.army && hasSupplySource(selectedState) && (
              <div className="rounded-md border border-line/50 bg-paper-50 px-3 py-2 text-xs text-ink-700">
                <p className="font-semibold uppercase tracking-[0.12em] text-ink-500">
                  {t('app.supplyZone')}
                </p>
                {state.season === 'winter' ? (
                  <p className="mt-1 italic text-ink-500">{t('app.noSupplyWinter')}</p>
                ) : supplyLoading ? (
                  <p className="mt-1 italic text-ink-500">
                    {t('app.supplyCalculating')}
                  </p>
                ) : supplyError ? (
                  <p className="mt-1 text-vermilion-700">{supplyError}</p>
                ) : selectedSupplyLine ? (
                  <p className="mt-1 text-success-700">
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
            {state.season !== 'winter' &&
              selectedState.army &&
              transferTargets.length > 0 && (
                <div className="rounded-md border border-line/50 bg-paper-50 px-3 py-2 text-xs text-ink-700">
                  <p className="font-semibold uppercase tracking-[0.12em] text-ink-500">
                    {t('app.transferPreview')}
                  </p>
                  <label
                    htmlFor="transfer-preview-target"
                    className="mt-2 block text-ink-500"
                  >
                    {t('app.transferTarget')}
                  </label>
                  <select
                    id="transfer-preview-target"
                    aria-label={t('app.transferTarget')}
                    value={selectedTransferTarget ?? transferTargets[0]}
                    onChange={(event) => onTransferTargetChange(event.target.value)}
                    className="mt-1 w-full rounded-md border border-line-strong bg-paper-50 px-2 py-1.5 text-xs text-ink-900 outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20"
                  >
                    {transferTargets.map((target) => (
                      <option key={target} value={target}>
                        {territoryLabel(target)}
                      </option>
                    ))}
                  </select>
                  {transferLoading ? (
                    <p className="mt-2 italic text-ink-500">
                      {t('app.supplyCalculating')}
                    </p>
                  ) : transferError ? (
                    <p className="mt-2 text-vermilion-700">{transferError}</p>
                  ) : transferLine ? (
                    <>
                      <p
                        className={`mt-2 font-semibold ${transferLine.reachable ? 'text-success-700' : 'text-vermilion-700'}`}
                        role="status"
                      >
                        {transferLine.reachable
                          ? t('app.transferReachable')
                          : t('reports.reason.transfer_path_blocked')}
                      </p>
                      <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 border-t border-line/40 pt-2">
                        <dt className="text-ink-500">{t('app.sourceLabel')}</dt>
                        <dd className="font-medium">
                          {territoryLabel(transferLine.source)}
                        </dd>
                        <dt className="text-ink-500">{t('app.transferTarget')}</dt>
                        <dd className="font-medium">
                          {territoryLabel(transferLine.target)}
                        </dd>
                        {transferLine.reachable &&
                          transferLine.distance !== undefined && (
                            <>
                              <dt className="text-ink-500">{t('app.distanceLabel')}</dt>
                              <dd className="font-medium">
                                {t(
                                  transferLine.distance > 1
                                    ? 'app.distances'
                                    : 'app.distance',
                                  { distance: transferLine.distance },
                                )}
                              </dd>
                            </>
                          )}
                      </dl>
                      {transferLine.path.length > 0 && (
                        <p className="mt-2 border-t border-line/40 pt-2">
                          <span className="text-ink-500">
                            {t('app.transferPath')}:{' '}
                          </span>
                          <span className="font-medium">
                            {transferLine.path.map(territoryLabel).join(' -> ')}
                          </span>
                        </p>
                      )}
                    </>
                  ) : null}
                </div>
              )}
          </div>

          <div className="space-y-2 border-t border-line/50 pt-4">
            <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
              {t('app.infrastructure')}
            </h3>
            {selectedState.infrastructures.length > 0 ? (
              <ul className="space-y-1.5 text-sm">
                {selectedState.infrastructures.map((infrastructure, index) => (
                  <li
                    key={`${infrastructure.type}-${index}`}
                    className="rounded-md bg-paper-100 px-3 py-2"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <span className="flex min-w-0 items-center gap-2 font-medium">
                        <span>{t(INFRASTRUCTURE_LABEL_KEYS[infrastructure.type])}</span>
                        {infrastructure.type === 'castle' && selectedCapitalPlayer && (
                          <span className="shrink-0 rounded-full bg-brass-300 px-1.5 py-0.5 text-2xs font-bold uppercase tracking-wide text-ink-900">
                            {t('app.capital')}
                          </span>
                        )}
                      </span>
                      <span className="shrink-0 text-xs text-ink-500">
                        {t('app.level', { level: infrastructure.level })}
                      </span>
                    </div>
                    {infrastructure.type === 'mill' && selectedState.millDestination && (
                      <p className="mt-1 text-xs text-ink-500">
                        {selectedState.millDestination === selectedState.id
                          ? t('app.millKeptInPlace')
                          : t('app.millDestination', {
                              destination: territoryLabel(selectedState.millDestination),
                            })}
                      </p>
                    )}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm italic text-ink-500">{t('app.noInfrastructure')}</p>
            )}
          </div>
        </>
      )}
    </div>
  )
}
