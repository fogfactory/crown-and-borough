import { vi } from 'vitest'

/**
 * Makes `useIsDesktop` answer false (a phone) until `vi.unstubAllGlobals()`.
 * Without a stub, jsdom has no `matchMedia` and the game screen is desktop.
 */
export function stubPhoneViewport() {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  }))
}
