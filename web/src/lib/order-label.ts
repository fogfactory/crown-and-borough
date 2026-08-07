import type { LiaisonMode, OrderType } from '@/types'

export interface OrderLabelData {
  type: OrderType
  position: string
  targets?: string[]
  nobleTargets?: string[]
  nobleAssignments?: Record<string, string[]>
  liaison?: LiaisonMode
}

const ORDER_SYMBOLS: Record<OrderType, string> = {
  attack: 'A',
  support: 'S',
  hold: 'H',
  join: 'J',
  pillage: 'P',
  disperse: 'D',
  hostage: 'O',
  dungeon: 'K',
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
    case 'hostage':
    case 'dungeon':
      parts = [order.position, symbol, order.nobleTargets?.[0] ?? '—']
      break
    case 'disperse':
      parts = [
        order.position,
        symbol,
        ...targets.map((target) => disperseTargetLabel(target, order.nobleAssignments)),
      ]
      break
    default:
      parts = [order.position, symbol, targets[0] ?? '—']
      break
  }

  const label = parts.join(' ')
  return order.liaison === 'loop' ? `(${label})` : label
}
