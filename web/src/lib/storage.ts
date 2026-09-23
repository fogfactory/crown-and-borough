import { useState } from 'react'

export function useLocalStorageState(
  key: string,
  initialValue: boolean,
): [boolean, (value: boolean) => void] {
  const [value, setValue] = useState<boolean>(() => {
    try {
      const stored = window.localStorage.getItem(key)
      return stored === null ? initialValue : stored === 'true'
    } catch {
      return initialValue
    }
  })

  const setStored = (next: boolean) => {
    setValue(next)
    try {
      window.localStorage.setItem(key, String(next))
    } catch {
      return
    }
  }

  return [value, setStored]
}

/**
 * Keeps an optional string in localStorage. Storage errors (private mode,
 * disabled storage) fall back to in-memory state.
 */
export function useLocalStorageText(
  key: string,
): [string | null, (value: string | null) => void] {
  const [value, setValue] = useState<string | null>(() => {
    try {
      return window.localStorage.getItem(key)
    } catch {
      return null
    }
  })

  const setStored = (next: string | null) => {
    setValue(next)
    try {
      if (next === null) {
        window.localStorage.removeItem(key)
      } else {
        window.localStorage.setItem(key, next)
      }
    } catch {
      return
    }
  }

  return [value, setStored]
}
