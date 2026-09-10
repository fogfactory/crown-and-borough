import type { Infrastructure, MapData, PlayerId, StateData, WinterCosts } from '@/types'
import { parseWinterDraft, type ParsedWinterOrder } from '@/lib/winter-parse'

export interface WinterCostEstimate {
  spent: number
  available: number
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0
}

export function isWinterCosts(value: unknown): value is WinterCosts {
  if (typeof value !== 'object' || value === null) return false
  const candidate = value as Record<string, unknown>
  return (
    isNonNegativeInteger(candidate.castle) &&
    Array.isArray(candidate.millLevels) &&
    candidate.millLevels.every(isNonNegativeInteger) &&
    isNonNegativeInteger(candidate.troop) &&
    isNonNegativeInteger(candidate.noble) &&
    isNonNegativeInteger(candidate.supplyDepot) &&
    isNonNegativeInteger(candidate.liberation)
  )
}

function hasSettlement(infrastructure: Infrastructure | undefined): boolean {
  return infrastructure?.type === 'castle' || infrastructure?.type === 'village'
}

function simulatedInfrastructure(state: StateData): Map<string, Infrastructure> {
  const result = new Map<string, Infrastructure>()
  for (const territory of state.territories) {
    const infrastructure = territory.infrastructures[0]
    if (infrastructure) result.set(territory.id, { ...infrastructure })
  }
  return result
}

function territoryExists(state: StateData, territoryID: string): boolean {
  return state.territories.some((territory) => territory.id === territoryID)
}

function controlsTerritory(
  state: StateData,
  player: PlayerId,
  territoryID: string,
): boolean {
  return state.territories.some(
    (territory) => territory.id === territoryID && territory.owner === player,
  )
}

function millCanBeBuiltAt(
  territoryID: string,
  infrastructureByTerritory: Map<string, Infrastructure>,
  map: MapData | undefined,
): boolean {
  if (!map) return true
  const territory = map.territories.find((candidate) => candidate.id === territoryID)
  if (!territory) return false
  if (hasSettlement(infrastructureByTerritory.get(territoryID))) return true
  return territory.adjacencies.some((neighborID) =>
    hasSettlement(infrastructureByTerritory.get(neighborID)),
  )
}

function buildCost(
  order: ParsedWinterOrder,
  costs: WinterCosts,
  infrastructureByTerritory: Map<string, Infrastructure>,
  map: MapData | undefined,
): number {
  const territoryID = order.territory
  if (!territoryID || order.type !== 'build') return 0
  const existing = infrastructureByTerritory.get(territoryID)

  if (order.infrastructure === 'mill') {
    if (!millCanBeBuiltAt(territoryID, infrastructureByTerritory, map)) return 0
    if (!existing) {
      const cost = costs.millLevels[0]
      if (cost === undefined) return 0
      infrastructureByTerritory.set(territoryID, { type: 'mill', level: 1 })
      return cost
    }
    if (existing.type !== 'mill') return 0
    const nextLevel = existing.level + 1
    const cost = costs.millLevels[nextLevel - 1]
    if (cost === undefined) return 0
    existing.level = nextLevel
    return cost
  }

  if (order.infrastructure === 'castle') {
    if (existing && existing.type !== 'village') return 0
    infrastructureByTerritory.set(territoryID, { type: 'castle', level: 1 })
    return costs.castle
  }

  if (existing) return 0
  infrastructureByTerritory.set(territoryID, { type: 'supply_depot', level: 1 })
  return costs.supplyDepot
}

function orderCost(
  order: ParsedWinterOrder,
  state: StateData,
  player: PlayerId,
  costs: WinterCosts,
  infrastructureByTerritory: Map<string, Infrastructure>,
  map: MapData | undefined,
): number {
  switch (order.type) {
    case 'recruit_noble':
      return order.territory && controlsTerritory(state, player, order.territory)
        ? costs.noble
        : 0
    case 'recruit_troop':
      return order.territory && controlsTerritory(state, player, order.territory)
        ? costs.troop
        : 0
    case 'build':
      return order.territory && controlsTerritory(state, player, order.territory)
        ? buildCost(order, costs, infrastructureByTerritory, map)
        : 0
    case 'liberate_noble':
      return costs.liberation
    case 'transfer':
      return order.source &&
        order.target &&
        territoryExists(state, order.source) &&
        territoryExists(state, order.target)
        ? (order.amount ?? 0)
        : 0
    case 'elect_capital':
    case 'hostage':
    case 'dungeon':
      return 0
  }
}

export function estimateWinterCost(
  state: StateData,
  player: PlayerId,
  costs: WinterCosts,
  draft: string,
  map?: MapData,
): WinterCostEstimate {
  const available = state.territories
    .filter(
      (territory) =>
        territory.owner === player && hasSettlement(territory.infrastructures[0]),
    )
    .reduce((total, territory) => total + Math.max(0, territory.resources), 0)
  const infrastructureByTerritory = simulatedInfrastructure(state)
  const spent = parseWinterDraft(draft).reduce(
    (total, order) =>
      total + orderCost(order, state, player, costs, infrastructureByTerritory, map),
    0,
  )
  return { spent, available }
}
