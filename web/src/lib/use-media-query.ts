import { useSyncExternalStore } from 'react'

/** Breakpoint from which the game screen shows the drawer (Tailwind `lg`). */
export const DESKTOP_QUERY = '(min-width: 1024px)'

const lists = new Map<string, MediaQueryList>()
let listsSource: typeof window.matchMedia | null = null

/** One shared list per query, or null without `matchMedia`. */
function mediaList(query: string): MediaQueryList | null {
  if (typeof window === 'undefined' || !window.matchMedia) return null
  // A replaced `matchMedia` (tests stub it) invalidates the cached lists.
  if (listsSource !== window.matchMedia) {
    lists.clear()
    listsSource = window.matchMedia
  }
  let list = lists.get(query)
  if (!list) {
    list = window.matchMedia(query)
    lists.set(query, list)
  }
  return list
}

function subscribe(query: string, onChange: () => void): () => void {
  const list = mediaList(query)
  if (!list) return () => {}
  list.addEventListener('change', onChange)
  return () => list.removeEventListener('change', onChange)
}

/**
 * Whether a media query matches, kept in sync with the window. Without
 * `matchMedia` (tests, server rendering) it answers `fallback`.
 */
export function useMediaQuery(query: string, fallback = true): boolean {
  return useSyncExternalStore(
    (onChange) => subscribe(query, onChange),
    () => mediaList(query)?.matches ?? fallback,
    () => fallback,
  )
}

/** Whether the game screen uses its desktop layout (drawer, bookmarks). */
export function useIsDesktop(): boolean {
  return useMediaQuery(DESKTOP_QUERY)
}
