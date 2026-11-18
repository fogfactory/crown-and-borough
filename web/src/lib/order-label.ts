import { MANEUVER_REST, type LiaisonMode, type OrderType } from '@/types'

export interface OrderLabelData {
  type: OrderType
  position: string
  targets?: string[]
  counts?: number[]
  nobleAssignments?: Record<string, string[]>
  liaison?: LiaisonMode
  amount?: number
}

const ORDER_SYMBOLS: Record<OrderType, string> = {
  attack: 'A',
  support: 'S',
  hold: 'H',
  join: 'J',
  pillage: 'P',
  disperse: 'D',
  maneuver: 'M',
  transfer: 'T',
}

function disperseTargetLabel(
  target: string,
  assignments: Record<string, string[]> | undefined,
): string {
  const nobles = assignments?.[target]
  if (!nobles || nobles.length === 0) return target
  const namedNobles = nobles.filter((noble) => noble !== '*')
  return `${target}*${namedNobles.join('*')}`
}

function maneuverTargetLabel(
  target: string,
  count: number | undefined,
  assignments: Record<string, string[]> | undefined,
): string {
  // A bare territory means one troop.
  const suffix =
    count === MANEUVER_REST ? '+' : count !== undefined && count !== 1 ? `/${count}` : ''
  return disperseTargetLabel(target, assignments).replace(/^([^*]*)/, `$1${suffix}`)
}

export function formatOrderLabel(order: OrderLabelData): string {
  const symbol = ORDER_SYMBOLS[order.type]
  const targets = order.targets ?? []
  let parts: string[]

  switch (order.type) {
    case 'hold':
    case 'pillage':
      parts = [symbol, order.position]
      break
    case 'support':
      parts = [order.position, symbol, targets[0] ?? '—']
      if (targets[1]) parts.push('-', targets[1])
      break
    case 'disperse':
      parts = [
        order.position,
        symbol,
        ...targets.map((target) => disperseTargetLabel(target, order.nobleAssignments)),
      ]
      break
    case 'maneuver':
      parts = [
        order.position,
        symbol,
        ...targets.map((target, index) =>
          maneuverTargetLabel(target, order.counts?.[index], order.nobleAssignments),
        ),
      ]
      break
    case 'transfer':
      parts = [order.position, symbol, targets[0] ?? '—', String(order.amount ?? '—')]
      break
    default:
      parts = [order.position, symbol, targets[0] ?? '—']
      break
  }

  const label = parts.join(' ')
  return order.liaison === 'loop' ? `(${label})` : label
}
