import { NEUTRAL_PLAYER_ID } from '@/types'
import type { Player, PlayerId, WinterInvestmentReport, WinterStockReport } from '@/types'

/** The nature of a winter order, to range a house's orders under a heading. */
export type WinterGroup = 'domain' | 'titles' | 'court' | 'other'

const WINTER_GROUP_ORDER: WinterGroup[] = ['domain', 'titles', 'court', 'other']

export function winterGroupOf(kind: string): WinterGroup {
  switch (kind) {
    case 'recruit':
    case 'build':
    case 'upgrade':
    case 'fortify':
    case 'capital_elected':
    case 'noble_transfer':
    case 'transfer':
    case 'prosperity_founded':
      return 'domain'
    case 'fief_founded':
    case 'fief_assigned':
    case 'fief_auto_assigned':
    case 'fief_annexed':
    case 'fief_rank_changed':
      return 'titles'
    case 'marriage_refused':
    case 'claim':
    case 'dignity':
    case 'ritual':
    case 'inquiry':
    case 'trial_filed':
    case 'noble_draw':
    case 'noble_discard':
      return 'court'
    default:
      return 'other'
  }
}

/** The key of the winter events no house ordered (villages founded on free land). */
const REALM_KEY = ''

/** A stock the winter conservation changed (halved, or emptied). */
export const isHalved = (stock: WinterStockReport): boolean =>
  stock.stockBefore !== stock.stockAfter

/** Sorts house keys: the viewer's first, then the players' order, neutral, the realm last. */
export function houseRank(key: string, players: Player[], viewerId?: PlayerId): number {
  if (key === viewerId) return -1
  if (key === REALM_KEY) return players.length + 2
  if (key === NEUTRAL_PLAYER_ID) return players.length + 1
  const index = players.findIndex((player) => player.id === key)
  return index < 0 ? players.length : index
}

/** What a winter block stands for: a house, the neutral stocks, or no house at all. */
export type WinterHouseKind = 'player' | 'neutral' | 'realm'

export interface WinterHouse {
  key: string
  kind: WinterHouseKind
  /** The house's orders, ranged by nature; groups without an order are left out. */
  groups: Array<{ group: WinterGroup; orders: WinterInvestmentReport[] }>
  orderCount: number
  failed: number
  /** What the orders cost; a gift sent to another house counts, its sender pays it. */
  cost: number
  /** The stocks the conservation changed. */
  halved: WinterStockReport[]
  /** Stocks at the start of the winter, and once conserved. */
  before: number
  after: number
  /** What other houses sent to these stocks. */
  received: number
  /** What the orders left of the stocks, before the conservation: before − cost + received. */
  left: number
}

/**
 * The winter orders and stocks of a report, one summary per house: the
 * viewer's first, then the players, the neutral stocks and the realm. Reads
 * the flat winter report only, so a stored report reads the same. A transfer
 * credits its target with exactly its amount.
 */
export function groupWinterReport(
  investments: WinterInvestmentReport[],
  stocks: WinterStockReport[],
  players: Player[],
  viewerId?: PlayerId,
): WinterHouse[] {
  const houses = new Map<
    string,
    { orders: WinterInvestmentReport[]; owned: WinterStockReport[] }
  >()
  const house = (key: string) => {
    let entry = houses.get(key)
    if (!entry) houses.set(key, (entry = { orders: [], owned: [] }))
    return entry
  }
  investments.forEach((order) => house(order.player || REALM_KEY).orders.push(order))
  stocks.forEach((stock) => house(stock.owner || NEUTRAL_PLAYER_ID).owned.push(stock))

  const sentTo = new Map<string, number>()
  investments
    .filter((order) => order.kind === 'transfer' && order.outcome === 'success')
    .forEach((order) => {
      if (order.target) sentTo.set(order.target, (sentTo.get(order.target) ?? 0) + (order.amount ?? 0))
    })

  const sum = (values: number[]) => values.reduce((total, value) => total + value, 0)
  return Array.from(houses.entries())
    .filter(([, { orders, owned }]) => orders.length > 0 || owned.some(isHalved))
    .sort(([a], [b]) => houseRank(a, players, viewerId) - houseRank(b, players, viewerId))
    .map(([key, { orders, owned }]) => {
      const cost = sum(orders.map((order) => order.cost))
      const before = sum(owned.map((stock) => stock.stockBefore))
      const received = sum(owned.map((stock) => sentTo.get(stock.territory) ?? 0))
      return {
        key,
        kind: key === REALM_KEY ? 'realm' : key === NEUTRAL_PLAYER_ID ? 'neutral' : 'player',
        groups: WINTER_GROUP_ORDER.map((group) => ({
          group,
          orders: orders.filter((order) => winterGroupOf(order.kind) === group),
        })).filter((entry) => entry.orders.length > 0),
        orderCount: orders.length,
        failed: orders.filter((order) => order.outcome !== 'success').length,
        cost,
        halved: owned.filter(isHalved),
        before,
        after: sum(owned.map((stock) => stock.stockAfter)),
        received,
        left: before - cost + received,
      }
    })
}
