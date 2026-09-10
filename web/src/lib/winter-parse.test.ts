import { describe, expect, it } from 'vitest'

import { parseWinterDraft, parseWinterDraftDetailed } from '@/lib/winter-parse'
import type { MapData } from '@/types'

const map: MapData = {
  territories: ['ROS', 'BRU'].map((id) => ({
    id,
    name: id,
    terrain: 'plain',
    village: false,
    points: [],
    adjacencies: [],
    impassable: [],
  })),
}

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

  it('reports structural and state-reference errors without dropping valid orders', () => {
    const result = parseWinterDraftDetailed(
      'R X ROS\nC M ZZZ\nL N JEA\nL N JE\nG ROS BRU 0\nZZZ A ROS\nR T ROS BRU\nBAD\nC Q ROS\nG ROS BRU',
      { map, nobles: [{ code: 'JEA' }] },
    )

    expect(result.orders).toEqual([{ line: 3, type: 'liberate_noble', noble: 'JEA' }])
    expect(result.errors).toEqual([
      {
        line: 1,
        key: 'error.winter.unknown_subtype',
        values: { symbol: 'R', subtype: 'X' },
      },
      {
        line: 2,
        key: 'error.winter.territory_unknown',
        values: { code: 'ZZZ' },
      },
      {
        line: 4,
        key: 'error.winter.noble_code_format',
        values: { code: 'JE' },
      },
      {
        line: 5,
        key: 'error.winter.transfer_amount',
        values: { amount: '0' },
      },
      {
        line: 6,
        key: 'error.winter.unknown_symbol',
        values: { symbol: 'ZZZ' },
      },
      { line: 7, key: 'error.winter.target_only_one' },
      { line: 8, key: 'error.winter.order_shape' },
      {
        line: 9,
        key: 'error.winter.unknown_subtype',
        values: { symbol: 'C', subtype: 'Q' },
      },
      { line: 10, key: 'error.winter.transfer_shape' },
    ])
  })

  it('reports unknown nobles when a noble list is provided', () => {
    const result = parseWinterDraftDetailed('L N NNN', {
      map,
      nobles: [{ code: 'JEA' }],
    })

    expect(result.orders).toEqual([])
    expect(result.errors).toEqual([
      {
        line: 1,
        key: 'error.winter.noble_unknown',
        values: { code: 'NNN' },
      },
    ])
  })
})
