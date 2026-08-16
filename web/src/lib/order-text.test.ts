import { describe, expect, it } from 'vitest'

import { addNobleHeader, hasChainContent } from '@/lib/order-text'

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
})
