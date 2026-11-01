import { describe, expect, it } from 'vitest'

import {
  buildAlliancePairs,
  buildHouses,
  buildLinks,
  countLines,
  layoutTree,
  MEDALLION_LINE_H,
  MEDALLION_W,
  placeTree,
  TREE_LANE,
  type ClaimRef,
  type TreeLayout,
} from '@/lib/lineage'
import type { Marriage, Noble, Player } from '@/types'

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

  it('lists the king, the queen and the dowager queen', () => {
    const titles = (
      house: number,
      code: string,
      religion: Parameters<typeof buildHouses>[5],
    ) =>
      buildHouses(players, nobles, [], [], [], religion)[house].members.find(
        (entry) => entry.code === code,
      )?.religious
    expect(titles(2, 'MAR', { king: 'MAR', queen: 'ANN' })).toEqual([{ title: 'king' }])
    expect(titles(1, 'ANN', { king: 'MAR', queen: 'ANN' })).toEqual([{ title: 'queen' }])
    expect(titles(1, 'EVE', { king: 'MAR', queen: 'ANN' })).toEqual([])
    // The widow of a dead king keeps the title, as a dowager.
    expect(titles(1, 'ANN', { queen: 'ANN', queenDowager: true })).toEqual([
      { title: 'dowager_queen' },
    ])
  })

  it('keeps ended claims on the heir, a live one taking precedence', () => {
    const ended = {
      heir: 'ANN',
      target: 'JEA',
      spouse: 'EVE',
      turn: 1,
      rank: 0,
      ended: true,
      endedTurn: 3,
      outcome: 'honoured' as const,
    }
    const live = { heir: 'MAR', target: 'EVE', spouse: 'JEA', turn: 2, rank: 1 }
    const houses = buildHouses(players, nobles, [], [], [ended, live])
    const claimOf = (code: string) =>
      houses.flatMap((house) => house.members).find((m) => m.code === code)?.claim
    expect(claimOf('ANN')).toMatchObject({ target: 'JEA', outcome: 'honoured' })
    expect(claimOf('MAR')?.outcome).toBeUndefined()
    // Only live claims count as claiming a noble.
    const claimed = houses
      .flatMap((house) => house.members)
      .map((m) => [m.code, m.claimedBy])
    expect(claimed.filter(([, count]) => count)).toEqual([['EVE', 1]])
  })

  it('flags the excommunicated nobles', () => {
    const houses = buildHouses(players, nobles, [], [], [], {
      excommunicated: [{ noble: 'ANN', reason: 'papal', turn: 1 }],
    })
    const flags = houses.flatMap((house) =>
      house.members.map((m) => [m.code, m.excommunicated]),
    )
    expect(flags.filter(([, flag]) => flag)).toEqual([['ANN', true]])
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
})

/** Houses from `{ P1: ['A1', 'A2'] }`: nobles in succession order. */
function world(spec: Record<string, string[]>) {
  const ids = Object.keys(spec)
  const people: Player[] = ids.map((id) => ({
    id,
    name: id,
    color: '#888',
    succession: spec[id],
  }))
  const all = ids.flatMap((id) => spec[id].map((code) => noble(code, id)))
  return buildHouses(people, all, [], [])
}
const marriage = (
  a: string,
  b: string,
  kind: 'head' | 'secondary' | 'ended' = 'secondary',
  weight = 1,
): Marriage => ({
  nobleA: a,
  nobleB: b,
  turn: 1,
  active: kind !== 'ended',
  category: kind === 'ended' ? undefined : kind,
  weight,
  activeHeadFor: kind === 'head' ? ['P1'] : [],
})

describe('layoutTree', () => {
  const rowsOf = (layout: TreeLayout) => Object.fromEntries(layout.rowOf)
  const columnIds = (layout: TreeLayout) => layout.columns.map((house) => house.player.id)

  it('puts the couple of the crown on one row with the heirs below it', () => {
    const houses = world({ P1: ['POP'], P2: ['KIN', 'PIE'], P3: ['QUE', 'ANN', 'GAS'] })
    const links = buildLinks(houses, [marriage('KIN', 'QUE', 'head', 5)])
    const claims: ClaimRef[] = [
      { heir: 'ANN', target: 'KIN', spouse: 'QUE' },
      { heir: 'PIE', target: 'QUE', spouse: 'KIN' },
      { heir: 'GAS', target: 'KIN', spouse: 'QUE' },
    ]
    const layout = layoutTree(houses, links, 'P2', claims)
    const rows = rowsOf(layout)
    expect(columnIds(layout)).toContain('P1')
    expect(rows.KIN).toBe(rows.QUE)
    expect(layout.marriages[0].aligned).toBe(true)
    expect(rows.PIE).toBeGreaterThan(rows.KIN)
    expect(rows.ANN).toBeGreaterThan(rows.QUE)
    expect(rows.GAS).toBeGreaterThan(rows.ANN)
    // One stem per couple: the three claims share a lane of the same gap.
    expect(new Set(layout.stems.map((stem) => `${stem.gap}:${stem.lane}`)).size).toBe(1)
  })

  it('keeps rows strictly increasing down a house, whatever the marriages', () => {
    const houses = world({ P1: ['A1', 'A2', 'A3'], P2: ['B1', 'B2', 'B3'] })
    const links = buildLinks(houses, [
      marriage('A3', 'B1'),
      marriage('A1', 'B3'),
      marriage('A2', 'B2'),
    ])
    const layout = layoutTree(houses, links, null)
    for (const house of layout.columns)
      house.members.forEach((member, index) => {
        if (index > 0)
          expect(layout.rowOf.get(member.code)!).toBeGreaterThan(
            layout.rowOf.get(house.members[index - 1].code)!,
          )
      })
  })

  it('aligns two marriages between the same houses at different ranks', () => {
    const houses = world({ P1: ['A1', 'A2', 'A3'], P2: ['B1', 'B2', 'B3'] })
    const links = buildLinks(houses, [marriage('A1', 'B2'), marriage('A3', 'B3')])
    const layout = layoutTree(houses, links, null)
    const rows = rowsOf(layout)
    expect(layout.marriages.every((m) => m.aligned)).toBe(true)
    expect(rows.A1).toBe(rows.B2)
    expect(rows.A3).toBe(rows.B3)
  })

  it('refuses the weakest of two contradictory marriages and bends its line', () => {
    const houses = world({ P1: ['A1', 'A2'], P2: ['B1', 'B2'] })
    // A1 x B2 and A2 x B1 cannot both be level: the head one wins.
    const links = buildLinks(houses, [
      marriage('A2', 'B1', 'secondary', 9),
      marriage('A1', 'B2', 'head', 1),
    ])
    const layout = layoutTree(houses, links, null)
    const aligned = Object.fromEntries(
      layout.marriages.map((m) => [m.link.key, m.aligned]),
    )
    expect(aligned).toEqual({ 'A1-B2': true, 'A2-B1': false })
    expect(layout.rowOf.get('A1')).toBe(layout.rowOf.get('B2'))
    expect(layout.bendLane.has('A2-B1')).toBe(true)
    const placed = placeTree(layout, () => 0)
    const bent = placed.marriages.find((m) => !m.aligned)!
    expect(bent.d).toMatch(/^M[\d.]+,[\d.]+ H[\d.]+ V[\d.]+ H[\d.]+$/)
  })

  it('refuses to level a noble with two spouses of the same house', () => {
    const houses = world({ P1: ['A1'], P2: ['B1', 'B2'] })
    const links = buildLinks(houses, [marriage('A1', 'B1', 'head'), marriage('A1', 'B2')])
    const layout = layoutTree(houses, links, null)
    expect(layout.marriages.map((m) => m.aligned)).toEqual([true, false])
  })

  it('clears the row of a marriage line in the columns it runs over', () => {
    const houses = world({
      P1: ['A1', 'A2', 'A3'],
      P2: ['B1', 'B2', 'B3'],
      P3: ['C1', 'C2', 'C3'],
    })
    const links = buildLinks(houses, [
      marriage('A1', 'B1', 'head', 5),
      marriage('B3', 'C3', 'head', 5),
      marriage('A2', 'C2', 'secondary', 1),
    ])
    const layout = layoutTree(houses, links, 'P2')
    expect(columnIds(layout)[1]).toBe('P2')
    const spanning = layout.marriages.filter(
      (m) =>
        layout.columns.findIndex((h) => h.members.some((x) => x.code === m.right)) -
          layout.columns.findIndex((h) => h.members.some((x) => x.code === m.left)) >
        1,
    )
    expect(spanning).toHaveLength(1)
    const middle = layout.columns[1].members.map((m) => layout.rowOf.get(m.code))
    expect(middle).not.toContain(layout.rowOf.get(spanning[0].left))
    for (const house of layout.columns)
      house.members.forEach((member, index) => {
        if (index > 0)
          expect(layout.rowOf.get(member.code)!).toBeGreaterThan(
            layout.rowOf.get(house.members[index - 1].code)!,
          )
      })
    // Every aligned marriage is still level.
    for (const m of layout.marriages)
      if (m.aligned) expect(layout.rowOf.get(m.left)).toBe(layout.rowOf.get(m.right))
  })

  it('orders nine and more houses without trying every permutation', () => {
    const ids = Array.from({ length: 10 }, (_, i) => `H${i}`)
    const spec: Record<string, string[]> = {}
    for (const id of ids) spec[id] = [`${id}a`, `${id}b`]
    // Houses given in a scrambled order, married in a ring.
    const scrambled = ['H0', 'H5', 'H1', 'H6', 'H2', 'H7', 'H3', 'H8', 'H4', 'H9']
    const houses = world(Object.fromEntries(scrambled.map((id) => [id, spec[id]])))
    const pairs = ids.map((id, i): [string, string] => [
      `${id}b`,
      `${ids[(i + 1) % 10]}a`,
    ])
    const links = buildLinks(
      houses,
      pairs.map(([a, b]) => marriage(a, b)),
    )
    const skipped = (order: string[]) => {
      const at = new Map(order.map((id, i) => [id, i]))
      return pairs.reduce((sum, [a, b]) => {
        const d = Math.abs(at.get(a.slice(0, 2))! - at.get(b.slice(0, 2))!)
        return sum + Math.max(0, d - 1)
      }, 0)
    }
    const layout = layoutTree(houses, links, null)
    expect([...columnIds(layout)].sort()).toEqual([...ids].sort())
    expect(skipped(columnIds(layout))).toBeLessThan(skipped(scrambled))
  })

  it('is deterministic, whatever the order of the marriages', () => {
    const houses = world({
      P1: ['A1', 'A2', 'A3'],
      P2: ['B1', 'B2', 'B3'],
      P3: ['C1', 'C2'],
      P4: ['D1', 'D2'],
    })
    const raw = [
      marriage('A1', 'B2', 'head', 4),
      marriage('A2', 'B1', 'secondary', 4),
      marriage('B3', 'C1', 'secondary', 2),
      marriage('A3', 'D2', 'secondary', 2),
      marriage('C2', 'D1', 'ended'),
    ]
    const claims: ClaimRef[] = [{ heir: 'D2', target: 'A3', spouse: 'D2' }]
    const dump = (marriages: Marriage[]) => {
      const layout = layoutTree(houses, buildLinks(houses, marriages), 'P1', claims)
      return JSON.stringify([
        columnIds(layout),
        [...layout.rowOf].sort(),
        layout.marriages.map((m) => [m.link.key, m.aligned]).sort(),
        layout.stems,
      ])
    }
    expect(dump(raw)).toBe(dump(raw))
    expect(dump([...raw].reverse())).toBe(dump(raw))
  })
})

describe('placeTree', () => {
  it('draws level spouses on one line, in rows as tall as their tallest medallion', () => {
    const houses = world({ P1: ['A1', 'A2'], P2: ['B1', 'B2'] })
    const links = buildLinks(houses, [marriage('A2', 'B2')])
    const layout = layoutTree(houses, links, null)
    const lines = new Map([['B1', 3]])
    const placed = placeTree(layout, (code) => lines.get(code) ?? 0)
    const a2 = placed.boxes.get('A2')!
    const b2 = placed.boxes.get('B2')!
    expect(a2.y).toBe(b2.y)
    const [{ d }] = placed.marriages
    expect(d).toBe(`M${a2.x + MEDALLION_W},${a2.y + 22} L${b2.x},${b2.y + 22}`)
    // The tall B1 pushes the second row down in both houses.
    const a1 = placed.boxes.get('A1')!
    const b1 = placed.boxes.get('B1')!
    expect(b1.h - a1.h).toBe(3 * MEDALLION_LINE_H + 4)
    expect(a2.y).toBeGreaterThanOrEqual(a1.y + b1.h)
  })

  it('gives each couple of a gap its own lane, 8px apart, and widens the gap', () => {
    const houses = world({ P1: ['A1', 'A2', 'A3'], P2: ['B1', 'B2', 'B3'] })
    const links = buildLinks(houses, [marriage('A1', 'B1'), marriage('A2', 'B2')])
    const claims: ClaimRef[] = [
      { heir: 'B3', target: 'A1', spouse: 'B1' },
      { heir: 'A3', target: 'B2', spouse: 'A2' },
    ]
    const layout = layoutTree(houses, links, null, claims)
    expect(layout.lanes).toEqual([2])
    const placed = placeTree(layout, () => 0)
    const stemX = placed.claims.map((c) => Number(/^M([\d.]+),/.exec(c.d)![1]))
    expect(Math.abs(stemX[0] - stemX[1])).toBe(TREE_LANE)
    const gap = placed.columnX[1] - (placed.columnX[0] + MEDALLION_W)
    expect(gap).toBeGreaterThanOrEqual(24 + 2 * TREE_LANE)
    for (const x of stemX) {
      expect(x).toBeGreaterThan(placed.columnX[0] + MEDALLION_W)
      expect(x).toBeLessThan(placed.columnX[1])
    }
  })

  it('counts every wrapped line of a long title', () => {
    expect(countLines(['Duchy · ROC'])).toBe(1)
    expect(countLines(['Bishop of Saint-Rémy-de-Provence-en-Alpilles'])).toBe(2)
    expect(countLines(['a', 'b'])).toBe(2)
  })
})
