import { playerDisplayName, type PlayerName } from '@/lib/player-label'
import type { PlayerId, StateData } from '@/types'

/** Year currently played, clamped to the configured game length once finished. */
export function internalYear(state: StateData): number {
  const year = state.year ?? Math.floor((state.turn - 1) / 4) + 1
  if (state.finished && state.yearCount && state.turn > state.yearCount * 4) {
    return state.yearCount
  }
  return year
}

export function remainingYears(state: StateData, fallbackYearCount = 10): number {
  if (state.finished) return 0
  const yearCount = state.yearCount ?? fallbackYearCount
  return Math.max(0, yearCount - internalYear(state) + 1)
}

export function remainingTurns(state: StateData, fallbackYearCount = 10): number {
  const yearCount = state.yearCount ?? fallbackYearCount
  return Math.max(0, yearCount * 4 - state.turn + 1)
}

export function ownerName(
  owner: PlayerId | null,
  state: StateData,
  preferredPlayers: readonly PlayerName[] = [],
  fallback?: string,
): string {
  return playerDisplayName(
    owner,
    [preferredPlayers, state.players],
    fallback ?? owner ?? '',
  )
}
