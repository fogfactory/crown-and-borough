import { describe, expect, it } from 'vitest'

import { playerDisplayName } from '@/lib/player-label'

describe('playerDisplayName', () => {
  it('uses the display name when the state still contains a player id placeholder', () => {
    expect(
      playerDisplayName(
        'P2',
        [[{ id: 'P2', name: 'P2' }], [{ id: 'P2', name: 'Bob' }]],
        'P2',
      ),
    ).toBe('Bob')
  })

  it('falls back to the first placeholder or supplied fallback', () => {
    expect(playerDisplayName('P2', [[{ id: 'P2', name: '' }]], 'P2')).toBe('P2')
    expect(playerDisplayName(null, [], 'Nobody')).toBe('Nobody')
  })
})
