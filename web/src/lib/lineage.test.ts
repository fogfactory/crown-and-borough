import { describe, expect, it } from 'vitest'

import {
  buildAlliancePairs,
  buildHouses,
  buildLinks,
  countLines,
  COUPLE_GAP,
  layoutTree,
  MEDALLION_ANCHOR_Y,
  MEDALLION_LINE_H,
  MEDALLION_W,
  placeTree,
  TREE_BUDGET,
  TREE_LANE,
  visibleNobles,
  wrapLines,
  type ClaimRef,
  type TreeLayout,
  type TreePlacement,
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
  it('marks a usurper and shows the exiled title as a reminder', () => {
    const houses = buildHouses(
      players,
      nobles,
      [],
      [
        { capital: 'AAA', title: 'barony', territories: ['AAA'], owner: 'P2', holder: 'ANN', usurped: true, exiled: 'JEA' },
      ],
    )
    const jean = houses[0].members.find((m) => m.code === 'JEA')
    const anne = houses[1].members.find((m) => m.code === 'ANN')
    expect(jean?.fiefs).toEqual([])
    expect(jean?.exiled).toEqual([{ title: 'barony', capital: 'AAA' }])
    expect(anne?.fiefs).toEqual([{ title: 'barony', capital: 'AAA' }])
    expect(anne?.usurped).toEqual(['AAA'])
  })

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


const ROW = MEDALLION_W + COUPLE_GAP

/** Houses, marriages and claims laid out; spouses are `[a, b, kind?]`. */
function build(
  spec: Record<string, string[]>,
  spouses: Array<[string, string, ('head' | 'secondary' | 'ended')?]> = [],
  claims: ClaimRef[] = [],
  focus: string | null = null,
) {
  const houses = world(spec)
  const links = buildLinks(
    houses,
    spouses.map(([a, b, kind]) => marriage(a, b, kind ?? 'secondary', 3)),
  )
  const layout = layoutTree(houses, links, focus, claims)
  return { houses, links, layout, placed: placeTree(layout, () => 0) }
}

/** What every layout owes: all nobles once, side-by-side spouses, heirs below, no overlap. */
function expectSound(
  layout: TreeLayout,
  placed: TreePlacement,
  codes: string[],
  claims: ClaimRef[] = [],
) {
  expect([...layout.rowOf.keys()].sort()).toEqual([...codes].sort())
  expect(layout.order.flat().sort()).toEqual([...codes].sort())
  for (const { left, right, aligned } of layout.marriages) {
    if (!aligned) continue
    expect(layout.rowOf.get(left)).toBe(layout.rowOf.get(right))
    expect(layout.x.get(right)! - layout.x.get(left)!).toBe(ROW)
  }
  for (const row of layout.order) {
    const spans = row
      .map((code) => placed.boxes.get(code)!)
      .sort((a, b) => a.x - b.x)
    spans.forEach((box, index) => {
      expect(box.x).toBeGreaterThanOrEqual(0)
      if (index > 0) expect(box.x).toBeGreaterThanOrEqual(spans[index - 1].x + box.w)
    })
  }
  for (const stem of layout.stems)
    if (stem.hangs) {
      expect(layout.rowOf.get(stem.heir)!).toBeGreaterThan(layout.rowOf.get(stem.target)!)
      if (stem.couple)
        for (const spouse of stem.couple)
          expect(layout.rowOf.get(stem.heir)!).toBeGreaterThan(layout.rowOf.get(spouse)!)
    }
  // Bars of different couples that overlap in a channel are on different lanes.
  const bars = layout.stems.flatMap((stem) =>
    stem.lanes.map((lane, hop) => ({
      key: stem.couple?.join('|') ?? stem.target,
      band: stem.rowFrom + hop,
      lane,
      lo: Math.min(stem.xs[hop], stem.xs[hop + 1]),
      hi: Math.max(stem.xs[hop], stem.xs[hop + 1]),
    })),
  )
  for (const p of bars)
    for (const q of bars)
      if (p !== q && p.band === q.band && p.key !== q.key && p.hi > p.lo && q.hi > q.lo)
        if (p.lo <= q.hi && q.lo <= p.hi) expect(p.lane).not.toBe(q.lane)
  expect(layout.stems).toHaveLength(
    claims.filter(
      (c) => codes.includes(c.heir) && codes.includes(c.target) && c.heir !== c.target,
    ).length,
  )
  // Only the margins of the stems that go round the cards add to the cards' width.
  const margin = layout.stems.filter((stem) => stem.detour).length * 8 + 8
  expect(layout.width).toBeLessThanOrEqual(
    Math.max(...[...placed.boxes.values()].map((box) => box.x + box.w)) +
      1 +
      (layout.stems.some((stem) => stem.detour) ? margin : 0),
  )
}

/** Stem crossings between two rows, from the horizontal position of both ends. */
function crossingsOf(layout: TreeLayout) {
  let total = 0
  const ends = layout.stems
    .filter((stem) => stem.hangs && stem.rowTo === stem.rowFrom + 1)
    .map((stem) => [stem.rowFrom, stem.xs[0], stem.xs[stem.xs.length - 1]])
  for (let i = 0; i < ends.length; i++)
    for (let j = i + 1; j < ends.length; j++)
      if (
        ends[i][0] === ends[j][0] &&
        (ends[i][1] - ends[j][1]) * (ends[i][2] - ends[j][2]) < 0
      )
        total++
  return total
}

describe('layoutTree', () => {
  it('lists the nobles without claim or marriage in a line per house, in succession order', () => {
    const { layout, placed } = build({ P2: ['B1', 'B2'], P1: ['A1', 'A2', 'A3'] })
    expect(layout.rows).toBe(1)
    // Houses in a fixed order, whatever the order given; siblings closer than houses.
    expect(layout.order[0]).toEqual(['A1', 'A2', 'A3', 'B1', 'B2'])
    const gap = (a: string, b: string) =>
      placed.boxes.get(b)!.x - placed.boxes.get(a)!.x - MEDALLION_W
    expect(gap('A1', 'A2')).toBe(gap('A2', 'A3'))
    expect(gap('A3', 'B1')).toBeGreaterThan(gap('A1', 'A2'))
    expect(gap('B1', 'B2')).toBe(gap('A1', 'A2'))
  })

  it('stands the spouses side by side, the earlier successors above and the later ones below', () => {
    const { layout, placed } = build({ P1: ['A1', 'A2', 'A3'], P2: ['B1', 'B2'] }, [
      ['A2', 'B2', 'head'],
    ])
    const [marriageLine] = layout.marriages
    expect(marriageLine.aligned).toBe(true)
    expect(layout.x.get('B2')! - layout.x.get('A2')!).toBe(ROW)
    expect([marriageLine.left, marriageLine.right].sort()).toEqual(['A2', 'B2'])
    // The couple is the tree; each house's earlier nobles stand above it, its later ones below.
    const row = (code: string) => layout.rowOf.get(code)!
    expect(row('A2')).toBe(row('B2'))
    expect(row('A1')).toBeLessThan(row('A2'))
    expect(row('B1')).toBeLessThan(row('B2'))
    expect(row('A3')).toBeGreaterThan(row('A2'))
    // Lines of different houses share a row when they fit.
    expect(row('A1')).toBe(row('B1'))
    expectSound(layout, placed, ['A1', 'A2', 'A3', 'B1', 'B2'])
  })

  it('stands a noble between two married nobles of its house beside the first, as a sibling', () => {
    // A2 comes between A1 and A3, who both marry: it joins the tree of A1.
    const { layout, placed } = build({ P1: ['A1', 'A2', 'A3', 'A4'], P2: ['B1'], P3: ['C1'] }, [
      ['A1', 'B1'],
      ['A3', 'C1'],
    ])
    const row = (code: string) => layout.rowOf.get(code)!
    expect(row('A2')).toBe(row('A1'))
    // On the outer side of the couple, a sibling gap away from A1, not between the spouses.
    const dx = layout.x.get('A2')! - layout.x.get('A1')!
    expect(Math.abs(dx)).toBe(MEDALLION_W + 20)
    expect(Math.sign(dx)).toBe(-Math.sign(layout.x.get('B1')! - layout.x.get('A1')!))
    expect(layout.marriages.every((m) => m.aligned)).toBe(true)
    expect(Math.abs(layout.x.get('B1')! - layout.x.get('A1')!)).toBe(ROW)
    // A4 is later than every married noble of its house.
    expect(row('A4')).toBeGreaterThan(row('A3'))
    expectSound(layout, placed, ['A1', 'A2', 'A3', 'A4', 'B1', 'C1'])
  })

  it('stands a sibling on the right of a lone heir, and keeps the heir centred under the knot', () => {
    const claims: ClaimRef[] = [
      { heir: 'A1', target: 'B1', spouse: 'C1' },
      { heir: 'A3', target: 'B1', spouse: 'C1' },
    ]
    // A2 comes between the heirs A1 and A3 of one couple.
    const { layout, placed } = build(
      { P1: ['A1', 'A2', 'A3'], P2: ['B1'], P3: ['C1'] },
      [['B1', 'C1']],
      claims,
    )
    const row = (code: string) => layout.rowOf.get(code)!
    expect(row('A2')).toBe(row('A1'))
    expect(layout.x.get('A2')! - layout.x.get('A1')!).toBe(MEDALLION_W + 20)
    const knot = layout.x.get('B1')! + MEDALLION_W + COUPLE_GAP / 2
    expect(layout.stems[0].xs[0]).toBeCloseTo(knot, 5)
    expectSound(layout, placed, ['A1', 'A2', 'A3', 'B1', 'C1'], claims)
  })

  it('hangs the heirs below their couple, centred under the knot of the marriage', () => {
    const claims: ClaimRef[] = [{ heir: 'C1', target: 'A1', spouse: 'B1' }]
    const { layout, placed } = build(
      { P1: ['A1', 'A2'], P2: ['B1'], P3: ['C1'] },
      [['A1', 'B1', 'head']],
      claims,
    )
    // A1 and B1 are the couple, C1 hangs under it and A2, later than A1, stands below the tree.
    expect(layout.rowOf.get('A1')).toBe(0)
    expect(layout.rowOf.get('B1')).toBe(0)
    expect(layout.rowOf.get('C1')).toBe(1)
    expect(layout.rowOf.get('A2')).toBe(2)
    const couple = layout.stems[0].couple!
    expect([...couple].sort()).toEqual(['A1', 'B1'])
    const knot = layout.x.get(couple[0])! + MEDALLION_W + COUPLE_GAP / 2
    expect(layout.x.get('C1')! + MEDALLION_W / 2).toBeCloseTo(knot, 5)
    expect(placed.boxes.get('C1')!.y).toBeGreaterThan(placed.boxes.get('A1')!.y)
    expectSound(layout, placed, ['A1', 'A2', 'B1', 'C1'], claims)
  })

  it('founds further generations: heirs who marry or carry claims go lower', () => {
    const claims: ClaimRef[] = [
      { heir: 'C1', target: 'A1', spouse: 'B1' },
      { heir: 'D1', target: 'C1', spouse: 'E1' },
    ]
    const { layout, placed } = build(
      { P1: ['A1'], P2: ['B1', 'E1'], P3: ['C1'], P4: ['D1'] },
      [
        ['A1', 'B1', 'head'],
        ['C1', 'E1'],
      ],
      claims,
    )
    expect(layout.rowOf.get('C1')).toBe(1)
    // E1 is claimless but married to the heir C1: it stands on C1's row.
    expect(layout.rowOf.get('E1')).toBe(1)
    expect(layout.rowOf.get('D1')).toBe(2)
    expect(layout.rows).toBe(3)
    expectSound(layout, placed, ['A1', 'B1', 'C1', 'D1', 'E1'], claims)
  })

  it('stands a noble with two spouses between them', () => {
    const { layout, placed } = build({ P1: ['A1'], P2: ['B1', 'B2'] }, [
      ['A1', 'B1', 'head'],
      ['A1', 'B2', 'secondary'],
    ])
    expect(layout.marriages.every((m) => m.aligned)).toBe(true)
    expect(layout.order[0]).toEqual(['B1', 'A1', 'B2'])
    expectSound(layout, placed, ['A1', 'B1', 'B2'])
  })

  it('arches the marriage of a third spouse over the cards between', () => {
    const { layout, placed } = build({ P1: ['A1'], P2: ['B1', 'B2', 'B3'] }, [
      ['A1', 'B1'],
      ['A1', 'B2'],
      ['A1', 'B3'],
    ])
    const arched = layout.marriages.filter((m) => !m.aligned)
    expect(arched).toHaveLength(1)
    expect(layout.order[0]).toHaveLength(4)
    const path = placed.marriages.find((m) => !m.aligned)!.d
    expect(path).toMatch(/^M[\d.]+,[\d.]+ V[\d.]+ H[\d.]+ V[\d.]+$/)
  })

  it('keeps ended marriages and ended claims in the tree', () => {
    const claims: ClaimRef[] = [{ heir: 'C1', target: 'A1', spouse: 'B1', ended: true }]
    const { layout, placed } = build(
      { P1: ['A1'], P2: ['B1'], P3: ['C1'] },
      [['A1', 'B1', 'ended']],
      claims,
    )
    expect(layout.marriages[0].link.kind).toBe('ended')
    expect(layout.marriages[0].aligned).toBe(true)
    expect(layout.rowOf.get('C1')).toBe(1)
    expectSound(layout, placed, ['A1', 'B1', 'C1'], claims)
  })

  it('hangs the heir under the target alone when no marriage joins the couple of the claim', () => {
    const claims: ClaimRef[] = [{ heir: 'C1', target: 'A1', spouse: 'B1' }]
    const { layout, placed } = build({ P1: ['A1'], P2: ['B1'], P3: ['C1'] }, [], claims)
    expect(layout.stems[0].couple).toBeUndefined()
    expect(layout.rowOf.get('C1')).toBe(layout.rowOf.get('A1')! + 1)
    const d = placed.claims[0].d
    const target = placed.boxes.get('A1')!
    expect(d.startsWith(`M${target.x + MEDALLION_W / 2},${target.y + target.h}`)).toBe(true)
    expectSound(layout, placed, ['A1', 'B1', 'C1'], claims)
  })

  it('leaves a claim that cannot hang off the rows and still draws it', () => {
    // A heir of B, B heir of A: the first claim hangs, the second would loop.
    const claims: ClaimRef[] = [
      { heir: 'B1', target: 'A1', spouse: 'A1', turn: 1 },
      { heir: 'A1', target: 'B1', spouse: 'B1', turn: 2 },
      // The heir is the spouse of the claim and married to the target.
      { heir: 'D1', target: 'C1', spouse: 'D1', turn: 3 },
    ]
    const { layout, placed } = build(
      { P1: ['A1'], P2: ['B1'], P3: ['C1'], P4: ['D1'] },
      [['C1', 'D1']],
      claims,
    )
    expect(layout.stems.map((stem) => stem.hangs)).toEqual([true, false, false])
    expect(layout.rowOf.get('B1')).toBe(1)
    expect(layout.rowOf.get('A1')).toBe(0)
    expect(layout.rowOf.get('C1')).toBe(layout.rowOf.get('D1'))
    const d = placed.claims[2].d
    expect(d).toMatch(/^M[\d.]+,[\d.]+ V[\d.]+ H[\d.]+ V[\d.]+$/)
    expectSound(layout, placed, ['A1', 'B1', 'C1', 'D1'], claims)
  })

  it('routes a claim that cannot hang round the cards, through a margin and the channels', () => {
    // B hangs under A, C under B; A, heir of C, cannot hang: its stem goes up two rows.
    const claims: ClaimRef[] = [
      { heir: 'B1', target: 'A1', spouse: 'A1', turn: 1 },
      { heir: 'C1', target: 'B1', spouse: 'B1', turn: 2 },
      { heir: 'A1', target: 'C1', spouse: 'C1', turn: 3 },
    ]
    const { layout, placed } = build({ P1: ['A1'], P2: ['B1'], P3: ['C1'] }, [], claims)
    expect(layout.stems.map((stem) => stem.hangs)).toEqual([true, true, false])
    const detour = layout.stems[2].detour
    expect(detour).toBeDefined()
    // The vertical line is outside every card, inside the layout.
    const lefts = [...placed.boxes.values()].map((box) => box.x)
    const rights = [...placed.boxes.values()].map((box) => box.x + box.w)
    expect(detour!.x < Math.min(...lefts) || detour!.x > Math.max(...rights)).toBe(true)
    expect(detour!.x).toBeGreaterThan(0)
    expect(detour!.x).toBeLessThan(placed.width)
    // No segment of the stem goes through the inside of a card.
    const nums = placed.claims[2].d.match(/[MVH][-\d.]+(,[-\d.]+)?/g)!
    let cx = 0
    let cy = 0
    const segments: Array<[number, number, number, number]> = []
    for (const token of nums) {
      const [px, py] = token.slice(1).split(',').map(Number)
      const [nx, ny] =
        token[0] === 'M' ? [px, py] : token[0] === 'V' ? [cx, px] : [px, cy]
      if (token[0] !== 'M') segments.push([cx, cy, nx, ny])
      cx = nx
      cy = ny
    }
    expect(segments.length).toBeGreaterThanOrEqual(4)
    for (const [x1, y1, x2, y2] of segments)
      for (const box of placed.boxes.values()) {
        const crosses =
          Math.max(x1, x2) > box.x &&
          Math.min(x1, x2) < box.x + box.w &&
          Math.max(y1, y2) > box.y &&
          Math.min(y1, y2) < box.y + box.h
        expect(crosses).toBe(false)
      }
    expectSound(layout, placed, ['A1', 'B1', 'C1'], claims)
  })

  it('leaves a free slot for a stem that skips a row', () => {
    // H hangs under (A1, B1) but is married to the heir G of a deeper couple.
    const claims: ClaimRef[] = [
      { heir: 'C1', target: 'A1', spouse: 'B1' },
      { heir: 'G1', target: 'C1', spouse: 'E1' },
      { heir: 'H1', target: 'A1', spouse: 'B1' },
    ]
    const { layout, placed } = build(
      { P1: ['A1'], P2: ['B1', 'E1'], P3: ['C1', 'G1', 'H1'] },
      [
        ['A1', 'B1', 'head'],
        ['C1', 'E1'],
        ['G1', 'H1'],
      ],
      claims,
    )
    expect(layout.rowOf.get('G1')).toBe(2)
    expect(layout.rowOf.get('H1')).toBe(2)
    const long = layout.stems.find((stem) => stem.heir === 'H1')!
    expect(long.rowTo - long.rowFrom).toBe(2)
    expect(long.xs).toHaveLength(3)
    const slotX = long.xs[1]
    for (const code of layout.order[1]) {
      const box = placed.boxes.get(code)!
      expect(slotX < box.x || slotX > box.x + box.w).toBe(true)
    }
    expectSound(layout, placed, ['A1', 'B1', 'C1', 'E1', 'G1', 'H1'], claims)
  })

  it('orders the rows to cross as few stems as it can', () => {
    // The heirs' houses sort the other way round than their couples.
    const claims: ClaimRef[] = [
      { heir: 'Z1', target: 'A1', spouse: 'B1' },
      { heir: 'Y1', target: 'A2', spouse: 'B2' },
    ]
    const { layout, placed } = build(
      { P1: ['A1', 'A2'], P2: ['B1', 'B2'], P8: ['Y1'], P9: ['Z1'] },
      [
        ['A1', 'B1'],
        ['A2', 'B2'],
      ],
      claims,
    )
    expect(crossingsOf(layout)).toBe(0)
    expectSound(layout, placed, ['A1', 'A2', 'B1', 'B2', 'Y1', 'Z1'], claims)
  })

  it('shares one lane between the heirs of one couple', () => {
    const claims: ClaimRef[] = [
      { heir: 'C1', target: 'A1', spouse: 'B1' },
      { heir: 'C2', target: 'A1', spouse: 'B1' },
      { heir: 'C3', target: 'B1', spouse: 'A1' },
    ]
    const { layout } = build(
      { P1: ['A1'], P2: ['B1'], P3: ['C1', 'C2', 'C3'] },
      [['A1', 'B1']],
      claims,
    )
    expect(layout.lanes[0]).toBe(1)
    expect(new Set(layout.stems.map((stem) => stem.lanes[0]))).toEqual(new Set([0]))
    expect(crossingsOf(layout)).toBe(0)
  })

  it('handles ten houses and more: every noble once, no overlap, deterministic', () => {
    const ids = Array.from({ length: 12 }, (_, i) => `H${String(i).padStart(2, '0')}`)
    const spec: Record<string, string[]> = {}
    for (const id of ids) spec[id] = [`${id}a`, `${id}b`, `${id}c`]
    const spouses: Array<[string, string]> = ids.map((id, i) => [
      `${id}b`,
      `${ids[(i + 1) % ids.length]}a`,
    ])
    const claims: ClaimRef[] = ids.map((id, i) => ({
      heir: `${id}c`,
      target: `${id}b`,
      spouse: `${ids[(i + 1) % ids.length]}a`,
    }))
    const first = build(spec, spouses, claims)
    expectSound(first.layout, first.placed, Object.values(spec).flat(), claims)
    const again = build(spec, spouses, claims)
    expect(JSON.stringify([...again.layout.x])).toBe(JSON.stringify([...first.layout.x]))
    // Twelve couples with an heir each fit the budget: the trees stack on shelves.
    expect(first.placed.width).toBeLessThanOrEqual(TREE_BUDGET)
    expect(first.layout.rows).toBeGreaterThanOrEqual(12)
  })

  it('stacks the lines of twelve houses with no marriage within the budget', () => {
    const ids = Array.from({ length: 12 }, (_, i) => `H${String(i).padStart(2, '0')}`)
    const spec: Record<string, string[]> = {}
    for (const id of ids) spec[id] = [`${id}a`, `${id}b`, `${id}c`, `${id}d`]
    const { layout, placed } = build(spec)
    expectSound(layout, placed, Object.values(spec).flat())
    expect(placed.width).toBeLessThanOrEqual(TREE_BUDGET)
    // A line is in succession order, left to right, on one row.
    for (const id of ids) {
      const row = layout.rowOf.get(`${id}a`)
      for (const letter of ['a', 'b', 'c', 'd'])
        expect(layout.rowOf.get(`${id}${letter}`)).toBe(row)
      expect(layout.x.get(`${id}a`)!).toBeLessThan(layout.x.get(`${id}d`)!)
    }
  })

  it('wraps a line wider than the budget', () => {
    const codes = Array.from({ length: 14 }, (_, i) => `A${String(i).padStart(2, '0')}`)
    const { layout, placed } = build({ P1: codes })
    expectSound(layout, placed, codes)
    expect(layout.rows).toBe(3)
    expect(placed.width).toBeLessThanOrEqual(TREE_BUDGET)
    expect(layout.order.flat()).toEqual(codes)
  })

  it('sends the deceased of a house with a tree below it', () => {
    const houses = buildHouses(
      [{ id: 'P1', name: 'P1', color: '#888', succession: ['A1', 'A2'] }, ...players.slice(1, 2)],
      [noble('A1', 'P1'), noble('A2', 'P1'), noble('B1', 'P2')],
      [{ code: 'D1', name: 'D1', owner: 'P1', cause: 'execution', turn: 1 }],
      [],
    )
    const links = buildLinks(houses, [marriage('A2', 'B1')])
    const layout = layoutTree(houses, links, null)
    // A1 comes before the married A2: above; the dead D1 is below.
    expect(layout.rowOf.get('A1')!).toBeLessThan(layout.rowOf.get('A2')!)
    expect(layout.rowOf.get('D1')!).toBeGreaterThan(layout.rowOf.get('A2')!)
  })

  it('is deterministic under shuffled houses, marriages and claims', () => {
    const spec = {
      P1: ['A1', 'A2', 'A3'],
      P2: ['B1', 'B2', 'B3'],
      P3: ['C1', 'C2'],
      P4: ['D1', 'D2'],
    }
    const raw: Array<[string, string, ('head' | 'secondary' | 'ended')?]> = [
      ['A1', 'B2', 'head'],
      ['A2', 'B1', 'secondary'],
      ['B3', 'C1', 'secondary'],
      ['A3', 'D2', 'secondary'],
      ['C2', 'D1', 'ended'],
    ]
    const claims: ClaimRef[] = [
      { heir: 'D2', target: 'A1', spouse: 'B2', turn: 1 },
      { heir: 'C2', target: 'B3', spouse: 'C1', turn: 2 },
      { heir: 'B1', target: 'A3', spouse: 'D2', turn: 3, ended: true },
    ]
    const dump = (
      keys: string[],
      marriages: typeof raw,
      list: ClaimRef[],
      focus: string | null,
    ) => {
      const houses = world(Object.fromEntries(keys.map((key) => [key, spec[key as 'P1']])))
      const links = buildLinks(
        houses,
        marriages.map(([a, b, kind]) => marriage(a, b, kind ?? 'secondary', 2)),
      )
      const layout = layoutTree(houses, links, focus, list)
      return JSON.stringify([
        layout.order,
        [...layout.x].sort(),
        [...layout.rowOf].sort(),
        layout.marriages.map((m) => [m.link.key, m.aligned]).sort(),
        layout.stems
          .map((s) => [s.heir, s.target, s.hangs, s.xs, s.lanes])
          .sort((p, q) => String(p[0]).localeCompare(String(q[0]))),
      ])
    }
    const keys = ['P1', 'P2', 'P3', 'P4']
    for (const focus of [null, 'P3']) {
      const base = dump(keys, raw, claims, focus)
      expect(dump([...keys].reverse(), [...raw].reverse(), [...claims].reverse(), focus)).toBe(base)
      expect(dump(['P3', 'P1', 'P4', 'P2'], [...raw].reverse(), claims, focus)).toBe(base)
    }
  })

  it('lays out the lineage demo: four houses, a hub, an ended marriage, four claims', () => {
    const claims: ClaimRef[] = [
      { heir: 'ZMD', target: 'ZLA', spouse: 'ZMA' },
      { heir: 'ZLE', target: 'ZMC', spouse: 'ZLB' },
      { heir: 'ZNC', target: 'ZLC', spouse: 'ZNB' },
      { heir: 'ZOC', target: 'ZLD', spouse: 'ZOA' },
    ]
    const spec = {
      P1: ['RAI', 'HUO', 'ZOE', 'ZLA', 'ZLB', 'ZLC', 'ZLD', 'ZLE'],
      P2: ['JEN', 'DRE', 'ZMA', 'ZMC', 'ZMD'],
      P3: ['LOU', 'RIH', 'ZNA', 'ZNB', 'ZNC'],
      P4: ['BOU', 'PHL', 'ZOA', 'ZOB', 'ZOC'],
    }
    const { layout, placed } = build(
      spec,
      [
        ['ZLA', 'ZMA'],
        ['ZLB', 'ZMC'],
        ['ZLC', 'ZNB'],
        ['ZLD', 'ZOA'],
        ['ZNA', 'ZOB', 'ended'],
      ],
      claims,
      'P1',
    )
    expectSound(layout, placed, Object.values(spec).flat(), claims)
    expect(layout.marriages.every((m) => m.aligned)).toBe(true)
    const row = (code: string) => layout.rowOf.get(code)!
    // Every heir hangs below the couple of its claim.
    for (const [heir, target] of [
      ['ZMD', 'ZLA'],
      ['ZLE', 'ZMC'],
      ['ZNC', 'ZLC'],
      ['ZOC', 'ZLD'],
    ])
      expect(row(heir)).toBe(row(target) + 1)
    // The nobles with neither claim nor marriage stand in lines above all the trees.
    const trees = ['ZLA', 'ZLB', 'ZLC', 'ZLD', 'ZNA'].map(row)
    for (const code of ['RAI', 'HUO', 'ZOE', 'JEN', 'DRE', 'LOU', 'RIH', 'BOU', 'PHL'])
      expect(row(code)).toBeLessThan(Math.min(...trees))
    // Compact: far narrower than the single row of every noble (24 cards, over 4600 px).
    expect(placed.width).toBeLessThanOrEqual(TREE_BUDGET)
    expect(crossingsOf(layout)).toBe(0)
  })

  it('lays out the crown demo: the royal couple and its heirs in one tree, the others around it', () => {
    const claims: ClaimRef[] = [
      { heir: 'ZAN', target: 'KIN', spouse: 'ZQU', turn: 1 },
      { heir: 'ZPI', target: 'ZQU', spouse: 'KIN', turn: 2 },
      { heir: 'ZGA', target: 'KIN', spouse: 'ZQU', turn: 3 },
    ]
    const spec = {
      P1: ['POP'],
      P2: ['KIN', 'CAR', 'ZPI'],
      P3: ['ZQU', 'ZAN', 'ZGA'],
    }
    const { layout, placed } = build(spec, [['KIN', 'ZQU', 'head']], claims, 'P2')
    expectSound(layout, placed, Object.values(spec).flat(), claims)
    const row = (code: string) => layout.rowOf.get(code)!
    expect(row('KIN')).toBe(row('ZQU'))
    for (const heir of ['ZAN', 'ZPI', 'ZGA']) expect(row(heir)).toBe(row('KIN') + 1)
    // POP has no tree: its line is above. CAR comes between KIN and the heir ZPI: beside KIN.
    expect(row('POP')).toBeLessThan(row('KIN'))
    expect(row('CAR')).toBe(row('KIN'))
    expect(layout.lanes[row('KIN')]).toBe(1)
    expect(crossingsOf(layout)).toBe(0)
  })

  it('draws only the nobles it is given: no claim, no stem', () => {
    const { layout } = build({ P1: ['A1'], P2: ['B1'] }, [['A1', 'B1']], [
      { heir: 'GONE', target: 'A1', spouse: 'B1' },
    ])
    expect(layout.stems).toHaveLength(0)
    expect(layout.rows).toBe(1)
  })
})

describe('placeTree', () => {
  it('draws spouses on one line, in rows as tall as their tallest cartouche', () => {
    const houses = world({ P1: ['A1', 'A2'], P2: ['B1'] })
    const links = buildLinks(houses, [marriage('A2', 'B1')])
    const layout = layoutTree(houses, links, null)
    const lines = new Map([['A1', 3]])
    const placed = placeTree(layout, (code) => lines.get(code) ?? 0)
    const a2 = placed.boxes.get('A2')!
    const b1 = placed.boxes.get('B1')!
    expect(a2.y).toBe(b1.y)
    const [{ d }] = placed.marriages
    expect(d).toBe(`M${a2.x + MEDALLION_W},${a2.y + MEDALLION_ANCHOR_Y} L${b1.x},${b1.y + MEDALLION_ANCHOR_Y}`)
    const a1 = placed.boxes.get('A1')!
    expect(a1.h - a2.h).toBe(3 * MEDALLION_LINE_H + 4)
  })

  it('draws a stem from the knot down to the top of the heir, through a bar in the channel', () => {
    const claims: ClaimRef[] = [{ heir: 'C1', target: 'A1', spouse: 'B1' }]
    const { layout, placed } = build(
      { P1: ['A1'], P2: ['B1'], P3: ['C1'] },
      [['A1', 'B1']],
      claims,
    )
    const heir = placed.boxes.get('C1')!
    const a1 = placed.boxes.get('A1')!
    const { d } = placed.claims[0]
    // Straight down: the heir stands right under the knot.
    const knot = Math.min(a1.x, placed.boxes.get('B1')!.x) + MEDALLION_W + COUPLE_GAP / 2
    expect(d).toBe(`M${knot},${a1.y + MEDALLION_ANCHOR_Y} V${heir.y}`)
    expect(heir.y).toBeGreaterThan(a1.y + a1.h)
    expect(layout.stems[0].lanes).toEqual([0])
  })

  it('draws the heirs of a couple from one bar in the channel, lanes 8px apart', () => {
    const claims: ClaimRef[] = [
      { heir: 'C1', target: 'A1', spouse: 'B1' },
      { heir: 'C2', target: 'A1', spouse: 'B1' },
    ]
    const { layout, placed } = build(
      { P1: ['A1'], P2: ['B1'], P3: ['C1', 'C2'] },
      [['A1', 'B1']],
      claims,
    )
    const a1 = placed.boxes.get('A1')!
    const bar = a1.y + a1.h + 16
    for (const { d } of placed.claims) expect(d).toMatch(new RegExp(` V${bar} H[\\d.]+ V`))
    // A second lane sits one lane step lower.
    const lowered = placeTree(
      { ...layout, stems: layout.stems.map((stem) => ({ ...stem, lanes: [1] })) },
      () => 0,
    )
    expect(lowered.claims[0].d).toContain(` V${bar + TREE_LANE} H`)
  })

  it('writes only axis-aligned commands the vine drawing can read', () => {
    const claims: ClaimRef[] = [
      { heir: 'C1', target: 'A1', spouse: 'B1' },
      { heir: 'C2', target: 'A2', spouse: 'B2' },
      { heir: 'D1', target: 'C1', spouse: 'E1' },
    ]
    const { placed } = build(
      { P1: ['A1', 'A2'], P2: ['B1', 'B2', 'E1'], P3: ['C1', 'C2'], P4: ['D1'] },
      [
        ['A1', 'B1'],
        ['A2', 'B2'],
        ['C1', 'E1'],
      ],
      claims,
    )
    for (const { d } of [...placed.claims, ...placed.marriages])
      expect(d).toMatch(/^M[-\d.]+,[-\d.]+( [VHL][-\d.,]+)+$/)
  })

  it('keeps the keys of a pair married again unique and draws its lines apart', () => {
    const houses = world({ P1: ['A1'], P2: ['B1'] })
    const links = buildLinks(houses, [
      marriage('A1', 'B1', 'ended'),
      { ...marriage('A1', 'B1', 'head'), turn: 7 },
      { ...marriage('A1', 'B1', 'head'), turn: 7 },
    ])
    expect(links.map((link) => link.key)).toEqual(['A1-B1', 'A1-B1#2', 'A1-B1#3'])
    const layout = layoutTree(houses, links, null, [])
    const placed = placeTree(layout, () => 0)
    expect(layout.marriages).toHaveLength(3)
    expect(new Set(placed.marriages.map((m) => m.d)).size).toBe(3)
  })

  it('shows the spouse of every claim it keeps when a house is focused', () => {
    const houses = world({ P1: ['A1', 'A2'], P2: ['B1', 'B2'], P3: ['C1'], P4: ['D1'] })
    const links = buildLinks(houses, [marriage('A1', 'B1'), marriage('A2', 'B2')])
    const shown = visibleNobles(
      houses,
      links,
      [
        // Heir and target are tied to the house by marriage; the spouse is not.
        { heir: 'B1', target: 'B2', spouse: 'C1' },
        // Kept once C1 is shown: its heir is the spouse just added.
        { heir: 'C1', target: 'B1', spouse: 'D1' },
        { heir: 'D1', target: 'B2', spouse: 'D1' },
      ],
      'P1',
    )!
    expect([...shown].sort()).toEqual(['A1', 'A2', 'B1', 'B2', 'C1', 'D1'])
    expect(visibleNobles(houses, links, [], null)).toBeNull()
  })

  it('counts wider capitals as more text than thin letters', () => {
    expect(wrapLines('iiiiiiiiiiiiiiiiiiiiiiiiii')).toBe(1)
    expect(wrapLines('WWWWWWWWWWWWWWWWWW')).toBe(2)
    expect(wrapLines('')).toBe(1)
  })

  it('counts every wrapped line of a long title', () => {
    expect(countLines(['Duchy · ROC'])).toBe(1)
    expect(countLines(['Bishop of Saint-Rémy-de-Provence-en-Alpilles'])).toBe(2)
    expect(countLines(['a', 'b'])).toBe(2)
  })
})
