import type { Order } from '@/types'

/** Transfer targets drafted from `territoryID`, from the server-parsed orders. */
export function transferTargetsForTerritory(
  draftOrders: Record<string, Order[]>,
  territoryID: string | null,
): string[] {
  if (!territoryID) return []

  const targets = new Set<string>()
  for (const orders of Object.values(draftOrders)) {
    for (const order of orders) {
      if (order.type !== 'transfer' || order.position !== territoryID) continue
      const target = order.targets?.[0]
      if (target) targets.add(target)
    }
  }
  return [...targets]
}

/** Indexes a preview's parsed chains by noble code. */
export function draftOrdersByNoble(
  chains: Array<{ noble: string; orders: Order[] }> | undefined,
): Record<string, Order[]> {
  return Object.fromEntries((chains ?? []).map((chain) => [chain.noble, chain.orders]))
}
