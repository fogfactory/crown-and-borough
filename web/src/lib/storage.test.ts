import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { useLocalStorageState, useLocalStorageText } from '@/lib/storage'

const KEY = 'cb.test.toggle'

beforeEach(() => {
  window.localStorage.clear()
})

describe('useLocalStorageState', () => {
  it('defaults to the initial value when nothing is stored', () => {
    const { result } = renderHook(() => useLocalStorageState(KEY, true))

    expect(result.current[0]).toBe(true)
  })

  it('reads an existing stored value', () => {
    window.localStorage.setItem(KEY, 'false')
    const { result } = renderHook(() => useLocalStorageState(KEY, true))

    expect(result.current[0]).toBe(false)
  })

  it('persists updates to localStorage', () => {
    const { result } = renderHook(() => useLocalStorageState(KEY, true))

    act(() => result.current[1](false))

    expect(result.current[0]).toBe(false)
    expect(window.localStorage.getItem(KEY)).toBe('false')
  })
})

describe('useLocalStorageText', () => {
  it('reads, persists and clears a stored string', () => {
    window.localStorage.setItem(KEY, 'game-1')
    const { result } = renderHook(() => useLocalStorageText(KEY))
    expect(result.current[0]).toBe('game-1')

    act(() => result.current[1]('game-2'))
    expect(window.localStorage.getItem(KEY)).toBe('game-2')

    act(() => result.current[1](null))
    expect(result.current[0]).toBeNull()
    expect(window.localStorage.getItem(KEY)).toBeNull()
  })
})
