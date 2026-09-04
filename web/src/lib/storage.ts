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
