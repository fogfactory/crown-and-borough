import { afterEach, describe, expect, it } from 'vitest'
import { loadDrafts, saveDrafts } from '@/lib/draft-storage'

afterEach(() => window.localStorage.clear())

describe('draft storage', () => {
  it('restores the drafts of the same turn', () => {
    saveDrafts('G', 'P', { turn: 3, chains: { ROS: 'M A B' }, winter: '', special: '' })
    expect(loadDrafts('G', 'P', 3)?.chains).toEqual({ ROS: 'M A B' })
  })

  it('ignores drafts written for another turn', () => {
    saveDrafts('G', 'P', { turn: 3, chains: {}, winter: 'R 1', special: '' })
    expect(loadDrafts('G', 'P', 4)).toBeNull()
  })

  it('removes the entry once all drafts are empty', () => {
    saveDrafts('G', 'P', { turn: 3, chains: {}, winter: 'R 1', special: '' })
    saveDrafts('G', 'P', { turn: 3, chains: { A: ' ' }, winter: '', special: '' })
    expect(loadDrafts('G', 'P', 3)).toBeNull()
  })
})
