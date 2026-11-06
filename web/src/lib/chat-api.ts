import { apiRequest, type TokenProvider } from '@/lib/api'
import type { ChatPostBody } from '@/lib/chat'

/**
 * `channel` is what the API expects: "public", or the other house of a
 * private conversation (never the conversation key).
 */
export function postChatMessage(
  tokenProvider: TokenProvider,
  gameId: string,
  channel: string,
  message: ChatPostBody,
): Promise<unknown> {
  return apiRequest<unknown>(
    tokenProvider,
    `/api/games/${encodeURIComponent(gameId)}/messages`,
    { method: 'POST', body: JSON.stringify({ channel, ...message }) },
  )
}

export function markChatRead(
  tokenProvider: TokenProvider,
  gameId: string,
  channel: string,
): Promise<unknown> {
  return apiRequest<unknown>(
    tokenProvider,
    `/api/games/${encodeURIComponent(gameId)}/messages/read`,
    { method: 'POST', body: JSON.stringify({ channel }) },
  )
}
