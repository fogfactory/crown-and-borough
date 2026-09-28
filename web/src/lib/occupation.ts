import type { TerritoryState } from '@/types'

/**
 * Whether territory is occupied against its controller: it has an owner and
 * an army whose owner differs from that owner (a NEUTRAL revolt included),
 * mirroring the engine's occupiedAgainstController (titres.md, #196). The
 * single seam every "occupied" display (map hatch, ownership tooltip,
 * territory detail) reads through, so they agree on what "occupied" means.
 */
export function isOccupiedAgainstController(
  territory: Pick<TerritoryState, 'owner' | 'army'> | null | undefined,
): boolean {
  return Boolean(
    territory?.owner && territory.army && territory.army.owner !== territory.owner,
  )
}
