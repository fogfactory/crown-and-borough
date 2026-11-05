import type { Infrastructure } from '@/types'

/** A ruin (pillaged castle, unfortified level-0 village) as the server
 * flags it: it shows no level. */
export function isRuined(infrastructure: Pick<Infrastructure, 'ruined'>): boolean {
  return infrastructure.ruined === true
}

/** Lowest level of an infrastructure type, as in the engine's
 * InfraType.MinLevel: 0 for a village (a ruin), 1 for every other type. */
export function minLevel(type: Infrastructure['type']): number {
  return type === 'village' ? 0 : 1
}

/** A marker shows its level once it rises above its type's minimum: villages
 * from level 1, every other infrastructure from level 2. */
export function showsLevelBadge(infrastructure: Pick<Infrastructure, 'type' | 'level'>): boolean {
  return infrastructure.level > minLevel(infrastructure.type)
}
