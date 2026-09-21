import { formatOrderLabel } from '@/lib/order-label'
import { formatCardLabel } from '@/lib/card-hand'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import type {
  InfraType,
  MapData,
  Outcome,
  Player,
  PlayerId,
  ReportArmy,
  CardReport,
  SeasonEffectReport,
  TurnReport,
  WinterInvestmentReport,
  WinterOrder,
} from '@/types'

interface ReportPanelProps {
  report: TurnReport | null
  map: MapData | null
  players: Player[]
}

const WINTER_INFRA_SYMBOLS: Partial<Record<InfraType, string>> = {
  mill: 'M',
  castle: 'C',
  supply_depot: 'D',
  village: 'V',
}

const OUTCOME_KEYS: Record<Outcome, MessageKey> = {
  success: 'reports.outcome.success',
  failure: 'reports.outcome.failure',
  invalid: 'reports.outcome.invalid',
}

const REASON_KEYS: Record<string, MessageKey> = {
  insufficient_resources: 'reports.reason.insufficient_resources',
  transferred: 'reports.reason.transferred',
  transferred_partially: 'reports.reason.transferred_partially',
  famished_sender: 'reports.reason.famished_sender',
  transfer_over_capacity: 'reports.reason.transfer_over_capacity',
  transfer_path_blocked: 'reports.reason.transfer_path_blocked',
  invalid_transfer_destination: 'reports.reason.invalid_transfer_destination',
  transfer_source_not_controlled: 'reports.reason.transfer_source_not_controlled',
  transfer_same_territory: 'reports.reason.transfer_same_territory',
  transfer_source_not_settlement: 'reports.reason.transfer_source_not_settlement',
  transfer_target_not_settlement: 'reports.reason.transfer_target_not_settlement',
  invalid_transfer_amount: 'reports.reason.invalid_transfer_amount',
  territory_not_controlled: 'reports.reason.territory_not_controlled',
  noble_requires_owned_army: 'reports.reason.noble_requires_owned_army',
  noble_requires_settlement: 'reports.reason.noble_requires_settlement',
  troop_requires_adjacent_noble: 'reports.reason.troop_requires_adjacent_noble',
  noble_not_prisoner: 'reports.reason.noble_not_prisoner',
  noble_not_held: 'reports.reason.noble_not_held',
  no_capital: 'reports.reason.no_capital',
  no_army_at_capital: 'reports.reason.no_army_at_capital',
  structure_present: 'reports.reason.structure_present',
  mill_requires_productive_neighbor: 'reports.reason.mill_requires_productive_neighbor',
  capital_requires_controlled_castle: 'reports.reason.capital_requires_controlled_castle',
  attack_wins: 'reports.reason.attack_wins',
  defense_holds: 'reports.reason.defense_holds',
  standoff: 'reports.reason.standoff',
  non_adjacent_destination: 'reports.reason.non_adjacent_destination',
  allied_destination: 'reports.reason.allied_destination',
  combat_lost: 'reports.reason.combat_lost',
  dislodged: 'reports.reason.dislodged',
  support_cut: 'reports.reason.support_cut',
  support_void: 'reports.reason.support_void',
  join_move: 'reports.reason.join_move',
  join_attack_arrival: 'reports.reason.join_attack_arrival',
  enemy_destination: 'reports.reason.enemy_destination',
  disperse_partial: 'reports.reason.disperse_partial',
  disperse_complete: 'reports.reason.disperse_complete',
  disperse_no_residual: 'reports.reason.disperse_no_residual',
  disperse_noble_left_behind: 'reports.reason.disperse_noble_left_behind',
  no_retreat_destination: 'reports.reason.no_retreat_destination',
  retreat_collision: 'reports.reason.retreat_collision',
  position_mismatch: 'reports.reason.position_mismatch',
  missing_pending_disperse: 'reports.reason.missing_pending_disperse',
  missing_disperse_residual: 'reports.reason.missing_disperse_residual',
  join_not_terminal: 'reports.reason.join_not_terminal',
  invalid_hold_shape: 'reports.reason.invalid_hold_shape',
  invalid_pillage_shape: 'reports.reason.invalid_pillage_shape',
  no_infrastructure: 'reports.reason.no_infrastructure',
  unknown_order_type: 'reports.reason.unknown_order_type',
  invalid_target_shape: 'reports.reason.invalid_target_shape',
  invalid_support_shape: 'reports.reason.invalid_support_shape',
  unknown_support_target: 'reports.reason.unknown_support_target',
  invalid_defensive_support: 'reports.reason.invalid_defensive_support',
  invalid_offensive_support: 'reports.reason.invalid_offensive_support',
  non_adjacent_disperse_destination: 'reports.reason.non_adjacent_disperse_destination',
  invalid_disperse_assignment_destination:
    'reports.reason.invalid_disperse_assignment_destination',
  duplicate_disperse_wildcard: 'reports.reason.duplicate_disperse_wildcard',
  invalid_disperse_noble_assignment: 'reports.reason.invalid_disperse_noble_assignment',
  join_convergence: 'reports.reason.join_convergence',
  join_enemy_convergence: 'reports.reason.join_enemy_convergence',
  attacked_destination: 'reports.reason.attacked_destination',
  join_host: 'reports.reason.join_host',
  join_pair: 'reports.reason.join_pair',
  disperse_friendly_fusion: 'reports.reason.disperse_friendly_fusion',
  support_applied: 'reports.reason.support_applied',
  unresolved_order: 'reports.reason.unresolved_order',
  unknown_infrastructure: 'reports.reason.unknown_infrastructure',
  pillaged: 'reports.reason.pillaged',
  army_destroyed: 'reports.reason.army_destroyed',
  held: 'reports.reason.held',
  disperse_summary: 'reports.reason.disperse_summary',
  invalid_winter_order: 'reports.reason.invalid_winter_order',
  unknown_territory: 'reports.reason.unknown_territory',
  territory_occupied_by_other_player: 'reports.reason.territory_occupied_by_other_player',
  invalid_infrastructure: 'reports.reason.invalid_infrastructure',
  unknown_noble: 'reports.reason.unknown_noble',
  mill_max_level_reached: 'reports.reason.mill_max_level_reached',
  bad_weather: 'reports.reason.bad_weather',
  disperse_residual_dislodged: 'reports.reason.disperse_residual_dislodged',
  invalid_transfer_shape: 'reports.reason.invalid_transfer_shape',
  no_available_first_name: 'reports.reason.no_available_first_name',
}

const RECEPTION_REASON_KEYS: Record<string, MessageKey> = {
  'error.assignment.no_army': 'reports.reason.reception.noArmy',
  'error.assignment.army_not_owned': 'reports.reason.reception.armyNotOwned',
  'error.assignment.noble_dungeon': 'reports.reason.reception.nobleDungeon',
  'error.assignment.emission_capacity': 'reports.reason.reception.emissionCapacity',
  'error.assignment.noble_unknown': 'reports.reason.reception.invalid',
  'error.assignment.pending_disperse': 'reports.reason.reception.invalid',
  'error.assignment.chain_id_in_use': 'reports.reason.reception.invalid',
  'error.assignment.chain_validation': 'reports.reason.reception.invalid',
}

function territoryLabel(
  map: MapData | null,
  id: string | undefined,
  t: Translate,
): string {
  if (!id) return '—'
  return (
    map?.territories.find((candidate) => candidate.id === id)?.id ??
    t('reports.unknownTerritory')
  )
}

function playerLabel(
  players: Player[],
  playerId: PlayerId | undefined,
  t: Translate,
): string {
  if (!playerId) return t('reports.unknownPlayer')
  return players.find((player) => player.id === playerId)?.name ?? playerId
}

function playerColor(players: Player[], playerId?: PlayerId): string {
  return players.find((player) => player.id === playerId)?.color ?? '#b7a786'
}

function playerMarker(players: Player[], playerId: PlayerId | undefined, t: Translate) {
  return (
    <span
      role="img"
      className="inline-block size-2.5 shrink-0 rounded-full border border-[#30291f]/30 shadow-inner"
      style={{ backgroundColor: playerColor(players, playerId) }}
      aria-label={t('app.colorOf', { player: playerLabel(players, playerId, t) })}
    />
  )
}

function outcomeClass(outcome: Outcome): string {
  return outcome === 'success' ? 'text-[#376341]' : 'text-[#a84632]'
}

function outcomeLabel(outcome: Outcome, t: Translate): string {
  return t(OUTCOME_KEYS[outcome])
}

function emptyMessage(label: string, t: Translate) {
  return (
    <p className="text-sm italic text-[#806f57]">{t('reports.noEvents', { label })}</p>
  )
}

function armyDescription(
  report: TurnReport,
  map: MapData | null,
  armyID: string,
  t: Translate,
): string {
  const army: ReportArmy | undefined = report.players
    .flatMap((player) => player.armies)
    .find((candidate) => candidate.id === armyID)
  if (!army) return t('reports.unknownArmy')
  return t('reports.armyDescription', {
    owner: army.owner,
    territory: territoryLabel(map, army.territory, t),
  })
}

function formatReportOrderLabel(
  reportOrder: TurnReport['orders'][number],
  map: MapData | null,
  t: Translate,
): string {
  const targets =
    reportOrder.targets ?? (reportOrder.target ? [reportOrder.target] : undefined)
  return formatOrderLabel({
    type: reportOrder.type ?? 'hold',
    position: territoryLabel(map, reportOrder.source, t),
    targets: targets?.map((target) => territoryLabel(map, target, t)),
    nobleAssignments: reportOrder.nobleAssignments,
    liaison: reportOrder.liaison ?? 'single',
    amount: reportOrder.amount,
  })
}

function winterOrderLabel(order: WinterOrder, map: MapData | null, t: Translate): string {
  const territory = territoryLabel(map, order.territory, t)
  switch (order.type) {
    case 'recruit_noble':
      return `R N ${territory}`
    case 'recruit_troop':
      return `R T ${territory}`
    case 'build':
      return `C ${WINTER_INFRA_SYMBOLS[order.infrastructureType ?? 'mill'] ?? '?'} ${territory}`
    case 'elect_capital':
      return `E C ${territory}`
    case 'liberate_noble':
      return `L N ${order.nobleCode ?? '—'}`
    case 'hostage':
      return `O N ${order.nobleCode ?? '—'}`
    case 'dungeon':
      return `P N ${order.nobleCode ?? '—'}`
    case 'transfer':
      return `G ${territoryLabel(map, order.source, t)} ${territoryLabel(map, order.target, t)} ${order.amount ?? '—'}`
  }
}

function investmentLabel(
  investment: WinterInvestmentReport,
  map: MapData | null,
  t: Translate,
): string {
  if (investment.order) return winterOrderLabel(investment.order, map, t)
  const territory = territoryLabel(map, investment.territory, t)
  switch (investment.kind) {
    case 'recruit':
      return investment.nobleCode ? `R N ${territory}` : `R T ${territory}`
    case 'build':
      return `C ${WINTER_INFRA_SYMBOLS[investment.type ?? 'mill'] ?? '?'} ${territory}`
    case 'upgrade':
      return `C ${WINTER_INFRA_SYMBOLS[investment.type ?? 'mill'] ?? '?'} ${territory}`
    case 'capital_elected':
      return `E C ${territory}`
    case 'liberation':
      return `L N ${investment.nobleCode ?? '—'}`
    case 'transfer':
      return `G ${territoryLabel(map, investment.source, t)} ${territoryLabel(map, investment.target, t)} ${investment.amount ?? '—'}`
    default:
      return t('reports.winterOrder')
  }
}

function reportReason(
  reason: string | undefined,
  t: Translate,
  reasonKey?: string,
  reasonArgs?: unknown[],
): string | undefined {
  if (!reason && !reasonKey) return undefined
  if (reasonKey === 'error.reception.concurrent') {
    return t('reports.reason.reception.concurrent', {
      territory: String(reasonArgs?.[0] ?? '—'),
      count: Number(reasonArgs?.[1] ?? 0),
      turn: Number(reasonArgs?.[2] ?? 0),
    })
  }
  if (reasonKey && RECEPTION_REASON_KEYS[reasonKey]) {
    const key = RECEPTION_REASON_KEYS[reasonKey]
    if (
      key === 'reports.reason.reception.noArmy' ||
      key === 'reports.reason.reception.armyNotOwned'
    ) {
      return t(key, { territory: String(reasonArgs?.[0] ?? '—') })
    }
    return t(key)
  }
  if (reason) {
    const base = reason.split(':', 1)[0]
    const key = REASON_KEYS[base]
    if (key) return t(key)
    return reason
  }
  return undefined
}

function winterDetails(
  investment: WinterInvestmentReport,
  map: MapData | null,
  t: Translate,
): string {
  if (investment.reason) return reportReason(investment.reason, t) ?? investment.reason
  if (investment.nobleName) return investment.nobleName
  if (investment.level) return t('reports.level', { level: investment.level })
  return territoryLabel(map, investment.territory, t)
}

function cardEventLabel(card: CardReport, map: MapData | null, t: Translate): string {
  const label = formatCardLabel(card.kind, t)
  const region = territoryLabel(map, card.region, t)
  const player = card.player ?? t('reports.unknownPlayer')
  switch (card.eventType) {
    case 'deck_draw':
      return t('reports.cardDrawn', { card: label, player })
    case 'deck_discard':
      return t('reports.cardDiscarded', { card: label, player })
    case 'deck_order_played':
      return t('reports.cardPlayed', { card: label, player, region })
    case 'calamity_scheduled':
      return t('reports.cardScheduled', {
        card: label,
        region,
        season: card.season ? t(SEASON_LABEL_KEYS[card.season]) : '—',
      })
    default:
      return label
  }
}

function seasonEffectLabel(effect: SeasonEffectReport, map: MapData | null, t: Translate): string {
  const region = territoryLabel(map, effect.region, t)
  const card = effect.cardKind ? formatCardLabel(effect.cardKind, t) : ''
  switch (effect.kind) {
    case 'calamity_canceled':
      return t('reports.calamityCanceled', { card, bonus: card, region })
    case 'bonus_effect':
      return t('reports.bonusEffect', { card, region })
    default:
      return card || effect.kind
  }
}

interface SeasonEffectLine {
  key: string
  owner?: PlayerId
  label: string
  /** Neutral-army lines render in a muted gray instead of the section red. */
  muted?: boolean
}

interface SeasonEffectGroup {
  key: string
  header?: string
  lines: SeasonEffectLine[]
}

function seasonEffectLine(
  effect: SeasonEffectReport,
  map: MapData | null,
  t: Translate,
  index: number,
): SeasonEffectLine {
  const region = territoryLabel(map, effect.region, t)
  const card = effect.cardKind ? formatCardLabel(effect.cardKind, t) : ''
  const key = `${effect.kind}-${effect.cardKind ?? ''}-${index}`
  const owner = effect.owner ?? t('reports.unknownPlayer')
  switch (effect.kind) {
    case 'calamity_applied':
      if (effect.army) {
        return {
          key,
          owner: effect.owner,
          label: t('reports.calamityPlagueArmy', {
            owner,
            territory: territoryLabel(map, effect.territory, t),
            before: effect.sizeBefore ?? 0,
            after: effect.sizeAfter ?? 0,
          }),
        }
      }
      return { key, label: t('reports.calamityApplied', { card, region }) }
    case 'bad_weather_blocked': {
      const base = {
        owner,
        territory: territoryLabel(map, effect.territory, t),
      }
      return {
        key,
        owner: effect.owner,
        label: effect.target
          ? t('reports.calamityBadWeatherBlocked', {
              ...base,
              target: territoryLabel(map, effect.target, t),
            })
          : t('reports.calamityBadWeatherBlockedNoTarget', base),
      }
    }
    case 'famine_loss':
      if (!effect.territory) {
        return {
          key,
          label: t('reports.calamityFamineRegion', {
            region,
            production: effect.productionLost ?? 0,
            rations: effect.rationsLost ?? 0,
          }),
        }
      }
      if ((effect.productionLost ?? 0) > 0) {
        return {
          key,
          label: t('reports.calamityFamineMill', {
            territory: territoryLabel(map, effect.territory, t),
            production: effect.productionLost ?? 0,
          }),
        }
      }
      return {
        key,
        label: t('reports.calamityFamineRations', {
          rations: effect.rationsLost ?? 0,
          territory: territoryLabel(map, effect.territory, t),
        }),
      }
    case 'plague_noble_death':
      return {
        key,
        label: t('reports.plagueDeath', {
          noble: effect.noble ?? '—',
          territory: territoryLabel(map, effect.territory, t),
        }),
      }
    case 'plague_noble_survived':
      return {
        key,
        label: t('reports.plagueSurvived', {
          noble: effect.noble ?? '—',
          territory: territoryLabel(map, effect.territory, t),
        }),
      }
    case 'famine':
      return {
        key,
        muted: true,
        label: t('reports.neutralFamine', {
          territory: territoryLabel(map, effect.territory, t),
          before: effect.sizeBefore ?? 0,
          after: effect.sizeAfter ?? 0,
        }),
      }
    case 'card_canceled': {
      const card = effect.cardKind ? formatCardLabel(effect.cardKind, t) : ''
      return {
        key,
        label: effect.territory
          ? t('reports.cardCanceled', {
              card,
              territory: territoryLabel(map, effect.territory, t),
            })
          : card,
      }
    }
    case 'neutral_army_created': {
      const count = effect.troops ?? 0
      return {
        key,
        muted: true,
        label: t(
          count === 1
            ? 'reports.neutralArmyCreatedTroop'
            : 'reports.neutralArmyCreated',
          { count, territory: territoryLabel(map, effect.territory, t) },
        ),
      }
    }
    default:
      return { key, label: seasonEffectLabel(effect, map, t) }
  }
}

function groupSeasonEffects(
  effects: SeasonEffectReport[],
  map: MapData | null,
  t: Translate,
): { flat: SeasonEffectLine[]; groups: SeasonEffectGroup[] } {
  const flat: SeasonEffectLine[] = []
  const groups = new Map<string, SeasonEffectGroup>()
  const groupFor = (effect: SeasonEffectReport): SeasonEffectGroup => {
    const key = `${effect.cardKind ?? ''}-${effect.region ?? ''}`
    let group = groups.get(key)
    if (!group) {
      group = { key, lines: [] }
      groups.set(key, group)
    }
    return group
  }
  effects.forEach((effect, index) => {
    const line = seasonEffectLine(effect, map, t, index)
    switch (effect.kind) {
      case 'calamity_applied':
        if (effect.army) {
          groupFor(effect).lines.push(line)
        } else {
          groupFor(effect).header = line.label
        }
        break
      case 'bad_weather_blocked':
      case 'famine_loss':
      case 'plague_noble_death':
      case 'plague_noble_survived':
      case 'card_canceled':
        groupFor(effect).lines.push(line)
        break
      case 'famine':
      case 'neutral_army_created':
        flat.push(line)
        break
      default:
        flat.push(line)
    }
  })
  return { flat, groups: Array.from(groups.values()) }
}

export function ReportPanel({ report, map, players }: ReportPanelProps) {
  const { t } = useLanguage()
  if (!report) return null
  const receptions = report.receptions ?? []
  const combats = report.combats ?? []
  const production = report.production ?? []
  const consumption = report.consumption ?? []
  const orders = report.orders ?? []
  const winterInvestments = report.winter?.investments ?? []
  const winterStocks = report.winter?.stocks ?? []
  const rumors = report.rumors ?? report.winter?.rumors ?? []
  const cards = report.cards ?? report.winter?.cards ?? []
  const seasonEffects = report.seasonEffects ?? []
  const seasonEffectView = groupSeasonEffects(seasonEffects, map, t)

  return (
    <section className="min-w-0 space-y-4">
      <div>
        <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-[#a84632]">
          {t('reports.title', { turn: report.header.turn })}
        </p>
        <h3 className="font-serif text-xl font-semibold">
          {t('reports.resolutionComplete')}
        </h3>
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          {t('reports.receptions')}
        </h4>
        {receptions.length === 0 ? (
          emptyMessage(t('reports.receptions').toLowerCase(), t)
        ) : (
          <div className="space-y-1 text-sm">
            {receptions.map((reception, index) => (
              <div
                key={`${reception.player}-${reception.noble}-${index}`}
                className="flex items-center justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
              >
                <span className="flex min-w-0 items-center gap-2">
                  {playerMarker(players, reception.player, t)}
                  <span>
                    {reception.player} · {reception.noble}
                  </span>
                </span>
                <span
                  className={
                    reception.received
                      ? 'shrink-0 text-[#376341]'
                      : 'shrink-0 text-[#a84632]'
                  }
                >
                  {reception.received
                    ? t('reports.receptionReceived')
                    : reception.reason
                      ? t('reports.receptionLost', {
                          reason:
                            reportReason(
                              reception.reason,
                              t,
                              reception.reasonKey,
                              reception.reasonArgs,
                            ) ?? reception.reason,
                        })
                      : t('reports.receptionLostPlain')}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          {t('reports.combats')}
        </h4>
        {combats.length === 0 ? (
          emptyMessage(t('reports.combats').toLowerCase(), t)
        ) : (
          <div className="overflow-x-auto rounded-md border border-[#b7a786]/60">
            <table className="w-full min-w-[24rem] text-left text-xs">
              <thead className="bg-[#f3ead9] text-[#806f57]">
                <tr>
                  <th className="px-2 py-2">{t('reports.square')}</th>
                  <th className="px-2 py-2">{t('reports.issue')}</th>
                  <th className="px-2 py-2">{t('reports.forces')}</th>
                </tr>
              </thead>
              <tbody>
                {combats.map((combat, index) => (
                  <tr
                    key={`${combat.territory}-${index}`}
                    className="border-t border-[#b7a786]/40"
                  >
                    <td className="px-2 py-2 font-semibold">
                      {territoryLabel(map, combat.territory, t)}
                    </td>
                    <td className="px-2 py-2">
                      {combat.visibility === 'general'
                        ? combat.summary
                        : combat.standoff
                          ? t('reports.standoff')
                          : combat.winner
                            ? t('reports.victory', {
                                army: armyDescription(report, map, combat.winner, t),
                              })
                            : (reportReason(combat.reason, t) ?? combat.reason)}
                    </td>
                    <td className="px-2 py-2">
                      {combat.visibility === 'general'
                        ? '—'
                        : (combat.contenders ?? [])
                            .map(
                              (contender) =>
                                `${contender.force ?? '—'}${contender.defender ? ` ${t('reports.defense')}` : ''}${
                                  contender.nobleBonus
                                    ? t('reports.nobleBonus', {
                                        bonus: contender.nobleBonus,
                                      })
                                    : ''
                                }`,
                            )
                            .join(' · ') || '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          {t('reports.productionTitle')}
        </h4>
        {production.length === 0 ? (
          emptyMessage(t('reports.productionTitle').toLowerCase(), t)
        ) : (
          <div className="space-y-1 text-sm">
            {production.map((line) => {
              const chips: Array<{ label: string; value: number }> = [
                { label: t('reports.productionTerrainRations'), value: line.terrainRations },
                { label: t('reports.productionInfraRations'), value: line.infraRations ?? 0 },
                { label: t('reports.productionBonusRations'), value: line.bonusRations ?? 0 },
                { label: t('reports.productionBaseProduction'), value: line.baseProduction ?? 0 },
                { label: t('reports.productionMillProduction'), value: line.millProduction ?? 0 },
                { label: t('reports.productionBonusProduction'), value: line.bonusProduction ?? 0 },
              ].filter((chip) => chip.value > 0)
              const suppressed =
                (line.suppressedRations ?? 0) + (line.suppressedProduction ?? 0)
              const sentDestinations = Object.keys(line.sentToRations ?? {}).sort()
              const hasStock =
                (line.stockBefore ?? 0) > 0 ||
                (line.stockConsumed ?? 0) > 0 ||
                (line.stockAfter ?? 0) > 0
              return (
                <div
                  key={line.territory}
                  className="rounded-md bg-[#f3ead9] px-3 py-2"
                >
                  <div className="flex items-center justify-between gap-3">
                    <span className="flex min-w-0 items-center gap-2">
                      {playerMarker(players, line.owner, t)}
                      <span className="font-mono text-xs">{line.territory}</span>
                      <span className="text-xs text-[#806f57]">
                        {territoryLabel(map, line.territory, t)}
                      </span>
                    </span>
                    <span className="shrink-0 text-xs font-semibold text-[#376341]">
                      {t('reports.productionProduced', { count: line.produced })}
                    </span>
                  </div>
                  {chips.length > 0 && (
                    <p className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px] text-[#806f57]">
                      {chips.map((chip) => (
                        <span key={chip.label}>
                          {chip.label} {chip.value}
                        </span>
                      ))}
                    </p>
                  )}
                  {suppressed > 0 && (
                    <p className="mt-1 text-[11px] font-semibold text-[#8d321e]">
                      {t('reports.productionSuppressed', { count: suppressed })}
                    </p>
                  )}
                  {(sentDestinations.length > 0 || hasStock) && (
                    <p className="mt-1 text-[11px] text-[#806f57]">
                      {sentDestinations
                        .map((destination) =>
                          t('reports.productionSent', {
                            count: line.sentToRations?.[destination] ?? 0,
                            territory: destination,
                          }),
                        )
                        .concat(
                          hasStock
                            ? [
                                t('reports.productionStockLine', {
                                  before: line.stockBefore ?? 0,
                                  consumed: line.stockConsumed ?? 0,
                                  after: line.stockAfter ?? 0,
                                }),
                              ]
                            : [],
                        )
                        .join(' · ')}
                    </p>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          {t('reports.consumptionTitle')}
        </h4>
        {consumption.length === 0 ? (
          emptyMessage(t('reports.consumptionTitle').toLowerCase(), t)
        ) : (
          <div className="space-y-1 text-sm">
            {consumption.map((line) => (
              <div
                key={line.army}
                className={`rounded-md px-3 py-2 ${
                  line.famine
                    ? 'border border-[#a84632]/30 bg-[#f8e5dd] text-xs text-[#8d321e]'
                    : 'bg-[#f3ead9] text-xs'
                }`}
              >
                <div className="flex items-center justify-between gap-3">
                  <span className="flex min-w-0 items-center gap-2">
                    {playerMarker(players, line.owner, t)}
                    <span>
                      {line.owner} · {territoryLabel(map, line.territory, t)}
                    </span>
                  </span>
                  <span className="shrink-0 text-[#806f57]">
                    {t(
                      line.size === 1 ? 'app.troop' : 'app.troops',
                      { count: line.size },
                    )}{' '}
                    · {t('reports.consumptionDemand', { count: line.demand })}
                  </span>
                </div>
                <p className="mt-1 text-[11px]">
                  {line.source
                    ? `${t('reports.consumptionSource', { source: line.source })} · `
                    : ''}
                  {t('reports.consumptionLine', {
                    local: line.receivedLocal,
                    transfer: line.receivedTransfer,
                    total: line.totalReceived,
                    missing: line.missing,
                  })}
                </p>
                {line.famine && (
                  <p className="mt-1 font-semibold">
                    {line.savedByPillage ? t('reports.savedByPillage') : ''}
                    {(line.troopsLost ?? 0) > 0
                      ? t(
                          line.troopsLost === 1
                            ? 'reports.lostTroop'
                            : 'reports.lostTroops',
                          { count: line.troopsLost ?? 0 },
                        )
                      : ''}
                    {(line.resourceCredit ?? 0) > 0
                      ? ' ' +
                        t('reports.consumptionPillageCredit', {
                          count: line.resourceCredit ?? 0,
                          territory: territoryLabel(map, line.creditTerritory, t),
                        })
                      : ''}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {report.winter && (
        <div className="space-y-2">
          <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
            {t('reports.winter')}
          </h4>
          <div className="space-y-1 text-sm">
            {winterInvestments.map((investment, index) => (
              <div
                key={`${investment.kind}-${investment.player}-${index}`}
                className="flex items-start justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
              >
                <span className="flex min-w-0 items-start gap-2">
                  {playerMarker(players, investment.player, t)}
                  <span>
                    <strong className="font-mono text-xs">
                      {investmentLabel(investment, map, t)}
                    </strong>
                    <span className="mt-1 block text-xs text-[#806f57]">
                      {investment.player} · {winterDetails(investment, map, t)}
                    </span>
                  </span>
                </span>
                <span className={`shrink-0 text-xs ${outcomeClass(investment.outcome)}`}>
                  {outcomeLabel(investment.outcome, t)}
                  {investment.outcome === 'success' && (
                    <span className="mt-1 block text-right text-[10px] text-[#806f57]">
                      {investment.cost > 0
                        ? t('reports.cost', { cost: investment.cost })
                        : t('reports.noCost')}
                    </span>
                  )}
                </span>
              </div>
            ))}
            {winterStocks.map((stock) => (
              <div
                key={stock.territory}
                className="flex items-center justify-between gap-3 rounded-md bg-[#e8f1e3] px-3 py-2 text-xs text-[#376341]"
              >
                <span>
                  {t('reports.conservation', {
                    territory: territoryLabel(map, stock.territory, t),
                  })}
                </span>
                <span>
                  {stock.stockBefore} → {stock.stockAfter}
                </span>
              </div>
            ))}
            {winterInvestments.length === 0 &&
              winterStocks.length === 0 &&
              emptyMessage(t('reports.winter').toLowerCase(), t)}
          </div>
        </div>
      )}

      {rumors.length > 0 && (
        <div className="space-y-2 rounded-lg border border-[#c8b0d9] bg-[#fbf5ff] p-3">
          <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#684b7d]">
            {t('reports.rumors')}
          </h4>
          <ul className="space-y-1 text-sm text-[#684b7d]">
            {rumors.map((rumor, index) => (
              <li key={`${rumor.key}-${index}`}>{t(rumor.key as MessageKey)}</li>
            ))}
          </ul>
        </div>
      )}

      {cards.length > 0 && (
        <div className="space-y-2 rounded-lg border border-[#c8b0d9] bg-[#fbf5ff] p-3">
          <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#684b7d]">
            {t('reports.cards')}
          </h4>
          <ol className="space-y-1 text-sm text-[#684b7d]">
            {cards.map((card, index) => (
              <li key={`${card.eventType}-${card.kind}-${index}`}>{cardEventLabel(card, map, t)}</li>
            ))}
          </ol>
        </div>
      )}

      {seasonEffects.length > 0 && (
        <div className="space-y-2 rounded-lg border border-[#e4b4a4] bg-[#fff5f0] p-3">
          <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#8d321e]">
            {t('reports.seasonEffects')}
          </h4>
          <ol className="space-y-2 text-sm text-[#8d321e]">
            {seasonEffectView.flat.map((line) => (
              <li
                key={line.key}
                className={line.muted ? 'text-[#806f57]' : undefined}
              >
                {line.label}
              </li>
            ))}
            {seasonEffectView.groups.map((group) => (
              <li key={group.key} className="space-y-1">
                {group.header && <p className="font-semibold">{group.header}</p>}
                {group.lines.length > 0 && (
                  <ul className="space-y-1 border-l-2 border-[#e4b4a4] pl-3">
                    {group.lines.map((line) => (
                      <li
                        key={line.key}
                        className={`flex items-center gap-2 ${line.muted ? 'text-[#806f57]' : ''}`}
                      >
                        {line.owner ? playerMarker(players, line.owner, t) : null}
                        <span>{line.label}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </li>
            ))}
          </ol>
        </div>
      )}

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          {t('reports.ordersExecuted')}
        </h4>
        {orders.length === 0 ? (
          emptyMessage(t('reports.ordersExecuted').toLowerCase(), t)
        ) : (
          <div className="space-y-1 text-xs">
            {orders.map((order, index) => (
              <div
                key={`${order.chain ?? 'hidden'}-${order.order ?? index}-${index}`}
                className="rounded-md bg-[#f3ead9] px-3 py-2"
              >
                {order.visibility === 'hidden' ? (
                  <div className="flex items-center justify-between gap-3">
                    <span className="italic text-[#806f57]">
                      {t('reports.hiddenOrder')}
                    </span>
                    <span className={`shrink-0 ${outcomeClass(order.outcome)}`}>
                      {outcomeLabel(order.outcome, t)}
                    </span>
                  </div>
                ) : (
                  <div className="flex items-start justify-between gap-3">
                    <span className="flex min-w-0 items-start gap-2">
                      {playerMarker(players, order.owner, t)}
                      <span className="min-w-0">
                        <strong className="font-mono text-sm">
                          {formatReportOrderLabel(order, map, t)}
                        </strong>
                        <span className="mt-1 block text-[#806f57]">
                          {order.owner || t('reports.unknownPlayer')} ·{' '}
                          {t('reports.noble', {
                            noble: order.noble || '—',
                          })}
                        </span>
                      </span>
                    </span>
                    <span
                      className={`shrink-0 text-right ${outcomeClass(order.outcome)}`}
                    >
                      {outcomeLabel(order.outcome, t)}
                      {order.reason ? (
                        <span className="block max-w-36 text-[10px] leading-tight">
                          {reportReason(order.reason, t)}
                        </span>
                      ) : null}
                    </span>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
