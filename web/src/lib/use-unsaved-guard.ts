import { useEffect } from 'react'

/**
 * Warns before unsaved work is lost: closing or reloading the tab (the
 * browser's own prompt) and following an in-app link to another page, such as
 * another game (a confirmation with `message`).
 */
export function useUnsavedGuard(active: boolean, message: string): void {
  useEffect(() => {
    if (!active) return
    const beforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      // Some browsers only prompt when a return value is set.
      event.returnValue = ''
    }
    const linkClick = (event: MouseEvent) => {
      if (
        event.defaultPrevented ||
        event.button !== 0 ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      ) {
        return
      }
      const anchor = (event.target as Element | null)?.closest?.('a[href]')
      if (!anchor || anchor.hasAttribute('download')) return
      const target = anchor.getAttribute('target')
      if (target && target !== '_self') return
      const url = new URL((anchor as HTMLAnchorElement).href, window.location.href)
      if (
        url.origin !== window.location.origin ||
        (url.pathname === window.location.pathname && url.search === window.location.search)
      ) {
        return
      }
      if (!window.confirm(message)) {
        event.preventDefault()
        event.stopPropagation()
      }
    }
    window.addEventListener('beforeunload', beforeUnload)
    document.addEventListener('click', linkClick, true)
    return () => {
      window.removeEventListener('beforeunload', beforeUnload)
      document.removeEventListener('click', linkClick, true)
    }
  }, [active, message])
}
