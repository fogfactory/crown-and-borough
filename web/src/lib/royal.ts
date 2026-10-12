import type { Noble, StateData } from '@/types'

/**
 * The noble who exercises the royal powers (politique.md "Pouvoirs royaux"):
 * the king, unless he is in a dungeon, else the queen as regent -- his wife
 * while he is held, or the dowager queen of a vacant throne. Mirrors the
 * server's GameState.RoyalIssuer.
 */
export function royalIssuerOf(state: StateData): Noble | undefined {
  const king = state.nobles.find((noble) => noble.code === state.king)
  if (king && king.status !== 'dungeon') return king
  if (king && state.queenDowager) return undefined
  const queen = state.nobles.find((noble) => noble.code === state.queen)
  return queen && queen.status !== 'dungeon' ? queen : undefined
}
