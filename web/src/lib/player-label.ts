import type { Player, PlayerId } from '@/types'

export type PlayerName = Pick<Player, 'id' | 'name'>

export function playerDisplayName(
  playerID: PlayerId | null | undefined,
  playerLists: readonly (readonly PlayerName[])[],
  fallback: string,
): string {
  if (!playerID) return fallback

  let placeholder: string | undefined
  for (const players of playerLists) {
    const player = players.find((candidate) => candidate.id === playerID)
    if (!player) continue
    if (player.name.trim() && player.name !== player.id) return player.name
    placeholder ??= player.name
  }

  return placeholder || fallback
}
