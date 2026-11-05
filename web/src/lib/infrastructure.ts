import type { Infrastructure } from '@/types'

/** A level-0 village is a ruin: it produces nothing and shows no level. */
export function isRuinedVillage(infrastructure: Pick<Infrastructure, 'type' | 'level'>): boolean {
  return infrastructure.type === 'village' && infrastructure.level === 0
}

/**
 * Villages show their level from 1 up; every other infrastructure shows it
 * from level 2, level 1 being the default.
 */
export function showsLevelBadge(infrastructure: Pick<Infrastructure, 'type' | 'level'>): boolean {
  return infrastructure.type === 'village' ? infrastructure.level >= 1 : infrastructure.level > 1
}
