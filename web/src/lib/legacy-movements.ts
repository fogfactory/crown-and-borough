import { orderTargets } from '@/lib/order-label'
import type { ArmyMovement, OrderReport, TurnReport } from '@/types'

/**
 * TEMPORARY. Reports stored before the engine computed `movements` only carry
 * `combats` and `orders`: this rebuilds the same list from them, so the old
 * turns stay readable. Delete this file, its test and its one call in
 * `ReportPanel` once those reports are migrated.
 *
 * Limits of the old data: a general combat cannot tell a fight from an
 * unopposed move (every attack on an empty territory reads as a conquest),
 * and a viewer sees the origin only of the orders it knows.
 */
export function legacyMovements(report: TurnReport): ArmyMovement[] {
  const movements: ArmyMovement[] = []
  const orders = (report.orders ?? []).filter(
    (order) => order.visibility !== 'hidden' && order.army,
  )
  const orderOf = new Map<string, OrderReport>()
  for (const order of orders) {
    if (order.army && order.source && !orderOf.has(order.army))
      orderOf.set(order.army, order)
  }
  const covered = new Set<string>()

  for (const combat of report.combats ?? []) {
    if (combat.visibility === 'general') {
      if (
        combat.outcome !== 'attack_wins' &&
        combat.outcome !== 'defense_holds' &&
        combat.outcome !== 'standoff'
      ) {
        continue
      }
      const won = combat.outcome === 'attack_wins'
      movements.push({
        visibility: 'general',
        kind: won ? 'victory' : 'failure',
        legs: [{ to: combat.territory }],
        combat: combat.territory,
        reason: won ? undefined : combat.outcome,
      })
      continue
    }
    // A lone attacker faced no one: that is a move, read from its order below.
    if ((combat.contenders?.length ?? 0) < 2) continue
    for (const contender of combat.contenders ?? []) {
      if (contender.defender || !contender.army) continue
      const won = combat.winner === contender.army
      covered.add(contender.army)
      movements.push({
        visibility: 'exact',
        kind: won ? 'victory' : 'failure',
        army: contender.army,
        owner: contender.owner,
        legs: [{ from: orderOf.get(contender.army)?.source, to: combat.territory }],
        combat: combat.territory,
        reason: won ? undefined : combat.standoff ? 'standoff' : combat.reason,
      })
    }
  }

  for (const order of orders) {
    if (!order.army || covered.has(order.army)) continue
    if (order.type !== 'attack' && order.type !== 'maneuver') continue
    const targets = orderTargets(order)
    if (targets.length === 0) continue
    const won = order.outcome === 'success'
    covered.add(order.army)
    movements.push({
      visibility: 'exact',
      kind: won ? 'move' : 'failure',
      army: order.army,
      owner: order.owner,
      legs: targets.map((to) => ({ from: order.source, to })),
      reason: won ? undefined : order.reason,
    })
  }
  return movements
}
