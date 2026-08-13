import { formatOrderLabel } from '@/lib/order-label'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import type {
  InfraType,
  MapData,
  Outcome,
  Player,
  PlayerId,
  ReportArmy,
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
    map?.territories.find((candidate) => candidate.id === id)?.code ??
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
    type: reportOrder.type,
    position: territoryLabel(map, reportOrder.source, t),
    targets: targets?.map((target) => territoryLabel(map, target, t)),
    nobleAssignments: reportOrder.nobleAssignments,
    liaison: reportOrder.liaison,
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

export function ReportPanel({ report, map, players }: ReportPanelProps) {
  const { t } = useLanguage()
  if (!report) return null

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
        {report.receptions.length === 0 ? (
          emptyMessage(t('reports.receptions').toLowerCase(), t)
        ) : (
          <div className="space-y-1 text-sm">
            {report.receptions.map((reception, index) => (
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
        {report.combats.length === 0 ? (
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
                {report.combats.map((combat, index) => (
                  <tr
                    key={`${combat.territory}-${index}`}
                    className="border-t border-[#b7a786]/40"
                  >
                    <td className="px-2 py-2 font-semibold">
                      {territoryLabel(map, combat.territory, t)}
                    </td>
                    <td className="px-2 py-2">
                      {combat.standoff
                        ? t('reports.standoff')
                        : combat.winner
                          ? t('reports.victory', {
                              army: armyDescription(report, map, combat.winner, t),
                            })
                          : (reportReason(combat.reason, t) ?? combat.reason)}
                    </td>
                    <td className="px-2 py-2">
                      {combat.contenders
                        .map(
                          (contender) =>
                            `${contender.force}${contender.defender ? ` ${t('reports.defense')}` : ''}${
                              contender.nobleBonus
                                ? t('reports.nobleBonus', { bonus: contender.nobleBonus })
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
          {t('reports.supply')}
        </h4>
        {report.supply.length === 0 && report.famines.length === 0 ? (
          emptyMessage(t('reports.supply').toLowerCase(), t)
        ) : (
          <div className="space-y-1 text-sm">
            {report.supply.map((supply) => (
              <div
                key={supply.source}
                className="flex items-center justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
              >
                <span>
                  {territoryLabel(map, supply.source, t)} ·{' '}
                  {supply.owner || t('reports.neutral')}
                </span>
                <span className="text-right text-xs text-[#806f57]">
                  {t('reports.produced', { count: supply.production })} ·{' '}
                  {t('reports.demanded', { count: supply.demand })} ·{' '}
                  {t('reports.stockConsumed', { count: supply.stockConsumed })} ·{' '}
                  {t('reports.stockRemaining', { count: supply.stockAfter })}
                </span>
              </div>
            ))}
            {report.famines.map((famine, index) => (
              <div
                key={`${famine.army}-${index}`}
                className="rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-xs text-[#8d321e]"
              >
                {t('reports.famine', {
                  owner: famine.owner,
                  territory: territoryLabel(map, famine.territory, t),
                  troops: famine.troops,
                })}
                {famine.savedByPillage ? t('reports.savedByPillage') : ''}
                {(famine.troopsLost ?? 0) > 0
                  ? t(
                      famine.troopsLost === 1
                        ? 'reports.lostTroop'
                        : 'reports.lostTroops',
                      { count: famine.troopsLost ?? 0 },
                    )
                  : ''}
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
            {report.winter.investments.map((investment, index) => (
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
            {report.winter.stocks.map((stock) => (
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
            {report.winter.investments.length === 0 &&
              report.winter.stocks.length === 0 &&
              emptyMessage(t('reports.winter').toLowerCase(), t)}
          </div>
        </div>
      )}

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          {t('reports.ordersExecuted')}
        </h4>
        {report.orders.length === 0 ? (
          emptyMessage(t('reports.ordersExecuted').toLowerCase(), t)
        ) : (
          <div className="space-y-1 text-xs">
            {report.orders.map((order, index) => (
              <div
                key={`${order.chain}-${order.order}-${index}`}
                className="rounded-md bg-[#f3ead9] px-3 py-2"
              >
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
                  <span className={`shrink-0 text-right ${outcomeClass(order.outcome)}`}>
                    {outcomeLabel(order.outcome, t)}
                    {order.reason ? (
                      <span className="block max-w-36 text-[10px] leading-tight">
                        {reportReason(order.reason, t)}
                      </span>
                    ) : null}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
