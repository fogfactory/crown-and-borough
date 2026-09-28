import { computeRegionOutlines, type RegionOutlineResult } from '@/lib/region-geometry'
import { pointsToPath } from '@/lib/map-svg-geometry'
import type { Fief, Region, Territory } from '@/types'

/**
 * Reuses `computeRegionOutlines` by treating every fief's territory group as
 * a pseudo-region seeded on its capital: fiefs are dynamic (state-driven)
 * and never overlap (a territory belongs to at most one fief), so the same
 * boundary-chaining algorithm as the static regions applies unchanged.
 */
export function computeFiefOutlines(
  territories: Territory[],
  fiefs: Fief[],
): RegionOutlineResult {
  const pseudoRegions: Region[] = fiefs.map((fief) => ({
    id: fief.capital,
    seed: fief.capital,
    territories: fief.territories,
  }))
  return computeRegionOutlines(territories, pseudoRegions)
}

/** Combined `d` path of every boundary loop of one fief's outline. */
export function fiefOutlinePath(outline: RegionOutlineResult, capital: string): string {
  const loops = outline.outlines.get(capital)?.loops ?? []
  return loops.map((loop) => pointsToPath(loop)).join(' ')
}
