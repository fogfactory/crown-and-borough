import { draftedTaxEffects, type DraftedTaxEffects } from '@/lib/projected-tax'
import type { PlayerId, StateData } from '@/types'

/** What the treasury bar sums up for a player (charte § Poste de commandement). */
export interface Treasury {
  /** Resources currently stored in the player's territories. */
  stock: number
  /** Projected income of the next action turn, drafted taxes included. */
  income: number
  /** Projected ration consumption of the next action turn; 0 in winter. */
  consumption: number
  /** Number of armies that would starve if nothing changes. */
  armiesAtRisk: number
  taxes: DraftedTaxEffects
}

/** The player's treasury, or null for a spectator or an unknown player. */
export function treasuryOf(
  state: StateData,
  playerId: PlayerId,
  specialDraft?: string,
): Treasury | null {
  const player = state.players.find((candidate) => candidate.id === playerId)
  if (!player) return null
  const taxes = draftedTaxEffects(state, playerId, specialDraft)
  const winter = state.season === 'winter'
  return {
    stock: state.territories.reduce(
      (total, territory) =>
        total + (territory.owner === playerId ? territory.resources : 0),
      0,
    ),
    income:
      (player.projectedIncome ?? 0) +
      Object.values(taxes.seigneurial).reduce((sum, extra) => sum + extra, 0) +
      (player.projectedMillIncome ?? 0) +
      (player.projectedRoyalRent ?? 0) +
      taxes.royalSupplement +
      taxes.tithe,
    consumption: winter ? 0 : (player.projectedConsumption ?? 0),
    armiesAtRisk: winter ? 0 : (player.armiesAtRisk ?? []).length,
    taxes,
  }
}
