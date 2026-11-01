import type {
  Claim,
  DeceasedNoble,
  Bishopric,
  Dignity,
  Excommunication,
  Fief,
  FiefTitle,
  ReligiousTitle,
  Marriage,
  MarriageCategory,
  Noble,
  NobleStatus,
  Player,
  PlayerId,
} from '@/types'

/** A religious or royal title a noble holds; `seat` names the bishopric of a bishop. */
export interface HeldReligiousTitle {
  title: ReligiousTitle | 'king' | 'queen' | 'dowager_queen'
  seat?: string
}

/** The religious and royal standings buildHouses reads titles from. */
export interface Religion {
  bishoprics?: Bishopric[]
  cardinals?: string[]
  pope?: string
  /** Code of the noble who holds the crown. */
  king?: string
  /** Code of the queen; a dowager queen is not the spouse of the king in place. */
  queen?: string
  queenDowager?: boolean
  excommunicated?: Excommunication[]
}

export interface HeldFief {
  title: FiefTitle
  capital: string
}

export interface HouseMember {
  code: string
  name: string
  sex?: 'male' | 'female'
  /** Captivity of a living noble; the deceased have none. */
  status?: NobleStatus
  dead: boolean
  cause?: DeceasedNoble['cause']
  /** Commissioner of this noble's assassination, when the viewer knows it. */
  commissioner?: { code: string; name: string }
  /** Victims of the assassinations this noble commissioned, when the viewer knows them. */
  commissioned: { code: string; name: string }[]
  /** Every dignity the noble carries (each counts as a title). */
  dignities: Dignity[]
  /** Every religious title the noble holds, highest first. */
  religious: HeldReligiousTitle[]
  /** The noble is excommunicated (its religious titles are lost). */
  excommunicated: boolean
  /** Every fief the noble holds, highest first. */
  fiefs: HeldFief[]
  /** The pretension this noble stakes as an heir, if any. */
  claim?: {
    target: string
    targetName: string
    rank: number
    targetFiefs: HeldFief[]
    /** Set once the claim ended: it got its way (honoured) or not (lapsed). */
    outcome?: 'honoured' | 'lapsed'
  }
  /** How many pretensions bear on this noble's titles. */
  claimedBy: number
}

export interface House {
  player: Player
  members: HouseMember[]
}

export type EdgeKind = 'ended' | 'head' | 'secondary'

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
  head: 2,
  secondary: 1,
}
const FIEF_RANK: Record<FiefTitle, number> = {
  barony: 1,
  county: 2,
  marquisate: 3,
  duchy: 4,
}

/** Royal and religious titles by noble code, highest first (king, queen, pope, cardinal, bishop). */
export function religiousTitlesByHolder(
  religion: Religion,
): Map<string, HeldReligiousTitle[]> {
  const byHolder = new Map<string, HeldReligiousTitle[]>()
  const add = (code: string | undefined, held: HeldReligiousTitle) => {
    if (!code) return
    byHolder.set(code, [...(byHolder.get(code) ?? []), held])
  }
  add(religion.king, { title: 'king' })
  add(religion.queen, { title: religion.queenDowager ? 'dowager_queen' : 'queen' })
  add(religion.pope, { title: 'pope' })
  for (const code of religion.cardinals ?? []) add(code, { title: 'cardinal' })
  for (const bishopric of religion.bishoprics ?? [])
    add(bishopric.bishop, { title: 'bishop', seat: bishopric.name })
  return byHolder
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
  religion: Religion = {},
): House[] {
  const religiousByHolder = religiousTitlesByHolder(religion)
  const excommunicatedCodes = new Set(
    (religion.excommunicated ?? []).map((entry) => entry.noble),
  )
  const fiefsByHolder = new Map<string, HeldFief[]>()
  for (const fief of fiefs) {
    if (!fief.holder) continue
    const held = fiefsByHolder.get(fief.holder) ?? []
    held.push({ title: fief.title, capital: fief.capital })
    fiefsByHolder.set(fief.holder, held)
  }
  for (const held of fiefsByHolder.values())
    held.sort((a, b) => FIEF_RANK[b.title] - FIEF_RANK[a.title])
  const nameByCode = new Map([
    ...deceased.map((noble): [string, string] => [noble.code, noble.name]),
    ...nobles.map((noble): [string, string] => [noble.code, noble.name]),
  ])
  const namedCodes = (codes: string[] = []) =>
    codes.map((code) => ({ code, name: nameByCode.get(code) ?? code }))
  // An heir shows its live claim, else its latest ended one.
  const claimByHeir = new Map<string, Claim>()
  for (const claim of claims) {
    const shown = claimByHeir.get(claim.heir)
    if (!shown || shown.ended || !claim.ended) claimByHeir.set(claim.heir, claim)
  }
  const claimedBy = new Map<string, number>()
  for (const claim of claims)
    if (!claim.ended) claimedBy.set(claim.target, (claimedBy.get(claim.target) ?? 0) + 1)

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
        status: noble.status,
        dead: false,
        commissioned: namedCodes(noble.commissioned),
        dignities: noble.dignities ?? [],
        religious: religiousByHolder.get(noble.code) ?? [],
        excommunicated: excommunicatedCodes.has(noble.code),
        fiefs: fiefsByHolder.get(noble.code) ?? [],
        claim: claim
          ? {
              target: claim.target,
              targetName: nameByCode.get(claim.target) ?? claim.target,
              rank: claim.rank,
              targetFiefs: claim.ended ? [] : (fiefsByHolder.get(claim.target) ?? []),
              outcome: claim.ended ? (claim.outcome ?? 'lapsed') : undefined,
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
        commissioner: noble.commissioner
          ? {
              code: noble.commissioner,
              name: nameByCode.get(noble.commissioner) ?? noble.commissioner,
            }
          : undefined,
        commissioned: namedCodes(noble.commissioned),
        dignities: [],
        religious: [],
        excommunicated: false,
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
    let kind: EdgeKind = 'ended'
    if (active && marriage.category) kind = isActiveHead ? 'head' : 'secondary'
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

/** A claim as the tree draws it: a stem from the couple (target and spouse) down to the heir. */
export interface ClaimRef {
  heir: string
  target: string
  spouse: string
  ended?: boolean
}

/** A marriage between two nobles of the tree, left column first. */
export interface TreeMarriage {
  link: LineageLink
  /** Both spouses stand on the same row; otherwise the line bends in a gap. */
  aligned: boolean
  left: string
  right: string
}

/** The stem of a claim: it leaves its couple in a gap and runs down to the heir. */
export interface ClaimStem {
  /** Index of the claim in the list given to `layoutTree`. */
  claim: number
  /** Stems of the same couple share a lane. */
  couple: string
  /** Gap `g` lies between columns `g` and `g + 1`. */
  gap: number
  lane: number
  heir: string
  target: string
  /** Noble whose row the stem leaves from (the marriage line or the target). */
  anchor: string
  /** No marriage line is drawn for the couple: the stem leaves the target itself. */
  fromTarget: boolean
}

export interface TreeLayout {
  /** The houses, one column each, in drawing order. */
  columns: House[]
  /** Integer row of every noble; strictly increasing down a house. */
  rowOf: Map<string, number>
  rows: number
  marriages: TreeMarriage[]
  stems: ClaimStem[]
  /** Lanes taken in each gap, claims and bent marriages together. */
  lanes: number[]
  /** Lane of the vertical stroke of each bent marriage, by link key, in its gap. */
  bendLane: Map<string, number>
}

const KIND_ORDER: Record<EdgeKind, number> = { head: 3, secondary: 2, ended: 1 }

/**
 * Lays the houses out as columns and the nobles on integer rows:
 *
 * 1. The columns: with a focus house, the houses linked to it by a marriage
 *    surround it by distance; the order is then improved to minimise the links
 *    that skip a column (all permutations up to 7 houses, a barycentre sweep
 *    and adjacent swaps beyond), keeping the first order on ties.
 * 2. Rows grow strictly down a house, in succession order. Marriages are taken
 *    strongest first (active head, secondary, ended, then weight) and put both
 *    spouses on one row when that keeps the rows consistent; a marriage that
 *    would contradict stronger ones stays `aligned: false`. An heir is put
 *    below its parents' row when that contradicts nothing.
 * 3. A marriage line running over columns in between needs its row to be free
 *    there: the nobles in the way are pushed down.
 * 4. One stem per couple leaves in the gap beside its heir's column, each
 *    couple on its own lane.
 */
export function layoutTree(
  houses: House[],
  links: LineageLink[],
  focus: PlayerId | null,
  claims: ClaimRef[] = [],
): TreeLayout {
  const houseOf = new Map<string, House>()
  for (const house of houses)
    for (const member of house.members) houseOf.set(member.code, house)

  const columns = orderColumns(houses, links, focus, claims, houseOf)
  const columnOf = new Map<string, number>()
  columns.forEach((house, index) => {
    for (const member of house.members) columnOf.set(member.code, index)
  })

  // Nodes in drawing order; a node's index decides which one roots a merged group.
  const nodes: string[] = []
  for (const house of columns) for (const member of house.members) nodes.push(member.code)
  const nodeIndex = new Map(nodes.map((code, index) => [code, index]))
  const chain: Array<[string, string]> = []
  for (const house of columns)
    for (let i = 1; i < house.members.length; i++)
      chain.push([house.members[i - 1].code, house.members[i].code])

  /** Groups of nobles sharing a row, or null when the constraints contradict each other. */
  const groupsOf = (merged: Array<[string, string]>, below: Array<[string, string]>) => {
    const parent = new Map(nodes.map((code) => [code, code]))
    const find = (code: string): string => {
      let root = code
      while (parent.get(root) !== root) root = parent.get(root)!
      return root
    }
    for (const [a, b] of merged) {
      const ra = find(a)
      const rb = find(b)
      if (ra === rb) continue
      if (nodeIndex.get(ra)! < nodeIndex.get(rb)!) parent.set(rb, ra)
      else parent.set(ra, rb)
    }
    const roots = nodes.filter((code) => find(code) === code)
    const successors = new Map<string, Set<string>>(
      roots.map((root) => [root, new Set()]),
    )
    const indegree = new Map<string, number>(roots.map((root) => [root, 0]))
    for (const [a, b] of [...chain, ...below]) {
      const from = find(a)
      const to = find(b)
      if (successors.get(from)!.has(to)) continue
      successors.get(from)!.add(to)
      indegree.set(to, indegree.get(to)! + 1)
    }
    const order: string[] = []
    const ready = roots.filter((root) => indegree.get(root) === 0)
    while (ready.length > 0) {
      const root = ready.shift()!
      order.push(root)
      for (const next of successors.get(root)!) {
        indegree.set(next, indegree.get(next)! - 1)
        if (indegree.get(next) === 0) ready.push(next)
      }
    }
    if (order.length < roots.length) return null
    return { find, order, successors }
  }

  // Marriages, strongest first.
  const merged: Array<[string, string]> = []
  const aligned = new Set<string>()
  const candidates = links
    .filter((link) => columnOf.has(link.a) && columnOf.has(link.b))
    .sort(
      (x, y) =>
        KIND_ORDER[y.kind] - KIND_ORDER[x.kind] ||
        (y.weight ?? 0) - (x.weight ?? 0) ||
        (x.key < y.key ? -1 : x.key > y.key ? 1 : 0),
    )
  for (const link of candidates) {
    if (columnOf.get(link.a) === columnOf.get(link.b)) continue
    if (groupsOf([...merged, [link.a, link.b]], [])) {
      merged.push([link.a, link.b])
      aligned.add(link.key)
    }
  }

  // Heirs below the couple they claim through, when nothing forbids it.
  const below: Array<[string, string]> = []
  for (const claim of claims) {
    for (const parent of [claim.spouse, claim.target]) {
      if (!columnOf.has(parent) || !columnOf.has(claim.heir)) continue
      if (groupsOf(merged, [...below, [parent, claim.heir]]))
        below.push([parent, claim.heir])
    }
  }

  const marriages: TreeMarriage[] = links
    .filter((link) => columnOf.has(link.a) && columnOf.has(link.b))
    .map((link) => {
      const [left, right] =
        columnOf.get(link.a)! <= columnOf.get(link.b)!
          ? [link.a, link.b]
          : [link.b, link.a]
      return { link, aligned: aligned.has(link.key), left, right }
    })

  // Rows: longest path, then sources pulled down against their successors; the
  // nobles in the way of a marriage line over intermediate columns are pushed down.
  const groups = groupsOf(merged, below)!
  const rowOfRoot = new Map<string, number>()
  const predecessors = new Map<string, string[]>()
  for (const [from, to] of groups.successors)
    for (const next of to)
      predecessors.set(next, [...(predecessors.get(next) ?? []), from])
  for (const root of groups.order) {
    rowOfRoot.set(
      root,
      Math.max(
        0,
        ...(predecessors.get(root) ?? []).map((from) => rowOfRoot.get(from)! + 1),
      ),
    )
  }
  for (const root of [...groups.order].reverse()) {
    const next = [...groups.successors.get(root)!]
    if (next.length === 0) continue
    rowOfRoot.set(
      root,
      Math.max(
        rowOfRoot.get(root)!,
        Math.min(...next.map((to) => rowOfRoot.get(to)!)) - 1,
      ),
    )
  }
  // A marriage line running over intermediate columns needs a free row there:
  // a blank row opens under it, every group but the marriage's own moving down
  // by one (groups move whole, so alignments and orders hold).
  for (let guard = 0; guard < 200; guard++) {
    const blocked = marriages.find(({ left, right }) => {
      const row = rowOfRoot.get(groups.find(left))!
      for (let c = columnOf.get(left)! + 1; c < columnOf.get(right)!; c++)
        for (const member of columns[c].members)
          if (
            rowOfRoot.get(groups.find(member.code)) === row &&
            groups.find(member.code) !== groups.find(left)
          )
            return true
      return false
    })
    if (!blocked) break
    const own = groups.find(blocked.left)
    const row = rowOfRoot.get(own)!
    for (const [root, at] of rowOfRoot)
      if (root !== own && at >= row) rowOfRoot.set(root, at + 1)
  }
  const rowOf = new Map<string, number>()
  for (const code of nodes) rowOf.set(code, rowOfRoot.get(groups.find(code))!)
  const rows = Math.max(-1, ...rowOf.values()) + 1

  // Lanes: bent marriages first, then one per couple that stakes claims.
  const lanes = new Array<number>(Math.max(0, columns.length - 1)).fill(0)
  const bendLane = new Map<string, number>()
  for (const marriage of marriages) {
    const from = columnOf.get(marriage.left)!
    const to = columnOf.get(marriage.right)!
    if (marriage.aligned || from === to) continue
    bendLane.set(marriage.link.key, lanes[to - 1]++)
  }
  const stems: ClaimStem[] = []
  const laneOfCouple = new Map<string, number>()
  claims.forEach((claim, index) => {
    const heir = columnOf.get(claim.heir)
    const target = columnOf.get(claim.target)
    if (heir === undefined || target === undefined || heir === target) return
    const couple = [claim.target, claim.spouse].sort().join('-')
    const marriage = marriages.find(
      ({ link }) =>
        (link.a === claim.target && link.b === claim.spouse) ||
        (link.b === claim.target && link.a === claim.spouse),
    )
    const gap = heir < target ? heir : heir - 1
    let anchor = claim.target
    if (marriage) {
      const near = (code: string) => Math.abs(columnOf.get(code)! - heir)
      anchor =
        near(marriage.left) <= near(marriage.right) ? marriage.left : marriage.right
    }
    const id = `${gap}:${couple}`
    if (!laneOfCouple.has(id)) laneOfCouple.set(id, lanes[gap]++)
    stems.push({
      claim: index,
      couple,
      gap,
      lane: laneOfCouple.get(id)!,
      heir: claim.heir,
      target: claim.target,
      anchor,
      fromTarget: !marriage,
    })
  })

  return { columns, rowOf, rows, marriages, stems, lanes, bendLane }
}

/**
 * Orders the houses as columns: around the focus house by marriage distance,
 * then improved to skip as few columns as possible.
 */
function orderColumns(
  houses: House[],
  links: LineageLink[],
  focus: PlayerId | null,
  claims: ClaimRef[],
  houseOf: Map<string, House>,
): House[] {
  const pairs: Array<{ a: string; b: string; weight: number }> = [
    ...links.map((link) => ({ a: link.a, b: link.b, weight: link.active ? 3 : 1 })),
    ...claims.map(({ heir, target }) => ({ a: heir, b: target, weight: 1 })),
  ]
  const focusHouse = houses.find((house) => house.player.id === focus)
  let sequence = houses
  if (focusHouse) {
    // Ties go to the first key, not to the first link received.
    const byKey = [...links].sort((x, y) => (x.key < y.key ? -1 : x.key > y.key ? 1 : 0))
    const order: House[] = []
    const placed = new Set<PlayerId>([focusHouse.player.id])
    // Breadth first: the houses married to the placed ones, strongest link first.
    for (;;) {
      let best: { weight: number; house: House } | undefined
      for (const link of byKey) {
        const ha = houseOf.get(link.a)
        const hb = houseOf.get(link.b)
        if (!ha || !hb) continue
        const weight = (link.active ? 1000 : 0) + (link.weight ?? 0)
        for (const [from, to] of [
          [ha, hb],
          [hb, ha],
        ]) {
          if (!placed.has(from.player.id) || placed.has(to.player.id)) continue
          if (!best || weight > best.weight) best = { weight, house: to }
        }
      }
      if (!best) break
      placed.add(best.house.player.id)
      order.push(best.house)
    }
    const unlinked = houses.filter((house) => !placed.has(house.player.id))
    const left: House[] = []
    const right: House[] = []
    order.forEach((house, index) => (index % 2 === 0 ? right : left).push(house))
    sequence = [...left.reverse(), focusHouse, ...right, ...unlinked]
  }
  return leastSkippingOrder(sequence, pairs, houseOf)
}

const MAX_PERMUTED_HOUSES = 7

/** The column order with the fewest skipped columns, keeping `start` when equally good. */
function leastSkippingOrder(
  start: House[],
  pairs: Array<{ a: string; b: string; weight: number }>,
  houseOf: Map<string, House>,
): House[] {
  if (start.length < 3) return start
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
  let best = start
  let bestCost = cost(start)
  if (start.length <= MAX_PERMUTED_HOUSES) {
    const displacement = (order: House[]) =>
      order.reduce((sum, house, index) => sum + Math.abs(index - start.indexOf(house)), 0)
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

  // Too many houses to try every order: sweep by barycentre of the linked
  // houses, then swap neighbours while that helps.
  const weights = new Map<PlayerId, Map<PlayerId, number>>()
  for (const { a, b, weight } of pairs) {
    const ha = houseOf.get(a)?.player.id
    const hb = houseOf.get(b)?.player.id
    if (!ha || !hb || ha === hb) continue
    for (const [x, y] of [
      [ha, hb],
      [hb, ha],
    ]) {
      const row = weights.get(x) ?? new Map<PlayerId, number>()
      row.set(y, (row.get(y) ?? 0) + weight)
      weights.set(x, row)
    }
  }
  for (let sweep = 0; sweep < 12; sweep++) {
    const position = new Map(best.map((house, index) => [house.player.id, index]))
    const barycentre = (house: House) => {
      const row = weights.get(house.player.id)
      if (!row || row.size === 0) return position.get(house.player.id)!
      let sum = 0
      let total = 0
      for (const [other, weight] of row) {
        sum += weight * position.get(other)!
        total += weight
      }
      return sum / total
    }
    const candidate = [...best].sort(
      (x, y) =>
        barycentre(x) - barycentre(y) ||
        position.get(x.player.id)! - position.get(y.player.id)!,
    )
    const c = cost(candidate)
    if (c >= bestCost) break
    best = candidate
    bestCost = c
  }
  for (let pass = 0; pass < 50; pass++) {
    let improved = false
    for (let i = 0; i + 1 < best.length; i++) {
      const swapped = [...best]
      ;[swapped[i], swapped[i + 1]] = [swapped[i + 1], swapped[i]]
      const c = cost(swapped)
      if (c < bestCost) {
        best = swapped
        bestCost = c
        improved = true
      }
    }
    if (!improved) break
  }
  return best
}

/** Width of a medallion, in pixels. */
export const MEDALLION_W = 176
/** Height of a medallion with no text line under its name: roundel and name. */
export const MEDALLION_BASE_H = 64
/** Radius of the roundel drawn around a noble's pawn. */
export const MEDALLION_ROUNDEL_R = 21
/** Height of one text line under the name. */
export const MEDALLION_LINE_H = 16
/** Characters of a text line that fit in a medallion before it wraps. */
export const MEDALLION_LINE_CHARS = 24
/** Room above the first row for the arms and name of each house. */
export const TREE_HEADER_H = 52
/** Vertical position of the lines (marriages, stems): the roundel's centre. */
export const MEDALLION_ANCHOR_Y = 22
export const TREE_ROW_GAP = 14
export const TREE_LANE = 8
const TREE_GAP_MIN = 56

/** Lines taken by texts of the given lengths once wrapped to `perLine` characters. */
export function countLines(texts: string[], perLine = MEDALLION_LINE_CHARS): number {
  return texts.reduce(
    (sum, text) => sum + Math.max(1, Math.ceil(text.length / perLine)),
    0,
  )
}

export interface TreeBox {
  x: number
  y: number
  w: number
  h: number
}

export interface PlacedMarriage {
  link: LineageLink
  aligned: boolean
  /** Centre line of the marriage, as an SVG path. */
  d: string
}

export interface PlacedClaim {
  /** Index of the claim in the list given to `layoutTree`. */
  claim: number
  d: string
}

export interface TreePlacement {
  width: number
  height: number
  /** Left edge of each column, in layout order. */
  columnX: number[]
  boxes: Map<string, TreeBox>
  marriages: PlacedMarriage[]
  claims: PlacedClaim[]
}

/**
 * Turns a layout into pixels. A row is as tall as its tallest medallion, so
 * spouses on one row share the height of their marriage line; the height of a
 * medallion comes from the number of text lines under its name
 * (`medallionLines`), never from the DOM.
 */
export function placeTree(
  layout: TreeLayout,
  medallionLines: (code: string) => number,
): TreePlacement {
  const { columns, rowOf, rows, lanes } = layout
  const heightOf = (code: string) =>
    MEDALLION_BASE_H +
    medallionLines(code) * MEDALLION_LINE_H +
    (medallionLines(code) > 0 ? 4 : 0)

  const rowHeight = new Array<number>(rows).fill(0)
  for (const house of columns)
    for (const member of house.members) {
      const row = rowOf.get(member.code)!
      rowHeight[row] = Math.max(rowHeight[row], heightOf(member.code))
    }
  const rowTop: number[] = []
  let top = TREE_HEADER_H
  for (let row = 0; row < rows; row++) {
    rowTop.push(top)
    top += rowHeight[row] + (rowHeight[row] > 0 ? TREE_ROW_GAP : 0)
  }

  const gapWidth = lanes.map((count) => Math.max(TREE_GAP_MIN, 24 + TREE_LANE * count))
  const columnX: number[] = []
  let x = 0
  columns.forEach((_, index) => {
    columnX.push(x)
    x += MEDALLION_W + (gapWidth[index] ?? 0)
  })

  const columnOf = new Map<string, number>()
  const boxes = new Map<string, TreeBox>()
  columns.forEach((house, index) => {
    for (const member of house.members) {
      columnOf.set(member.code, index)
      boxes.set(member.code, {
        x: columnX[index],
        y: rowTop[rowOf.get(member.code)!],
        w: MEDALLION_W,
        h: heightOf(member.code),
      })
    }
  })
  const lineY = (code: string) => boxes.get(code)!.y + MEDALLION_ANCHOR_Y
  const laneX = (gap: number, lane: number) =>
    columnX[gap] +
    MEDALLION_W +
    (gapWidth[gap] - TREE_LANE * (lanes[gap] - 1)) / 2 +
    TREE_LANE * lane

  const marriages: PlacedMarriage[] = []
  for (const { link, aligned, left, right } of layout.marriages) {
    const from = columnOf.get(left)!
    const to = columnOf.get(right)!
    if (from === to) continue
    const x1 = columnX[from] + MEDALLION_W
    const x2 = columnX[to]
    const y1 = lineY(left)
    const y2 = lineY(right)
    const bend = laneX(to - 1, layout.bendLane.get(link.key) ?? 0)
    marriages.push({
      link,
      aligned,
      d: aligned ? `M${x1},${y1} L${x2},${y2}` : `M${x1},${y1} H${bend} V${y2} H${x2}`,
    })
  }

  const claims: PlacedClaim[] = layout.stems.map((stem) => {
    const heir = boxes.get(stem.heir)!
    const heirColumn = columnOf.get(stem.heir)!
    // The stem reaches the heir from the side of the gap it runs in.
    const heirX = stem.gap === heirColumn ? heir.x + heir.w : heir.x
    const stemX = laneX(stem.gap, stem.lane)
    const heirY = lineY(stem.heir)
    let start = `M${stemX},${lineY(stem.anchor)}`
    if (stem.fromTarget) {
      const target = boxes.get(stem.target)!
      start = `M${columnOf.get(stem.target)! < heirColumn ? target.x + target.w : target.x},${lineY(stem.target)} H${stemX}`
    }
    return { claim: stem.claim, d: `${start} V${heirY} H${heirX}` }
  })
  return { width: x, height: top, columnX, boxes, marriages, claims }
}
