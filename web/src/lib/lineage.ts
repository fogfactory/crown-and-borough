import type {
  Claim,
  DeceasedNoble,
  Dignity,
  Fief,
  FiefTitle,
  Marriage,
  MarriageCategory,
  Noble,
  Player,
  PlayerId,
} from '@/types'

export interface HeldFief {
  title: FiefTitle
  capital: string
}

export interface HouseMember {
  code: string
  name: string
  sex?: 'male' | 'female'
  dead: boolean
  cause?: DeceasedNoble['cause']
  /** Every dignity the noble carries (each counts as a title). */
  dignities: Dignity[]
  /** Every fief the noble holds, highest first. */
  fiefs: HeldFief[]
  /** The pretension this noble stakes as an heir, if any. */
  claim?: { target: string; targetName: string; rank: number; targetFiefs: HeldFief[] }
  /** How many pretensions bear on this noble's titles. */
  claimedBy: number
}

export interface House {
  player: Player
  members: HouseMember[]
}

export type EdgeKind = 'ended' | 'head' | 'mixed' | 'secondary'

export interface LineageLink {
  key: string
  a: string
  b: string
  houseA?: PlayerId
  houseB?: PlayerId
  active: boolean
  category?: MarriageCategory
  weight?: number
  activeHeadFor: PlayerId[]
  headSuccessors: NonNullable<Marriage['headSuccessors']>
  /** Head weight that is not the active head: a reserve head, drawn as mixed. */
  reserve: boolean
  kind: EdgeKind
}

export interface AlliancePair {
  key: string
  houses: [PlayerId, PlayerId]
  links: LineageLink[]
  weight: number
  /** The strongest kind among the pair's active alliances. */
  kind: Exclude<EdgeKind, 'ended'>
  /** True when one of the alliances is the active head of at least one side. */
  head: boolean
  /** The marriage carrying the alliance: cancelling it weakens or breaks the pair. */
  carrier: LineageLink
}

const KIND_RANK: Record<Exclude<EdgeKind, 'ended'>, number> = {
  head: 3,
  mixed: 2,
  secondary: 1,
}
const FIEF_RANK: Record<FiefTitle, number> = {
  barony: 1,
  county: 2,
  marquisate: 3,
  duchy: 4,
}

/**
 * Builds each house's members: living nobles in succession order, then the
 * deceased by date of death. A noble missing from `player.succession` (older
 * snapshots) keeps its state order after the ones listed.
 */
export function buildHouses(
  players: Player[],
  nobles: Noble[],
  deceased: DeceasedNoble[],
  fiefs: Fief[],
  claims: Claim[] = [],
): House[] {
  const fiefsByHolder = new Map<string, HeldFief[]>()
  for (const fief of fiefs) {
    if (!fief.holder) continue
    const held = fiefsByHolder.get(fief.holder) ?? []
    held.push({ title: fief.title, capital: fief.capital })
    fiefsByHolder.set(fief.holder, held)
  }
  for (const held of fiefsByHolder.values())
    held.sort((a, b) => FIEF_RANK[b.title] - FIEF_RANK[a.title])
  const nameByCode = new Map(nobles.map((noble) => [noble.code, noble.name]))
  const claimByHeir = new Map(claims.map((claim) => [claim.heir, claim]))
  const claimedBy = new Map<string, number>()
  for (const claim of claims)
    claimedBy.set(claim.target, (claimedBy.get(claim.target) ?? 0) + 1)

  return players.map((player) => {
    const own = nobles.filter((noble) => noble.owner === player.id)
    const order = new Map((player.succession ?? []).map((code, index) => [code, index]))
    const living = [...own].sort(
      (a, b) => (order.get(a.code) ?? order.size) - (order.get(b.code) ?? order.size),
    )
    const members: HouseMember[] = living.map((noble) => {
      const claim = claimByHeir.get(noble.code)
      return {
        code: noble.code,
        name: noble.name,
        sex: noble.sex,
        dead: false,
        dignities: noble.dignities ?? [],
        fiefs: fiefsByHolder.get(noble.code) ?? [],
        claim: claim
          ? {
              target: claim.target,
              targetName: nameByCode.get(claim.target) ?? claim.target,
              rank: claim.rank,
              targetFiefs: fiefsByHolder.get(claim.target) ?? [],
            }
          : undefined,
        claimedBy: claimedBy.get(noble.code) ?? 0,
      }
    })
    const dead = deceased
      .filter((noble) => noble.owner === player.id)
      .sort((a, b) => a.turn - b.turn)
    for (const noble of dead) {
      members.push({
        code: noble.code,
        name: noble.name,
        sex: noble.sex,
        dead: true,
        cause: noble.cause,
        dignities: [],
        fiefs: [],
        claimedBy: 0,
      })
    }
    return { player, members }
  })
}

export function buildLinks(houses: House[], marriages: Marriage[]): LineageLink[] {
  const houseByCode = new Map<string, PlayerId>()
  for (const house of houses) {
    for (const member of house.members) houseByCode.set(member.code, house.player.id)
  }
  return marriages.map((marriage) => {
    const active = marriage.active ?? true
    const activeHeadFor = marriage.activeHeadFor ?? []
    const isActiveHead = activeHeadFor.length > 0
    const reserve = active && marriage.category === 'head' && !isActiveHead
    let kind: EdgeKind = 'ended'
    if (active && marriage.category) {
      if (isActiveHead) kind = 'head'
      else if (reserve || marriage.category === 'mixed') kind = 'mixed'
      else kind = 'secondary'
    }
    return {
      key: `${marriage.nobleA}-${marriage.nobleB}`,
      a: marriage.nobleA,
      b: marriage.nobleB,
      houseA: houseByCode.get(marriage.nobleA),
      houseB: houseByCode.get(marriage.nobleB),
      active,
      category: marriage.category,
      weight: marriage.weight,
      activeHeadFor,
      headSuccessors: marriage.headSuccessors ?? [],
      reserve,
      kind,
    }
  })
}

/** Groups the active alliances between the same two houses into one edge. */
export function buildAlliancePairs(links: LineageLink[]): AlliancePair[] {
  const pairs = new Map<string, AlliancePair>()
  for (const link of links) {
    if (
      link.kind === 'ended' ||
      !link.houseA ||
      !link.houseB ||
      link.houseA === link.houseB
    )
      continue
    const kind = link.kind
    const houses = [link.houseA, link.houseB].sort() as [PlayerId, PlayerId]
    const key = houses.join('|')
    const pair = pairs.get(key) ?? {
      key,
      houses,
      links: [],
      weight: 0,
      kind,
      head: false,
      carrier: link,
    }
    pair.links.push(link)
    pair.weight += link.weight ?? 0
    if (KIND_RANK[kind] > KIND_RANK[pair.kind]) pair.kind = kind
    if (link.activeHeadFor.length > 0) pair.head = true
    pairs.set(key, pair)
  }
  for (const pair of pairs.values()) {
    pair.carrier = [...pair.links].sort(
      (x, y) =>
        Number(y.activeHeadFor.length > 0) - Number(x.activeHeadFor.length > 0) ||
        (y.weight ?? 0) - (x.weight ?? 0),
    )[0]
  }
  return [...pairs.values()]
}

export const CARD_W = 208
export const CARD_H = 100
export const GAP_X = 80
export const PITCH = 112

/** Top-left corner of a card, in pixels. */
export interface Box {
  x: number
  y: number
}

/** The straight line of a marriage, from the right edge of the left card to the left edge of the right one. */
export function marriageSegment(a: Box, b: Box) {
  const [left, right] = a.x <= b.x ? [a, b] : [b, a]
  return {
    x1: left.x + CARD_W,
    y1: left.y + CARD_H / 2,
    x2: right.x,
    y2: right.y + CARD_H / 2,
  }
}

/**
 * Route of a claim, drawn as a filiation: from the heir's card, horizontally
 * to the middle of the marriage line between the target and the spouse, then
 * vertically into that line. Only horizontal and vertical strokes.
 */
export function claimRoute(heir: Box, target: Box, spouse: Box): Array<[number, number]> {
  const { x1, y1, x2, y2 } = marriageSegment(target, spouse)
  const midX = (x1 + x2) / 2
  const midY = (y1 + y2) / 2
  const startY = heir.y + CARD_H / 2
  const startX = midX <= heir.x ? heir.x : heir.x + CARD_W
  return [
    [startX, startY],
    [midX, startY],
    [midX, midY],
  ]
}

export interface ClaimRef {
  heir: string
  target: string
  spouse: string
}

export interface ColumnLayout {
  house: House
  /** Row of the house's first member, so linked spouses share a row. */
  offset: number
}

/**
 * Orders the houses as columns and places each on rows:
 *
 * 1. With a focus house, the houses linked to it by a marriage surround it by
 *    distance, and each house is shifted so that the spouse of its strongest
 *    marriage towards an already placed house stands on that spouse's row.
 *    Without a focus, houses keep their order and start on row 0.
 * 2. The column order is then permuted (for up to 7 houses) to minimise the
 *    links that skip a column, preferring the order of step 1 on ties.
 * 3. Finally, a straight link that would cross a card of a column in between
 *    pushes that column one row down, repeatedly, until it passes through a
 *    gap. The focus house never moves.
 *
 * Claims are drawn as filiations (see `claimRoute`): when one would run over
 * a card, the column of that card moves down by half a row, which lets the
 * stroke pass between two cards.
 */
export function layoutColumns(
  houses: House[],
  links: LineageLink[],
  focus: PlayerId | null,
  claims: ClaimRef[] = [],
): { columns: ColumnLayout[]; rows: number } {
  const pairs: Array<{ a: string; b: string; weight: number }> = [
    ...links.map((link) => ({ a: link.a, b: link.b, weight: link.active ? 3 : 1 })),
    ...claims.map(({ heir, target }) => ({ a: heir, b: target, weight: 1 })),
  ]
  const linkPairs = pairs.slice(0, links.length)
  const houseOf = new Map<string, House>()
  for (const house of houses)
    for (const member of house.members) houseOf.set(member.code, house)
  const focusHouse = houses.find((house) => house.player.id === focus)
  const offsets = new Map<PlayerId, number>(houses.map((house) => [house.player.id, 0]))
  let sequence = houses

  if (focusHouse) {
    const rowOf = new Map<string, number>()
    const place = (house: House) => {
      const base = offsets.get(house.player.id) ?? 0
      house.members.forEach((member, index) => rowOf.set(member.code, base + index))
    }
    place(focusHouse)
    const order: House[] = []
    const placed = new Set<PlayerId>([focusHouse.player.id])
    const strength = (link: LineageLink) => (link.active ? 1000 : 0) + (link.weight ?? 0)
    for (;;) {
      let best:
        { link: LineageLink; fromCode: string; house: House; toCode: string } | undefined
      for (const link of links) {
        const ha = houseOf.get(link.a)
        const hb = houseOf.get(link.b)
        if (!ha || !hb) continue
        for (const [from, to, fromCode, toCode] of [
          [ha, hb, link.a, link.b],
          [hb, ha, link.b, link.a],
        ] as const) {
          if (!placed.has(from.player.id) || placed.has(to.player.id)) continue
          if (!best || strength(link) > strength(best.link))
            best = { link, fromCode, house: to, toCode }
        }
      }
      if (!best) break
      const spouseIndex = best.house.members.findIndex(
        (member) => member.code === best!.toCode,
      )
      offsets.set(best.house.player.id, (rowOf.get(best.fromCode) ?? 0) - spouseIndex)
      placed.add(best.house.player.id)
      place(best.house)
      order.push(best.house)
    }
    const unlinked = houses.filter((house) => !placed.has(house.player.id))
    const left: House[] = []
    const right: House[] = []
    order.forEach((house, index) => (index % 2 === 0 ? right : left).push(house))
    sequence = [...left.reverse(), focusHouse, ...right, ...unlinked]
  }

  sequence = leastSkippingOrder(sequence, pairs, houseOf)

  const column = new Map<PlayerId, number>(
    sequence.map((house, index) => [house.player.id, index]),
  )
  const rowOfCode = (code: string) => {
    const house = houseOf.get(code)!
    const index = house.members.findIndex((member) => member.code === code)
    return (offsets.get(house.player.id) ?? 0) + index
  }
  const cardTop = (row: number) => row * PITCH
  for (let guard = 0; guard < 60; guard++) {
    let blocked: PlayerId | undefined
    let step = 1
    for (const { a, b } of linkPairs) {
      const ha = houseOf.get(a)
      const hb = houseOf.get(b)
      if (!ha || !hb) continue
      const ca = column.get(ha.player.id)!
      const cb = column.get(hb.player.id)!
      if (Math.abs(ca - cb) < 2) continue
      const [left, right, leftCode, rightCode] = ca < cb ? [ca, cb, a, b] : [cb, ca, b, a]
      const x1 = left * (CARD_W + GAP_X) + CARD_W
      const y1 = cardTop(rowOfCode(leftCode)) + CARD_H / 2
      const x2 = right * (CARD_W + GAP_X)
      const y2 = cardTop(rowOfCode(rightCode)) + CARD_H / 2
      const yAt = (x: number) => y1 + ((y2 - y1) * (x - x1)) / (x2 - x1)
      for (let k = left + 1; k < right && !blocked; k++) {
        const mid = sequence[k]
        if (mid.player.id === focusHouse?.player.id) continue
        const spanFrom = yAt(k * (CARD_W + GAP_X))
        const spanTo = yAt(k * (CARD_W + GAP_X) + CARD_W)
        const lo = Math.min(spanFrom, spanTo) - 4
        const hi = Math.max(spanFrom, spanTo) + 4
        const base = offsets.get(mid.player.id) ?? 0
        const hit = mid.members.some((_, index) => {
          const top = cardTop(base + index)
          return hi > top && lo < top + CARD_H
        })
        if (hit) blocked = mid.player.id
      }
      if (blocked) break
    }
    if (!blocked) {
      const boxOf = (code: string): Box | undefined => {
        const house = houseOf.get(code)
        if (!house) return undefined
        return {
          x: column.get(house.player.id)! * (CARD_W + GAP_X),
          y: cardTop(rowOfCode(code)),
        }
      }
      for (const claim of claims) {
        const heir = boxOf(claim.heir)
        const target = boxOf(claim.target)
        const spouse = boxOf(claim.spouse)
        const heirHouse = houseOf.get(claim.heir)
        if (!heir || !target || !spouse || !heirHouse) continue
        if (
          !links.some(
            (l) =>
              (l.a === claim.target && l.b === claim.spouse) ||
              (l.b === claim.target && l.a === claim.spouse),
          )
        )
          continue
        const [start, corner, end] = claimRoute(heir, target, spouse)
        for (let k = 0; k < sequence.length && !blocked; k++) {
          const mid = sequence[k]
          if (mid === heirHouse) continue
          const midIsFocus = mid.player.id === focusHouse?.player.id
          // The focus house never moves: its heir's house moves instead.
          if (midIsFocus && heirHouse.player.id === focusHouse?.player.id) continue
          const left = k * (CARD_W + GAP_X)
          const base = offsets.get(mid.player.id) ?? 0
          const hit = mid.members.some((_, index) => {
            const top = cardTop(base + index)
            const inY = (y: number) => y > top - 4 && y < top + CARD_H + 4
            const inX = (x: number) => x > left - 4 && x < left + CARD_W + 4
            const hRun =
              Math.min(start[0], corner[0]) < left + CARD_W &&
              Math.max(start[0], corner[0]) > left
            const vRun =
              Math.min(corner[1], end[1]) < top + CARD_H &&
              Math.max(corner[1], end[1]) > top
            return (hRun && inY(start[1])) || (vRun && inX(corner[0]))
          })
          if (hit) {
            blocked = midIsFocus ? heirHouse.player.id : mid.player.id
            step = 0.5
          }
        }
        if (blocked) break
      }
    }
    if (!blocked) break
    offsets.set(blocked, (offsets.get(blocked) ?? 0) + step)
  }

  const min = Math.min(...sequence.map((house) => offsets.get(house.player.id) ?? 0))
  const columns = sequence.map((house) => ({
    house,
    offset: (offsets.get(house.player.id) ?? 0) - min,
  }))
  const rows = Math.max(1, ...columns.map((c) => c.offset + c.house.members.length))
  return { columns, rows }
}

/** The column order with the fewest skipped columns, keeping `start` when equally good. */
function leastSkippingOrder(
  start: House[],
  pairs: Array<{ a: string; b: string; weight: number }>,
  houseOf: Map<string, House>,
): House[] {
  if (start.length < 3 || start.length > 7) return start
  const cost = (order: House[]) => {
    const column = new Map(order.map((house, index) => [house.player.id, index]))
    let total = 0
    for (const { a, b, weight } of pairs) {
      const ha = houseOf.get(a)
      const hb = houseOf.get(b)
      if (!ha || !hb) continue
      total +=
        weight *
        Math.max(0, Math.abs(column.get(ha.player.id)! - column.get(hb.player.id)!) - 1)
    }
    return total
  }
  const displacement = (order: House[]) =>
    order.reduce((sum, house, index) => sum + Math.abs(index - start.indexOf(house)), 0)
  let best = start
  let bestCost = cost(start)
  let bestMove = 0
  const permute = (rest: House[], chosen: House[]) => {
    if (rest.length === 0) {
      const c = cost(chosen)
      const move = displacement(chosen)
      if (c < bestCost || (c === bestCost && move < bestMove)) {
        best = chosen
        bestCost = c
        bestMove = move
      }
      return
    }
    rest.forEach((house, index) =>
      permute([...rest.slice(0, index), ...rest.slice(index + 1)], [...chosen, house]),
    )
  }
  permute(start, [])
  return best
}
