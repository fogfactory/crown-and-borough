import { describe, expect, it } from 'vitest'

import { parseWinterDraft } from '@/lib/winter-parse'

describe('parseWinterDraft', () => {
  it('parses every winter order family', () => {
    expect(
      parseWinterDraft(
        'R T XXX\nR N XXX\nC M YYY\nC C YYY\nC D YYY\nE C YYY\nL N NNN\nO N NNN\nP N NNN\nG XXX YYY 4',
      ),
    ).toEqual([
      { line: 1, type: 'recruit_troop', territory: 'XXX' },
      { line: 2, type: 'recruit_noble', territory: 'XXX' },
      { line: 3, type: 'build', territory: 'YYY', infrastructure: 'mill' },
      { line: 4, type: 'build', territory: 'YYY', infrastructure: 'castle' },
      { line: 5, type: 'build', territory: 'YYY', infrastructure: 'supply_depot' },
      { line: 6, type: 'elect_capital', territory: 'YYY' },
      { line: 7, type: 'liberate_noble', noble: 'NNN' },
      { line: 8, type: 'hostage', noble: 'NNN' },
      { line: 9, type: 'dungeon', noble: 'NNN' },
      { line: 10, type: 'transfer', source: 'XXX', target: 'YYY', amount: 4 },
    ])
  })

  it('normalizes case and ignores comments and malformed lines', () => {
    expect(
      parseWinterDraft(
        '# comment\n\nr t ros # recruit\nC X ROS\nG ROS BRU 0\nG ROS BRU 2',
      ),
    ).toEqual([
      { line: 3, type: 'recruit_troop', territory: 'ROS' },
      { line: 6, type: 'transfer', source: 'ROS', target: 'BRU', amount: 2 },
    ])
  })
})
