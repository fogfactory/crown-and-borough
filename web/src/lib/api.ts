export interface TokenProvider {
  getIdToken: () => Promise<string | null>
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly payload: unknown

  constructor(status: number, code: string, message: string, payload?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.payload = payload
  }
}

interface ErrorPayload {
  code?: string
  message?: string
}

function isJsonResponse(response: Response): boolean {
  return response.headers.get('content-type')?.includes('application/json') ?? false
}

export async function apiRequest<T>(
  tokenProvider: TokenProvider,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const token = await tokenProvider.getIdToken()
  if (!token) {
    throw new ApiError(401, 'unauthorized', 'an authenticated actor is required')
  }

  const headers = new Headers(init.headers)
  headers.set('Authorization', `Bearer ${token}`)
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(path, { ...init, headers })
  const payload = isJsonResponse(response)
    ? ((await response.json().catch(() => null)) as unknown)
    : await response.text()

  if (!response.ok) {
    const errorPayload = (payload ?? {}) as ErrorPayload
    throw new ApiError(
      response.status,
      errorPayload.code ?? 'request_failed',
      errorPayload.message ?? `request failed (${response.status})`,
      payload,
    )
  }

  return payload as T
}

export async function apiTextRequest(
  tokenProvider: TokenProvider,
  path: string,
): Promise<string> {
  const token = await tokenProvider.getIdToken()
  if (!token) {
    throw new ApiError(401, 'unauthorized', 'an authenticated actor is required')
  }

  const response = await fetch(path, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const text = await response.text()
  if (!response.ok) {
    throw new ApiError(response.status, 'request_failed', text)
  }
  return text
}
