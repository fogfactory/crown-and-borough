import { Popover } from 'radix-ui'
import { IconCrown, IconX } from '@tabler/icons-react'

import { BannerIcon, ShieldIcon } from '@/components/ArmsIcons'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { formatOrderLabel } from '@/lib/order-label'
import { formatCardLabel } from '@/lib/card-hand'
import { CodeTag, NobleRef, TerritoryRef } from '@/components/MapLinks'
import { resolveArmsByOwner } from '@/lib/heraldry'
import { NoblePawnIcon } from '@/components/NobleHolder'
import { richTranslate } from '@/lib/rich-text'
import { religiousTitlesByHolder } from '@/lib/lineage'
import { nobleAttributesByCode } from '@/lib/noble-attributes'
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
  /** Closes the cartouche (its cross). */
  onClose?: () => void
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
  onClose,
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
  const armsByOwner = resolveArmsByOwner(
    state,
    new Map(state.players.map((player) => [player.id, player.color])),
  )
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
  const excommunicatedCodes = new Set(
    (state.excommunicated ?? []).map((entry) => entry.noble),
  )
  const pawnAttributes = nobleAttributesByCode(state)
  const presentNobles = state.nobles.filter(
    (noble) => noble.location === selectedTerritory.id,
  )
  // Text label, for the transfer target options (no markup inside <option>).
  const territoryLabel = (territoryID: string) => {
    const territory = mapTerritories.find((candidate) => candidate.id === territoryID)
    return territory ? `${territory.name} (${territory.id})` : territoryID
  }
  // Another territory named in the sheet: full label linked to the map.
  const territoryRef = (territoryID: string) => (
    <TerritoryRef
      id={territoryID}
      name={mapTerritories.find((candidate) => candidate.id === territoryID)?.name}
    />
  )
  const activeRegionEffects = selectedRegion
    ? (state.activeRegionEffects ?? []).filter(
        (effect) => effect.regionSeed === selectedRegion.seed,
      )
    : []
  const controller = selectedState?.owner ?? null
  const controllerArms = controller ? armsByOwner.get(controller) : undefined
  const army = selectedState?.army ?? null
  const armyArms = army ? armsByOwner.get(army.owner) : undefined
  const currentOrder =
    selectedChain && selectedChain.visibility !== 'hidden'
      ? selectedChain.orders?.[selectedChain.currentIndex ?? 0]
      : undefined
  const settlementLine = [
    selectedCapitalPlayer
      ? t('app.capitalOf', {
          player: displayOwner(selectedCapitalPlayer.id, selectedCapitalPlayer.name),
        })
      : null,
    isFiefCapital ? t('app.cityBonus') : null,
  ].filter(Boolean)
  const hasDetails = Boolean(
    selectedRegion ||
    activeRegionEffects.length > 0 ||
    army ||
    (selectedState && hasSupplySource(selectedState)) ||
    selectedState?.infrastructures.some((infra) => infra.type === 'mill'),
  )

  return (
    <div className="space-y-3 text-sm" data-territory-cartouche={selectedTerritory.id}>
      <div className="flex items-start gap-2">
        {controller && controllerArms && (
          <BannerIcon
            id={`cartouche-${selectedTerritory.id}`}
            arms={controllerArms}
            crown={isFiefCapital ? selectedFief?.title : undefined}
          />
        )}
        <div className="min-w-0 flex-1">
          <h2 className="text-lg font-semibold italic leading-tight text-ink-900">
            {selectedTerritory.name} <CodeTag code={selectedTerritory.id} />
          </h2>
          <p className="text-xs text-ink-500">
            {t(TERRAIN_LABEL_KEYS[selectedTerritory.terrain])}
            {' · '}
            {displayOwner(controller, t('app.noOwner'))}
            {occupied && (
              <span className="text-vermilion-700">
                {' · '}
                {t('app.occupiedBy', {
                  player: displayOwner(army?.owner, army?.owner ?? ''),
                })}
              </span>
            )}
          </p>
        </div>
        {onClose && (
          <button
            type="button"
            aria-label={t('map.closeCard')}
            title={t('map.closeCard')}
            className="-mt-1 -mr-1 flex size-11 shrink-0 items-center justify-center rounded-md text-ink-500 transition hover:bg-paper-100 hover:text-ink-900 lg:size-7"
            onClick={onClose}
          >
            <IconX aria-hidden="true" className="size-4" />
          </button>
        )}
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5">
        {selectedFief && (
          <>
            <dt className="text-xs text-ink-500">{t('app.fiefTitle')}</dt>
            <dd className="flex items-center gap-1 font-medium">
              <IconCrown
                aria-hidden="true"
                className="size-3.5 shrink-0 text-brass-700"
              />
              <span>
                {richTranslate(t, 'map.fiefLabel', {
                  title: t(FIEF_TITLE_KEYS[selectedFief.title]),
                  capital: territoryRef(selectedFief.capital),
                })}
              </span>
            </dd>
            <dt className="text-xs text-ink-500">{t('app.fiefHolder')}</dt>
            <dd className="font-medium">
              {selectedFiefHolder ? (
                <NobleRef code={selectedFiefHolder.code} name={selectedFiefHolder.name} />
              ) : (
                t('app.fiefVacant')
              )}
            </dd>
          </>
        )}
        {settlementLine.length > 0 && (
          <>
            <dt className="text-xs text-ink-500">{t('app.settlement')}</dt>
            <dd className="font-medium">{settlementLine.join(' · ')}</dd>
          </>
        )}
        {selectedState && (
          <>
            <dt className="text-xs text-ink-500">{t('app.resources')}</dt>
            <dd className="font-medium" data-territory-income>
              {selectedState.resources} R
              {selectedState.owner && selectedState.projectedIncome !== undefined && (
                <>
                  {' · '}
                  {selectedState.incomeDestination ? (
                    <>
                      {t('treasury.income', { amount: selectedState.projectedIncome })}
                      {selectedState.incomeDestination !== selectedState.id && (
                        <> → {territoryRef(selectedState.incomeDestination)}</>
                      )}
                    </>
                  ) : (
                    <span className="text-vermilion-700">
                      {t('app.territoryIncomeLost')}
                    </span>
                  )}
                </>
              )}
            </dd>
          </>
        )}
        {army && (
          <>
            <dt className="text-xs text-ink-500">{t('app.army')}</dt>
            <dd className="flex flex-wrap items-center gap-x-1.5 font-medium">
              {armyArms && (
                <ShieldIcon
                  id={`cartouche-army-${selectedTerritory.id}`}
                  arms={armyArms}
                  size={army.size}
                />
              )}
              <span>{displayOwner(army.owner, army.owner)}</span>
              <span className="text-xs text-ink-500">
                {t(army.size === 1 ? 'app.troop' : 'app.troops', { count: army.size })}
                {' · '}
                {selectedChain?.visibility === 'hidden' ? (
                  t('app.hiddenChain')
                ) : currentOrder ? (
                  <span className="font-mono">{formatOrderLabel(currentOrder)}</span>
                ) : (
                  t('app.noOrders')
                )}
              </span>
              {army.starving && (
                <span className="basis-full text-xs font-semibold text-vermilion-700">
                  {t('app.armyStarving')}
                </span>
              )}
            </dd>
          </>
        )}
        {presentNobles.length > 0 && (
          <>
            <dt className="text-xs text-ink-500">{t('app.noblesPresent')}</dt>
            <dd className="flex flex-wrap gap-1">
              {presentNobles.map((noble) => {
                const holder =
                  noble.status !== 'free' ? (selectedState?.army?.owner ?? null) : null
                return (
                  <Popover.Root key={noble.id}>
                    <Popover.Trigger asChild>
                      <button
                        type="button"
                        aria-label={`${noble.name} (${noble.code})`}
                        title={`${noble.name} (${noble.code})`}
                        data-noble-chip={noble.code}
                        className="flex size-11 items-center justify-center rounded-md border border-line bg-paper-50 text-base transition hover:bg-paper-100 lg:size-8"
                      >
                        <NoblePawnIcon
                          noble={noble}
                          color={playerColor(noble.owner)}
                          attributes={pawnAttributes.get(noble.code)}
                        />
                      </button>
                    </Popover.Trigger>
                    <Popover.Portal>
                      <Popover.Content
                        align="start"
                        sideOffset={6}
                        className="z-50 w-64 max-w-[calc(100vw-1.5rem)] space-y-2 rounded-md border border-line bg-paper-50 p-3 text-sm shadow-panel"
                      >
                        <p className="flex items-center justify-between gap-2">
                          <NobleRef
                            code={noble.code}
                            name={noble.name}
                            className="font-semibold"
                          />
                          <span
                            className={`shrink-0 text-xs ${noble.status === 'dungeon' ? 'text-brass-700' : 'text-success-700'}`}
                          >
                            {t(`orders.nobleStatus.${noble.status}` as MessageKey)}
                          </span>
                        </p>
                        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs text-ink-500">
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
                            if (
                              fiefTitles.length === 0 &&
                              religious.length === 0 &&
                              !excommunicated
                            )
                              return null
                            return (
                              <>
                                <dt>{t('app.titles')}</dt>
                                <dd
                                  className="font-medium text-ink-700"
                                  data-noble-titles
                                >
                                  {[
                                    ...religious.map((held) =>
                                      t(`lineage.religious.${held.title}` as MessageKey, {
                                        seat: held.seat ?? '',
                                      }),
                                    ),
                                    ...fiefTitles.map(
                                      (fief) =>
                                        `${t(FIEF_TITLE_KEYS[fief.title])} · ${fief.capital}`,
                                    ),
                                  ].join(', ')}
                                  {excommunicated && (
                                    <span
                                      className="text-vermilion-700"
                                      data-excommunicated
                                    >
                                      {religious.length + fiefTitles.length > 0
                                        ? ' · '
                                        : ''}
                                      {t('lineage.excommunicated')}
                                    </span>
                                  )}
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
                                      state.nobles.find((other) => other.code === code)
                                        ?.name ??
                                      state.deceased?.find((other) => other.code === code)
                                        ?.name
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
                                {t(
                                  noble.sex === 'female' ? 'app.wifeOf' : 'app.husbandOf',
                                  {
                                    spouse: noble.spouse,
                                  },
                                )}
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
                      </Popover.Content>
                    </Popover.Portal>
                  </Popover.Root>
                )
              })}
            </dd>
          </>
        )}
        {selectedState && selectedState.infrastructures.length > 0 && (
          <>
            <dt className="text-xs text-ink-500">{t('app.infrastructure')}</dt>
            <dd className="font-medium">
              {selectedState.infrastructures
                .map(
                  (infrastructure) =>
                    `${t(INFRASTRUCTURE_LABEL_KEYS[infrastructure.type])} ${infrastructure.level}`,
                )
                .join(' · ')}
            </dd>
          </>
        )}
      </dl>

      {hasDetails && (
        <details className="group border-t border-line-soft pt-2" data-territory-details>
          <summary className="cursor-pointer text-xs font-semibold text-ink-700">
            {t('app.details')}
          </summary>
          <div className="mt-2 space-y-3">
            {selectedRegion && (
              <p className="text-xs text-ink-700">
                <span className="text-ink-500">{t('app.region')} </span>
                <span>{selectedRegion.seed}</span>
                <span className="ml-2 text-ink-500">
                  {t('app.regionSeedLabel', { seed: selectedRegion.seed })}
                </span>
              </p>
            )}
            {selectedFief?.projectedIncome !== undefined && (
              <p className="text-xs text-ink-700">
                <span className="text-ink-500">{t('app.fiefProjectedIncome')} </span>
                {t('app.fiefProjectedIncomeAmount', {
                  amount: selectedFief.projectedIncome,
                })}
              </p>
            )}
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
            {selectedState?.infrastructures.map((infrastructure, index) =>
              infrastructure.type === 'mill' && selectedState.millDestination ? (
                <p key={`mill-${index}`} className="text-xs text-ink-700">
                  {selectedState.millDestination === selectedState.id
                    ? t('app.millKeptInPlace')
                    : richTranslate(t, 'app.millDestination', {
                        destination: territoryRef(selectedState.millDestination),
                      })}
                </p>
              ) : null,
            )}
            {army && (
              <div className="rounded-md bg-paper-100 px-3 py-2">
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
            )}
            {selectedState && (
              <>
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
                      <p className="mt-1 text-success-700">
                        {t('app.localRationsSufficient')}
                      </p>
                    ) : selectedSupplyLine?.source ? (
                      <>
                        <p className="mt-1">
                          {t('app.sourceLabel')}{' '}
                          <strong>
                            {sourceTerritory ? (
                              <TerritoryRef
                                id={sourceTerritory.id}
                                name={sourceTerritory.name}
                              />
                            ) : (
                              territoryRef(selectedSupplyLine.source)
                            )}
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
                      <p className="mt-1 text-vermilion-700">
                        {t('app.noAccessibleSource')}
                      </p>
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
                      <p className="mt-1 italic text-ink-500">
                        {t('app.noSupplyWinter')}
                      </p>
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
                              {territoryRef(transferLine.source)}
                            </dd>
                            <dt className="text-ink-500">{t('app.transferTarget')}</dt>
                            <dd className="font-medium">
                              {territoryRef(transferLine.target)}
                            </dd>
                            {transferLine.reachable &&
                              transferLine.distance !== undefined && (
                                <>
                                  <dt className="text-ink-500">
                                    {t('app.distanceLabel')}
                                  </dt>
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
              </>
            )}
          </div>
        </details>
      )}
    </div>
  )
}
