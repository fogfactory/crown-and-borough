import { messages, type MessageKey, type Translate } from '@/i18n/messages'
import { ApiError } from '@/lib/api'

/**
 * Order rejections list their lines in `errors`, and the server words them in
 * the requested language: their message carries the detail the code lacks.
 */
function hasLocalizedDetail(error: ApiError): boolean {
  const errors = (error.payload as { errors?: unknown } | null)?.errors
  return Array.isArray(errors) && errors.length > 0
}

/**
 * The translated message of an API error (`error.api.<code>`, whose messages
 * are written for players: the server ones are English diagnostics). Unknown
 * codes fall back to the HTTP status.
 */
export function apiErrorMessage(error: ApiError, t: Translate): string {
  if (hasLocalizedDetail(error)) return error.message
  const key = `error.api.${error.code}`
  if (key in messages.en) return t(key as MessageKey)
  return t('error.requestFailed', { status: error.status })
}

/** Any thrown value as player-facing text: API errors are translated. */
export function errorText(error: unknown, t: Translate, fallback: string): string {
  if (error instanceof ApiError) return apiErrorMessage(error, t)
  if (error instanceof Error) return error.message
  return fallback
}
