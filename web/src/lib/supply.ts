import type { InfraType, TerritoryState } from '@/types'

const SUPPLY_SOURCE_TYPES: InfraType[] = ['castle', 'village']

export function hasSupplySource(territory: TerritoryState | undefined): boolean {
  const infrastructure = territory?.infrastructures[0]
  return Boolean(
    territory?.owner &&
    infrastructure &&
    SUPPLY_SOURCE_TYPES.includes(infrastructure.type),
  )
}
