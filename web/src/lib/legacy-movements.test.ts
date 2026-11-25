import { describe, expect, it } from 'vitest'

import { legacyMovements } from '@/lib/legacy-movements'
import type { OrderReport, TurnReport } from '@/types'

const base: TurnReport = {
  header: { year: 1, season: 'spring', turn: 1 },
  players: [],
  receptions: [],
  production: [],
  consumption: [],
  combats: [],
  orders: [],
  moves: [],
  nobles: [],
}

const attack = (
  army: string,
  source: string,
  target: string,
  outcome: OrderReport['outcome'],
): OrderReport => ({
  visibility: 'known',
  army,
  owner: 'P1',
  type: 'attack',
  source,
  targets: [target],
  outcome,
})

describe('legacyMovements', () => {
  it('rebuilds conquests, moves and failures from combats and orders', () => {
    const movements = legacyMovements({
      ...base,
      combats: [
        {
          visibility: 'exact',
          territory: 'BRU',
          contenders: [
            { army: 'A1', owner: 'P1', force: 3, defender: false },
            { army: 'A2', owner: 'P2', force: 2, defender: true },
          ],
          winner: 'A1',
        },
        {
          visibility: 'exact',
          territory: 'CHA',
          contenders: [{ army: 'A3', owner: 'P1', force: 1, defender: false }],
          winner: 'A3',
        },
        { visibility: 'general', territory: 'VAL', outcome: 'standoff' },
      ],
      orders: [
        attack('A1', 'ROS', 'BRU', 'success'),
        attack('A3', 'ROS', 'CHA', 'success'),
        {
          visibility: 'known',
          army: 'A4',
          owner: 'P1',
          type: 'maneuver',
          source: 'ROS',
          targets: ['BRU'],
          outcome: 'failure',
          reason: 'dislodged',
        },
        {
          visibility: 'known',
          army: 'A5',
          owner: 'P1',
          type: 'hold',
          source: 'ROS',
          outcome: 'success',
        },
        { visibility: 'hidden', outcome: 'success' },
      ],
    })
    expect(
      movements.map((movement) => [movement.army, movement.kind, movement.visibility]),
    ).toEqual([
      ['A1', 'victory', 'exact'],
      [undefined, 'failure', 'general'],
      ['A3', 'move', 'exact'],
      ['A4', 'failure', 'exact'],
    ])
    expect(movements[0].legs).toEqual([{ from: 'ROS', to: 'BRU' }])
    expect(movements[1].reason).toBe('standoff')
    expect(movements[3].reason).toBe('dislodged')
  })

  it('tolerates the null collections of the oldest reports', () => {
    expect(
      legacyMovements({ ...base, combats: null, orders: null } as unknown as TurnReport),
    ).toEqual([])
  })
})
