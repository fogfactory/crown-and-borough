import { deployedVersion } from '@/components/VersionBadge'
import { REPOSITORY_URL } from '@/lib/repository'

/** Calls kept for a report: the most recent ones, oldest dropped first. */
export const API_CALL_BUFFER_SIZE = 30

/** Total length of the pre-filled GitHub URL, which browsers and GitHub cap near 8 kB. */
export const GITHUB_URL_LIMIT = 8000

export const MASKED = '[masked]'

interface RawApiCall {
  at: number
  method: string
  path: string
  status: number
}

export interface ApiCallRecord {
  at: string
  method: string
  path: string
  /** 0 when the request never got a response. */
  status: number
}

const apiCalls: RawApiCall[] = []

// Keep in sync with sensitiveKey and secretPatterns in internal/api/bugs.go.
const SENSITIVE_KEY =
  /token|authorization|password|secret|api[-_]?key|cookie|invite|credential/i
const SENSITIVE_QUERY = /token|key|auth|secret|password|code|sig/i
const SECRET_IN_TEXT = [
  /bearer\s+[A-Za-z0-9._~+/=-]+/gi,
  /eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]*/g,
  /\bgh[pousr]_[A-Za-z0-9]{20,}/g,
  /\bgithub_pat_[A-Za-z0-9_]{20,}/g,
]

/** Masks authentication material in free text. */
export function maskText(text: string): string {
  return SECRET_IN_TEXT.reduce((acc, pattern) => acc.replace(pattern, MASKED), text)
}

/** Deep copy of a JSON value with sensitive keys and token-like strings masked. */
export function maskSecrets(value: unknown): unknown {
  if (typeof value === 'string') return maskText(value)
  if (Array.isArray(value)) return value.map(maskSecrets)
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value).map(([key, entry]) => [
        key,
        SENSITIVE_KEY.test(key) ? MASKED : maskSecrets(entry),
      ]),
    )
  }
  return value
}

function maskPath(path: string): string {
  const [base, query] = path.split('?', 2)
  if (!query) return maskText(base)
  const params = new URLSearchParams(query)
  for (const key of [...params.keys()]) {
    if (SENSITIVE_QUERY.test(key)) params.set(key, MASKED)
  }
  return maskText(`${base}?${decodeURIComponent(params.toString())}`)
}

/**
 * Records one API call in the circular buffer. Only the method, path and
 * status are kept: never headers (Authorization) nor bodies. Masking is
 * deferred to capture time to keep the per-request cost to a push.
 */
export function recordApiCall(method: string, path: string, status: number): void {
  apiCalls.push({ at: Date.now(), method, path, status })
  if (apiCalls.length > API_CALL_BUFFER_SIZE) apiCalls.shift()
}

export function recentApiCalls(): ApiCallRecord[] {
  return apiCalls.map((call) => ({
    at: new Date(call.at).toISOString(),
    method: call.method.toUpperCase(),
    path: maskPath(call.path),
    status: call.status,
  }))
}

export function clearApiCalls(): void {
  apiCalls.length = 0
}

export interface BugContext {
  version: string
  userAgent: string
  language: string
  page: string
  capturedAt: string
  recentApiCalls: ApiCallRecord[]
}

/** Browser side capture; the server attaches the player's game projection itself. */
export function captureBugContext(language: string): BugContext {
  return {
    version: deployedVersion(),
    userAgent: maskText(navigator.userAgent),
    language,
    page: maskPath(window.location.pathname),
    capturedAt: new Date().toISOString(),
    recentApiCalls: recentApiCalls(),
  }
}

/** Game id of the page being viewed, when it is a game page. */
export function currentGameId(pathname: string): string | null {
  const match = /^\/games\/([^/]+)/.exec(pathname)
  return match ? decodeURIComponent(match[1]) : null
}

/** What the player sees in the preview and copies for GitHub: the capture plus their game view. */
export function contextJson(context: BugContext, game: unknown): string {
  return JSON.stringify({ ...context, game: game ? maskSecrets(game) : null }, null, 2)
}

/** Summary pre-filled in the GitHub issue body; the full JSON is pasted by the player. */
export function githubSummary(
  description: string,
  context: BugContext,
  pasteHint: string,
): string {
  return [
    maskText(description),
    '',
    `- Version: ${context.version}`,
    `- Page: ${context.page}`,
    `- User agent: ${context.userAgent}`,
    '',
    pasteHint,
    '',
  ].join('\n')
}

/** `issues/new` URL with title and summary, trimmed so it stays under the length cap. */
export function githubIssueUrl(title: string, summary: string): string {
  const build = (body: string) =>
    `${REPOSITORY_URL}/issues/new?${new URLSearchParams({
      title,
      body,
      labels: 'bug',
    }).toString()}`
  let body = summary
  let url = build(body)
  while (url.length > GITHUB_URL_LIMIT && body.length > 0) {
    const excess = url.length - GITHUB_URL_LIMIT
    body = body.slice(0, Math.max(0, body.length - Math.max(excess / 3, 20)))
    url = build(`${body}…`)
  }
  return url
}
