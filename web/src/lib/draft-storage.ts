/**
 * Unsent order drafts survive a page reload (the "new version available"
 * prompt asks for one). They are kept in localStorage per game and player,
 * tagged with the turn they were written for.
 */
interface StoredDrafts {
  turn: number
  chains: Record<string, string>
  winter: string
  special: string
}

const key = (gameId: string, playerId: string) => `cb.drafts.${gameId}.${playerId}`

export function loadDrafts(gameId: string, playerId: string, turn: number): StoredDrafts | null {
  try {
    const raw = window.localStorage.getItem(key(gameId, playerId))
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<StoredDrafts>
    if (parsed.turn !== turn) return null
    return {
      turn,
      chains: Object.fromEntries(
        Object.entries(parsed.chains ?? {}).filter(
          (entry): entry is [string, string] => typeof entry[1] === 'string',
        ),
      ),
      winter: typeof parsed.winter === 'string' ? parsed.winter : '',
      special: typeof parsed.special === 'string' ? parsed.special : '',
    }
  } catch {
    return null
  }
}

/** Writes the drafts, or removes the entry when they are all empty. */
export function saveDrafts(gameId: string, playerId: string, drafts: StoredDrafts): void {
  try {
    const empty =
      drafts.winter.trim() === '' &&
      drafts.special.trim() === '' &&
      Object.values(drafts.chains).every((text) => text.trim() === '')
    if (empty) window.localStorage.removeItem(key(gameId, playerId))
    else window.localStorage.setItem(key(gameId, playerId), JSON.stringify(drafts))
  } catch {
    // Storage unavailable: drafts stay in memory only.
  }
}
