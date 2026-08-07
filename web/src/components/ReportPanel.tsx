import type { MapData, TurnReport } from '@/types'

interface ReportPanelProps {
  report: TurnReport | null
  map: MapData | null
}

function territoryLabel(map: MapData | null, id?: string): string {
  if (!id) return '—'
  const territory = map?.territories.find((candidate) => candidate.id === id)
  return territory ? territory.code : id
}

function emptyMessage(label: string) {
  return <p className="text-sm italic text-[#806f57]">Aucun événement de {label}.</p>
}

export function ReportPanel({ report, map }: ReportPanelProps) {
  if (!report) return null

  return (
    <section className="space-y-4 border-t border-[#b7a786]/50 pt-5">
      <div>
        <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-[#a84632]">
          Rapport du tour {report.header.turn}
        </p>
        <h3 className="font-serif text-xl font-semibold">Résolution terminée</h3>
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">Réception des chaînes</h4>
        {report.receptions.length === 0 ? emptyMessage('réception') : (
          <div className="space-y-1 text-sm">
            {report.receptions.map((reception, index) => (
              <div key={`${reception.player}-${reception.noble}-${index}`} className="flex items-center justify-between rounded-md bg-[#f3ead9] px-3 py-2">
                <span>{reception.player} · {reception.noble}</span>
                <span className={reception.received ? 'text-[#376341]' : 'text-[#a84632]'}>
                  {reception.received ? 'Reçue' : `Perdue${reception.reason ? ` · ${reception.reason}` : ''}`}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">Combats</h4>
        {report.combats.length === 0 ? emptyMessage('combat') : (
          <div className="overflow-x-auto rounded-md border border-[#b7a786]/60">
            <table className="w-full text-left text-xs">
              <thead className="bg-[#f3ead9] text-[#806f57]"><tr><th className="px-2 py-2">Case</th><th className="px-2 py-2">Issue</th><th className="px-2 py-2">Forces</th></tr></thead>
              <tbody>
                {report.combats.map((combat, index) => (
                  <tr key={`${combat.territory}-${index}`} className="border-t border-[#b7a786]/40">
                    <td className="px-2 py-2 font-semibold">{territoryLabel(map, combat.territory)}</td>
                    <td className="px-2 py-2">{combat.standoff ? 'Statu quo' : combat.winner ? `Victoire ${combat.winner}` : combat.reason}</td>
                    <td className="px-2 py-2">{combat.contenders.map((contender) => `${contender.force}${contender.defender ? ' défense' : ''}`).join(' · ') || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">Ravitaillement</h4>
        {report.supply.length === 0 && report.famines.length === 0 ? emptyMessage('ravitaillement') : (
          <div className="space-y-1 text-sm">
            {report.supply.map((supply) => (
              <div key={supply.source} className="flex items-center justify-between rounded-md bg-[#f3ead9] px-3 py-2">
                <span>{territoryLabel(map, supply.source)} · {supply.owner || 'Neutre'}</span>
                <span className="text-xs text-[#806f57]">{supply.production} produits · {supply.demand} demandés · {supply.stockConsumed} stock consommé</span>
              </div>
            ))}
            {report.famines.map((famine, index) => (
              <div key={`${famine.army}-${index}`} className="rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-xs text-[#8d321e]">
                Famine de {famine.army} à {territoryLabel(map, famine.territory)} ({famine.troops} troupes){famine.savedByPillage ? ' · sauvée par pillage' : ''}
              </div>
            ))}
          </div>
        )}
      </div>

      {report.winter && (
        <div className="space-y-2">
          <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">Hiver</h4>
          <div className="space-y-1 text-sm">
            {report.winter.investments.map((investment, index) => (
              <div key={`${investment.kind}-${investment.player}-${index}`} className="flex items-center justify-between rounded-md bg-[#f3ead9] px-3 py-2">
                <span>{investment.player} · {investment.kind}</span>
                <span className="text-xs text-[#806f57]">{territoryLabel(map, investment.territory)}{investment.reason ? ` · ${investment.reason}` : ''}</span>
              </div>
            ))}
            {report.winter.stocks.map((stock) => (
              <div key={stock.territory} className="flex items-center justify-between rounded-md bg-[#e8f1e3] px-3 py-2 text-xs text-[#376341]">
                {territoryLabel(map, stock.territory)} · conservation {stock.stockBefore} → {stock.stockAfter}
              </div>
            ))}
            {report.winter.investments.length === 0 && report.winter.stocks.length === 0 && emptyMessage('hiver')}
          </div>
        </div>
      )}

      <div className="space-y-2">
        <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">Ordres exécutés</h4>
        {report.orders.length === 0 ? emptyMessage('ordre') : (
          <div className="space-y-1 text-xs">
            {report.orders.map((order, index) => (
              <div key={`${order.army}-${order.order}-${index}`} className="flex items-center justify-between rounded-md bg-[#f3ead9] px-3 py-2">
                <span>{order.army} · {order.type} · {territoryLabel(map, order.source)}</span>
                <span className={order.outcome === 'success' ? 'text-[#376341]' : 'text-[#a84632]'}>{order.outcome}{order.reason ? ` · ${order.reason}` : ''}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
