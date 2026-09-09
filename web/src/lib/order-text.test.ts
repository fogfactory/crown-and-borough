import { describe, expect, it } from 'vitest'

import { addNobleHeader, hasChainContent, stripNobleHeader } from '@/lib/order-text'

describe('order text helpers', () => {
  it('adds the noble trigram before a chain', () => {
    expect(addNobleHeader('JEA', 'H A B')).toBe('JEA\nH A B')
  })

  it('replaces an already entered noble header without duplicating it', () => {
    expect(addNobleHeader('JEA', '  jea # old header\nH A B')).toBe('JEA\nH A B')
  })

  it('does not submit a draft containing only the automatic header', () => {
    expect(hasChainContent('JEA', addNobleHeader('JEA', ''))).toBe(false)
    expect(hasChainContent('JEA', addNobleHeader('JEA', 'H A B'))).toBe(true)
  })

  it('strips the noble header when rehydrating a chain', () => {
    expect(stripNobleHeader('JEA', 'JEA\nH A B')).toBe('H A B')
    expect(stripNobleHeader('JEA', '  jea # header\nH A B\nB A C')).toBe('H A B\nB A C')
    expect(stripNobleHeader('JEA', 'H A B')).toBe('H A B')
    expect(stripNobleHeader('JEA', 'JEA')).toBe('')
  })
})
