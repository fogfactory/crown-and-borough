import { describe, expect, it } from 'vitest'

import {
  buildAlliancePairs,
  buildHouses,
  buildLinks,
  CARD_H,
  CARD_W,
  claimRoute,
  GAP_X,
  layoutColumns,
  PITCH,
} from '@/lib/lineage'
import type { Noble, Player } from '@/types'

const players: Player[] = [
  { id: 'P1', name: 'Alice', color: '#a00', succession: ['LUC', 'JEA'] },
  { id: 'P2', name: 'Bob', color: '#00a', succession: ['ANN', 'EVE'] },
  { id: 'P3', name: 'Cleo', color: '#0a0', succession: ['MAR'] },
]
const noble = (code: string, owner: string, extra: Partial<Noble> = {}): Noble => ({
  id: code,
  code,
  name: code,
  owner,
  location: 'ROS',
  status: 'free',
  ...extra,
})
const nobles = [
  noble('JEA', 'P1'),
  noble('LUC', 'P1', { dignities: ['bastard'] }),
  noble('ANN', 'P2'),
  noble('EVE', 'P2'),
  noble('MAR', 'P3'),
]

describe('lineage', () => {
  it('lists every title, the deceased, and the claims on and by a noble', () => {
    const houses = buildHouses(
      players,
      nobles,
      [{ code: 'OLD', name: 'Old', owner: 'P1', cause: 'natural', turn: 2 }],
      [
        {
          capital: 'ROC',
          title: 'duchy',
          territories: ['ROC'],
          owner: 'P1',
          holder: 'JEA',
        },
        {
          capital: 'AAA',
          title: 'barony',
          territories: ['AAA'],
          owner: 'P1',
          holder: 'JEA',
        },
      ],
      [{ heir: 'ANN', target: 'JEA', spouse: 'EVE', turn: 1, rank: 1 }],
    )
    expect(houses[0].members.map((m) => m.code)).toEqual(['LUC', 'JEA', 'OLD'])
    expect(houses[0].members[0].dignities).toEqual(['bastard'])
    expect(houses[0].members[1].fiefs.map((f) => f.title)).toEqual(['duchy', 'barony'])
    expect(houses[0].members[1].claimedBy).toBe(1)
    expect(houses[1].members[0].claim).toMatchObject({ target: 'JEA', rank: 1 })
    expect(houses[0].members[2].dead).toBe(true)
  })

  it('lists the religious titles of a noble, highest first', () => {
    const houses = buildHouses(players, nobles, [], [], [], {
      pope: 'LUC',
      cardinals: ['LUC'],
      bishoprics: [{ region: 'R1', name: 'Alpilles', territories: [], bishop: 'LUC' }],
    })
    const member = houses[0].members.find((entry) => entry.code === 'LUC')
    expect(member?.religious).toEqual([
      { title: 'pope' },
      { title: 'cardinal' },
      { title: 'bishop', seat: 'Alpilles' },
    ])
  })

  it('classifies marriages: active head, secondary, ended', () => {
    const houses = buildHouses(players, nobles, [], [])
    const links = buildLinks(houses, [
      {
        nobleA: 'JEA',
        nobleB: 'ANN',
        turn: 1,
        active: true,
        category: 'head',
        weight: 5,
        activeHeadFor: ['P1', 'P2'],
      },
      {
        nobleA: 'LUC',
        nobleB: 'EVE',
        turn: 2,
        active: true,
        category: 'secondary',
        weight: 5,
      },
      {
        nobleA: 'LUC',
        nobleB: 'MAR',
        turn: 2,
        active: true,
        category: 'secondary',
        weight: 1,
      },
      { nobleA: 'EVE', nobleB: 'MAR', turn: 3, active: false },
    ])
    expect(links.map((l) => l.kind)).toEqual(['head', 'secondary', 'secondary', 'ended'])
    const pairs = buildAlliancePairs(links)
    expect(pairs).toHaveLength(2)
    expect(pairs[0]).toMatchObject({ weight: 10, kind: 'head', head: true })
    expect(pairs[0].carrier.key).toBe('JEA-ANN')
  })

  it('centres the focus house and aligns linked spouses on the same row', () => {
    const houses = buildHouses(players, nobles, [], [])
    const links = buildLinks(houses, [
      {
        nobleA: 'JEA',
        nobleB: 'ANN',
        turn: 1,
        active: true,
        category: 'head',
        weight: 5,
        activeHeadFor: ['P1', 'P2'],
      },
      {
        nobleA: 'EVE',
        nobleB: 'MAR',
        turn: 3,
        active: true,
        category: 'secondary',
        weight: 3,
      },
    ])
    const { columns } = layoutColumns(houses, links, 'P2')
    expect(columns.map((c) => c.house.player.id)).toEqual(['P3', 'P2', 'P1'])
    const row = (house: number, code: string) =>
      columns[house].offset +
      columns[house].house.members.findIndex((m) => m.code === code)
    expect(row(2, 'JEA')).toBe(row(1, 'ANN'))
    expect(row(0, 'MAR')).toBe(row(1, 'EVE'))
    expect(Math.min(...columns.map((c) => c.offset))).toBe(0)
  })

  it('keeps straight links clear of the cards in the columns between their ends', () => {
    const ring = [
      ...players,
      { id: 'P4', name: 'Dan', color: '#aa0', succession: ['DAN', 'DOM'] },
    ]
    const all = [...nobles, noble('DAN', 'P4'), noble('DOM', 'P4')]
    const houses = buildHouses(ring, all, [], [])
    const links = buildLinks(houses, [
      {
        nobleA: 'JEA',
        nobleB: 'ANN',
        turn: 1,
        active: true,
        category: 'secondary',
        weight: 1,
      },
      {
        nobleA: 'EVE',
        nobleB: 'MAR',
        turn: 1,
        active: true,
        category: 'secondary',
        weight: 1,
      },
      {
        nobleA: 'DAN',
        nobleB: 'MAR',
        turn: 1,
        active: true,
        category: 'secondary',
        weight: 1,
      },
      {
        nobleA: 'DOM',
        nobleB: 'LUC',
        turn: 1,
        active: true,
        category: 'secondary',
        weight: 1,
      },
    ])
    const { columns } = layoutColumns(houses, links, 'P1')
    const columnOf = new Map<string, number>()
    const topOf = new Map<string, number>()
    columns.forEach((column, index) =>
      column.house.members.forEach((member, i) => {
        columnOf.set(member.code, index)
        topOf.set(member.code, (column.offset + i) * PITCH)
      }),
    )
    for (const link of links) {
      const [l, r] =
        columnOf.get(link.a)! < columnOf.get(link.b)!
          ? [link.a, link.b]
          : [link.b, link.a]
      const cl = columnOf.get(l)!
      const cr = columnOf.get(r)!
      const x1 = cl * (CARD_W + GAP_X) + CARD_W
      const x2 = cr * (CARD_W + GAP_X)
      const y1 = topOf.get(l)! + CARD_H / 2
      const y2 = topOf.get(r)! + CARD_H / 2
      for (let k = cl + 1; k < cr; k++) {
        const ys = [k * (CARD_W + GAP_X), k * (CARD_W + GAP_X) + CARD_W].map(
          (x) => y1 + ((y2 - y1) * (x - x1)) / (x2 - x1),
        )
        for (const member of columns[k].house.members) {
          const top = topOf.get(member.code)!
          expect(Math.max(...ys) > top && Math.min(...ys) < top + CARD_H).toBe(false)
        }
      }
    }
  })

  it('routes a claim as a filiation into the marriage line without crossing a card', () => {
    const houses = buildHouses(players, nobles, [], [])
    const links = buildLinks(houses, [
      {
        nobleA: 'ANN',
        nobleB: 'MAR',
        turn: 1,
        active: true,
        category: 'secondary',
        weight: 1,
      },
    ])
    const claim = { heir: 'LUC', target: 'ANN', spouse: 'MAR' }
    const { columns } = layoutColumns(houses, links, 'P2', [claim])
    const box = new Map<string, { x: number; y: number }>()
    columns.forEach((column, index) =>
      column.house.members.forEach((member, i) =>
        box.set(member.code, {
          x: index * (CARD_W + GAP_X),
          y: (column.offset + i) * PITCH,
        }),
      ),
    )
    const [start, corner, end] = claimRoute(
      box.get('LUC')!,
      box.get('ANN')!,
      box.get('MAR')!,
    )
    expect(corner[1]).toBe(start[1])
    expect(end[0]).toBe(corner[0])
    const marriageY = (box.get('ANN')!.y + box.get('MAR')!.y) / 2 + CARD_H / 2
    expect(end[1]).toBe(marriageY)
    for (const [code, at] of box) {
      if (code === 'LUC') continue
      const hit =
        (Math.min(start[0], corner[0]) < at.x + CARD_W &&
          Math.max(start[0], corner[0]) > at.x &&
          start[1] > at.y &&
          start[1] < at.y + CARD_H) ||
        (corner[0] > at.x &&
          corner[0] < at.x + CARD_W &&
          Math.min(corner[1], end[1]) < at.y + CARD_H &&
          Math.max(corner[1], end[1]) > at.y)
      expect(hit).toBe(false)
    }
  })
})
