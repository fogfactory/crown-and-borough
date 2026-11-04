import { apiRequest, type TokenProvider } from '@/lib/api'
import type { BugContext } from '@/lib/bug-report'

export async function fetchBugReportsEnabled(tokens: TokenProvider): Promise<boolean> {
  try {
    const result = await apiRequest<{ enabled?: boolean }>(tokens, '/api/bugs')
    return result.enabled === true
  } catch {
    return false
  }
}

/** The player's own game view, as the game page shows it (nothing more than they can already see). */
export async function fetchGameView(
  tokens: TokenProvider,
  gameId: string,
): Promise<unknown> {
  try {
    return await apiRequest<unknown>(
      tokens,
      `/api/games/${encodeURIComponent(gameId)}/state`,
    )
  } catch {
    return null
  }
}

export async function sendBugReport(
  tokens: TokenProvider,
  report: {
    title: string
    description: string
    gameId: string | null
    context: BugContext
  },
): Promise<string> {
  const result = await apiRequest<{ url: string }>(tokens, '/api/bugs', {
    method: 'POST',
    body: JSON.stringify(report),
  })
  return result.url
}
