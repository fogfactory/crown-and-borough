import { describe, expect, it } from 'vitest'

import { groupWinterReport, winterGroupOf } from '@/lib/winter-report'
import type { Player, WinterInvestmentReport, WinterStockReport } from '@/types'

const players: Player[] = [
  { id: 'P1', name: 'One', color: '#a84632' },
  { id: 'P2', name: 'Two', color: '#2f5d8a' },
  { id: 'P3', name: 'Three', color: '#3f7d3c' },
]

const order = (fields: Partial<WinterInvestmentReport>): WinterInvestmentReport => ({
  kind: 'build',
  player: 'P1',
  outcome: 'success',
  cost: 0,
  ...fields,
})

const stock = (fields: Partial<WinterStockReport>): WinterStockReport => ({
  territory: 'AAA',
  owner: 'P1',
  stockBefore: 0,
  stockAfter: 0,
  ...fields,
})

describe('winterGroupOf', () => {
  it('ranges orders by nature, unknown kinds apart', () => {
    expect(winterGroupOf('recruit')).toBe('domain')
    expect(winterGroupOf('fief_founded')).toBe('titles')
    expect(winterGroupOf('inquiry')).toBe('court')
    expect(winterGroupOf('rejected')).toBe('other')
  })
})

describe('groupWinterReport', () => {
  it('puts the viewer first, then the players, the neutral stocks and the realm', () => {
    const houses = groupWinterReport(
      [
        order({ player: '' , kind: 'prosperity_founded' }),
        order({ player: 'P1' }),
        order({ player: 'P2' }),
      ],
      [stock({ territory: 'ZZZ', owner: undefined, stockBefore: 4, stockAfter: 2 })],
      players,
      'P2',
    )
    expect(houses.map((house) => [house.key, house.kind])).toEqual([
      ['P2', 'player'],
      ['P1', 'player'],
      ['NEUTRAL', 'neutral'],
      ['', 'realm'],
    ])
  })

  it('ranges a house orders by nature and counts its failures and costs', () => {
    const [house] = groupWinterReport(
      [
        order({ kind: 'inquiry', cost: 3 }),
        order({ kind: 'build', cost: 3 }),
        order({ kind: 'rejected', outcome: 'failure' }),
      ],
      [],
      players,
    )
    expect(house.groups.map((entry) => entry.group)).toEqual(['domain', 'court', 'other'])
    expect(house.orderCount).toBe(3)
    expect(house.failed).toBe(1)
    expect(house.cost).toBe(6)
  })

  it('balances stocks − costs + received, a gift being a cost of its sender', () => {
    const houses = groupWinterReport(
      [order({ kind: 'transfer', cost: 3, source: 'AAA', target: 'BBB', amount: 3 })],
      [
        stock({ territory: 'AAA', owner: 'P1', stockBefore: 5, stockAfter: 1 }),
        stock({ territory: 'BBB', owner: 'P2', stockBefore: 0, stockAfter: 2 }),
      ],
      players,
    )
    const [sender, receiver] = houses
    expect([sender.before, sender.cost, sender.received, sender.left, sender.after]).toEqual([5, 3, 0, 2, 1])
    expect([receiver.before, receiver.cost, receiver.received, receiver.left, receiver.after]).toEqual([0, 0, 3, 3, 2])
  })

  it('ignores a failed transfer and sums several transfers into one stock', () => {
    const houses = groupWinterReport(
      [
        order({ kind: 'transfer', outcome: 'failure', target: 'BBB', amount: 9 }),
        order({ kind: 'transfer', player: 'P1', cost: 2, target: 'BBB', amount: 2 }),
        order({ kind: 'transfer', player: 'P3', cost: 1, target: 'BBB', amount: 1 }),
      ],
      [stock({ territory: 'BBB', owner: 'P2', stockBefore: 4, stockAfter: 4 })],
      players,
    )
    // A stock the conservation left as it was does not open a block on its own.
    expect(houses.map((house) => house.key)).toEqual(['P1', 'P3'])
    const [receiver] = groupWinterReport(
      [
        order({ kind: 'transfer', outcome: 'failure', target: 'BBB', amount: 9 }),
        order({ kind: 'transfer', player: 'P1', cost: 2, target: 'BBB', amount: 2 }),
        order({ kind: 'transfer', player: 'P3', cost: 1, target: 'BBB', amount: 1 }),
      ],
      [stock({ territory: 'BBB', owner: 'P2', stockBefore: 4, stockAfter: 3 })],
      players,
      'P2',
    )
    expect(receiver.received).toBe(3)
    expect(receiver.left).toBe(7)
  })

  it('lists only the stocks the conservation changed, yet counts every stock in the balance', () => {
    const [house] = groupWinterReport(
      [],
      [
        stock({ territory: 'AAA', stockBefore: 40, stockAfter: 20 }),
        stock({ territory: 'BBB', stockBefore: 1, stockAfter: 1 }),
      ],
      players,
    )
    expect(house.halved.map((entry) => entry.territory)).toEqual(['AAA'])
    expect(house.before).toBe(41)
    expect(house.after).toBe(21)
  })
})
