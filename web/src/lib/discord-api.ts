import { apiRequest, type TokenProvider } from '@/lib/api'
import type {
  DiscordEvent,
  DiscordLanguage,
  DiscordPreview,
  DiscordWebhookSave,
  DiscordWebhookStatus,
} from '@/types'

function webhookPath(gameId: string, suffix = ''): string {
  return `/api/games/${encodeURIComponent(gameId)}/discord-webhook${suffix}`
}

/** Host only. Answers 404 `discord_disabled` where the channel is not mounted. */
export function fetchDiscordWebhook(
  tokenProvider: TokenProvider,
  gameId: string,
): Promise<DiscordWebhookStatus> {
  return apiRequest<DiscordWebhookStatus>(tokenProvider, webhookPath(gameId))
}

export function saveDiscordWebhook(
  tokenProvider: TokenProvider,
  gameId: string,
  body: DiscordWebhookSave,
): Promise<DiscordWebhookStatus> {
  return apiRequest<DiscordWebhookStatus>(tokenProvider, webhookPath(gameId), {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export function deleteDiscordWebhook(
  tokenProvider: TokenProvider,
  gameId: string,
): Promise<DiscordWebhookStatus> {
  return apiRequest<DiscordWebhookStatus>(tokenProvider, webhookPath(gameId), {
    method: 'DELETE',
  })
}

/** Sends a test message to the stored webhook. */
export function testDiscordWebhook(
  tokenProvider: TokenProvider,
  gameId: string,
  // With a URL the test goes to that webhook (the one of the form) and
  // nothing is stored; without, to the stored webhook.
  form?: { url: string; language: DiscordLanguage },
): Promise<DiscordWebhookStatus> {
  return apiRequest<DiscordWebhookStatus>(tokenProvider, webhookPath(gameId, '/test'), {
    method: 'POST',
    ...(form ? { body: JSON.stringify(form) } : {}),
  })
}

/** Renders a template with example values; sends nothing. */
export function previewDiscordTemplate(
  tokenProvider: TokenProvider,
  gameId: string,
  event: DiscordEvent,
  language: DiscordLanguage,
  template: string,
): Promise<DiscordPreview> {
  return apiRequest<DiscordPreview>(tokenProvider, webhookPath(gameId, '/preview'), {
    method: 'POST',
    body: JSON.stringify({ event, language, template }),
  })
}
