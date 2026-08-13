import { formatOrderLabel } from '@/lib/order-label'
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

const OUTCOME_LABELS: Record<Outcome, string> = {
  success: 'Réussi',
  failure: 'Échec',
  invalid: 'Invalidé',
}

const WINTER_INFRA_SYMBOLS: Partial<Record<InfraType, string>> = {
  mill: 'M',
  castle: 'C',
  supply_depot: 'D',
  village: 'V',
}

function territoryLabel(map: MapData | null, id?: string): string {
  if (!id) return '—'
  return (
    map?.territories.find((candidate) => candidate.id === id)?.code ??
    'Territoire inconnu'
  )
}

function playerLabel(players: Player[], playerId?: PlayerId): string {
  if (!playerId) return 'Joueur inconnu'
  return players.find((player) => player.id === playerId)?.name ?? playerId
}

function playerColor(players: Player[], playerId?: PlayerId): string {
  return players.find((player) => player.id === playerId)?.color ?? '#b7a786'
}

function playerMarker(players: Player[], playerId?: PlayerId) {
  return (
    <span
      role="img"
      className="inline-block size-2.5 shrink-0 rounded-full border border-[#30291f]/30 shadow-inner"
      style={{ backgroundColor: playerColor(players, playerId) }}
      aria-label={`Couleur de ${playerLabel(players, playerId)}`}
    />
  )
}

function outcomeClass(outcome: Outcome): string {
  return outcome === 'success' ? 'text-[#376341]' : 'text-[#a84632]'
}

function outcomeLabel(outcome: Outcome): string {
  return OUTCOME_LABELS[outcome]
}

function emptyMessage(label: string) {
  return <p className="text-sm italic text-[#806f57]">Aucun événement de {label}.</p>
}

function armyDescription(
  report: TurnReport,
  map: MapData | null,
  armyID: string,
): string {
  const army: ReportArmy | undefined = report.players
    .flatMap((player) => player.armies)
    .find((candidate) => candidate.id === armyID)
  if (!army) return 'armée inconnue'
  return `armée de ${army.owner} à ${territoryLabel(map, army.territory)}`
}

function formatReportOrderLabel(
  reportOrder: TurnReport['orders'][number],
  map: MapData | null,
): string {
  const targets =
    reportOrder.targets ?? (reportOrder.target ? [reportOrder.target] : undefined)
  return formatOrderLabel({
    type: reportOrder.type,
    position: territoryLabel(map, reportOrder.source),
    targets: targets?.map((target) => territoryLabel(map, target)),
    nobleAssignments: reportOrder.nobleAssignments,
    liaison: reportOrder.liaison,
  })
}

function winterOrderLabel(order: WinterOrder, map: MapData | null): string {
  const territory = territoryLabel(map, order.territory)
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
): string {
  if (investment.order) return winterOrderLabel(investment.order, map)
  const territory = territoryLabel(map, investment.territory)
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
      return 'Ordre d’hiver'
  }
}

function winterDetails(investment: WinterInvestmentReport, map: MapData | null): string {
  if (investment.reason) return investment.reason
  if (investment.nobleName) return investment.nobleName
  if (investment.level) return `Niveau ${investment.level}`
  return territoryLabel(map, investment.territory)
}

export function ReportPanel({ report, map, players }: ReportPanelProps) {
  if (!report) return null

  return (
    <section className="min-w-0 space-y-4">
      <div>
        <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-[#a84632]">
          Rapport du tour {report.header.turn}
        </p>
        <h3 className="font-serif text-xl font-semibold">Résolution terminée</h3>
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          Réception des chaînes
        </h4>
        {report.receptions.length === 0 ? (
          emptyMessage('réception')
        ) : (
          <div className="space-y-1 text-sm">
            {report.receptions.map((reception, index) => (
              <div
                key={`${reception.player}-${reception.noble}-${index}`}
                className="flex items-center justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
              >
                <span className="flex min-w-0 items-center gap-2">
                  {playerMarker(players, reception.player)}
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
                    ? 'Reçue'
                    : `Perdue${reception.reason ? ` · ${reception.reason}` : ''}`}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          Combats
        </h4>
        {report.combats.length === 0 ? (
          emptyMessage('combat')
        ) : (
          <div className="overflow-x-auto rounded-md border border-[#b7a786]/60">
            <table className="w-full min-w-[24rem] text-left text-xs">
              <thead className="bg-[#f3ead9] text-[#806f57]">
                <tr>
                  <th className="px-2 py-2">Case</th>
                  <th className="px-2 py-2">Issue</th>
                  <th className="px-2 py-2">Forces</th>
                </tr>
              </thead>
              <tbody>
                {report.combats.map((combat, index) => (
                  <tr
                    key={`${combat.territory}-${index}`}
                    className="border-t border-[#b7a786]/40"
                  >
                    <td className="px-2 py-2 font-semibold">
                      {territoryLabel(map, combat.territory)}
                    </td>
                    <td className="px-2 py-2">
                      {combat.standoff
                        ? 'Statu quo'
                        : combat.winner
                          ? `Victoire · ${armyDescription(report, map, combat.winner)}`
                          : combat.reason}
                    </td>
                    <td className="px-2 py-2">
                      {combat.contenders
                        .map(
                          (contender) =>
                            `${contender.force}${contender.defender ? ' défense' : ''}`,
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
          Ravitaillement
        </h4>
        {report.supply.length === 0 && report.famines.length === 0 ? (
          emptyMessage('ravitaillement')
        ) : (
          <div className="space-y-1 text-sm">
            {report.supply.map((supply) => (
              <div
                key={supply.source}
                className="flex items-center justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
              >
                <span>
                  {territoryLabel(map, supply.source)} · {supply.owner || 'Neutre'}
                </span>
                <span className="text-right text-xs text-[#806f57]">
                  {supply.production} produits · {supply.demand} demandés ·{' '}
                  {supply.stockConsumed} stock consommé · {supply.stockAfter} R en stock
                </span>
              </div>
            ))}
            {report.famines.map((famine, index) => (
              <div
                key={`${famine.army}-${index}`}
                className="rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-xs text-[#8d321e]"
              >
                Famine de l&apos;armée de {famine.owner} à{' '}
                {territoryLabel(map, famine.territory)} ({famine.troops} troupes)
                {famine.savedByPillage ? ' · sauvée par pillage' : ''}
                {(famine.troopsLost ?? 0) > 0
                  ? ` · perd ${famine.troopsLost} troupe${famine.troopsLost === 1 ? '' : 's'}`
                  : ''}
              </div>
            ))}
          </div>
        )}
      </div>

      {report.winter && (
        <div className="space-y-2">
          <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
            Hiver
          </h4>
          <div className="space-y-1 text-sm">
            {report.winter.investments.map((investment, index) => (
              <div
                key={`${investment.kind}-${investment.player}-${index}`}
                className="flex items-start justify-between gap-3 rounded-md bg-[#f3ead9] px-3 py-2"
              >
                <span className="flex min-w-0 items-start gap-2">
                  {playerMarker(players, investment.player)}
                  <span>
                    <strong className="font-mono text-xs">
                      {investmentLabel(investment, map)}
                    </strong>
                    <span className="mt-1 block text-xs text-[#806f57]">
                      {investment.player} · {winterDetails(investment, map)}
                    </span>
                  </span>
                </span>
                <span className={`shrink-0 text-xs ${outcomeClass(investment.outcome)}`}>
                  {outcomeLabel(investment.outcome)}
                  {investment.outcome === 'success' && (
                    <span className="mt-1 block text-right text-[10px] text-[#806f57]">
                      {investment.cost > 0 ? `coût : ${investment.cost} R` : 'aucun coût'}
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
                <span>{territoryLabel(map, stock.territory)} · conservation</span>
                <span>
                  {stock.stockBefore} → {stock.stockAfter}
                </span>
              </div>
            ))}
            {report.winter.investments.length === 0 &&
              report.winter.stocks.length === 0 &&
              emptyMessage('hiver')}
          </div>
        </div>
      )}

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
          Ordres exécutés
        </h4>
        {report.orders.length === 0 ? (
          emptyMessage('ordre')
        ) : (
          <div className="space-y-1 text-xs">
            {report.orders.map((order, index) => (
              <div
                key={`${order.chain}-${order.order}-${index}`}
                className="rounded-md bg-[#f3ead9] px-3 py-2"
              >
                <div className="flex items-start justify-between gap-3">
                  <span className="flex min-w-0 items-start gap-2">
                    {playerMarker(players, order.owner)}
                    <span className="min-w-0">
                      <strong className="font-mono text-sm">
                        {formatReportOrderLabel(order, map)}
                      </strong>
                      <span className="mt-1 block text-[#806f57]">
                        {order.owner || 'Joueur inconnu'} · Noble{' '}
                        {order.noble || 'inconnu'}
                      </span>
                    </span>
                  </span>
                  <span className={`shrink-0 text-right ${outcomeClass(order.outcome)}`}>
                    {outcomeLabel(order.outcome)}
                    {order.reason ? (
                      <span className="block max-w-36 text-[10px] leading-tight">
                        {order.reason}
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
