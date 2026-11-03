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

  const noblesByOwner = new Map<string, Noble[]>()
  for (const noble of nobles) {
    const owned = noblesByOwner.get(noble.owner) ?? []
    owned.push(noble)
    noblesByOwner.set(noble.owner, owned)
  }
  const deceasedByOwner = new Map<string, DeceasedNoble[]>()
  for (const noble of deceased) {
    const owned = deceasedByOwner.get(noble.owner) ?? []
    owned.push(noble)
    deceasedByOwner.set(noble.owner, owned)
  }

  return players.map((player) => {
    const own = noblesByOwner.get(player.id) ?? []
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
    const dead = [...(deceasedByOwner.get(player.id) ?? [])].sort((a, b) => a.turn - b.turn)
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
  const seen = new Map<string, number>()
  return marriages.map((marriage) => {
    // A pair married again keeps the first key; the next ones are numbered.
    const pair = `${marriage.nobleA}-${marriage.nobleB}`
    const count = (seen.get(pair) ?? 0) + 1
    seen.set(pair, count)
    const key = count === 1 ? pair : `${pair}#${count}`
    const active = marriage.active ?? true
    const activeHeadFor = marriage.activeHeadFor ?? []
    const isActiveHead = activeHeadFor.length > 0
    let kind: EdgeKind = 'ended'
    if (active && marriage.category) kind = isActiveHead ? 'head' : 'secondary'
    return {
      key,
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
  /** Turn the claim was staked, to order the claims that compete for a row. */
  turn?: number
}

/** A marriage between two nobles of the tree, left card first. */
export interface TreeMarriage {
  link: LineageLink
  /** Spouses stand side by side; otherwise the line arches over the cards between. */
  aligned: boolean
  left: string
  right: string
}

/** The stem of a claim: it leaves its couple and runs down to the heir. */
export interface ClaimStem {
  /** Index of the claim in the list given to `layoutTree`. */
  claim: number
  heir: string
  target: string
  /**
   * The stem leaves the knot of this marriage (left and right spouse); without
   * it, it leaves the card of the target: no marriage side by side joins the
   * target and the spouse of the claim.
   */
  couple?: [string, string]
  /**
   * The heir hangs on a row below its couple. False for a claim that cannot
   * (the heir is the couple itself, or its couple descends from the heir): the
   * stem is then drawn from the couple to the heir wherever they stand.
   */
  hangs: boolean
  /** Rows the stem goes from and to. */
  rowFrom: number
  rowTo: number
  /** Horizontal position of the stem on each row from `rowFrom` to `rowTo`. */
  xs: number[]
  /** Lane of the stem's bar in the channel under each row it crosses (one per hop of `xs`). */
  lanes: number[]
  /**
   * A stem that does not hang and would run through cards (the heir is above
   * the couple, or more than one row below) goes round them: down to a bar under
   * the couple's row, along it to a free vertical line outside the cards at `x`,
   * along that line, then through a bar in the channel next to the heir's row.
   * `lanes` are those of the two bars; `xs` and `lanes` above only hold the ends.
   */
  detour?: {
    x: number
    /** Channel (the one under this row) of the second bar. */
    entryBand: number
    lanes: [number, number]
  }
}

export interface TreeLayout {
  rows: number
  /** Generation row of every noble. */
  rowOf: Map<string, number>
  /** Left edge of every noble's card, in pixels. */
  x: Map<string, number>
  width: number
  /** The nobles of each row, left to right. */
  order: string[][]
  marriages: TreeMarriage[]
  stems: ClaimStem[]
  /** Lanes taken in the channel under each row. */
  lanes: number[]
}

/** Width of a noble's cartouche, in pixels. */
export const MEDALLION_W = 196
/** Height of a cartouche with no text line under its name. */
export const MEDALLION_BASE_H = 56
/** Height of one text line under the name. */
export const MEDALLION_LINE_H = 16
/** Characters of a text line that fit in a cartouche before it wraps. */
export const MEDALLION_LINE_CHARS = 22
/** Vertical position of the marriage lines: the middle of the cartouche's heading. */
export const MEDALLION_ANCHOR_Y = 28
/** Room above the first row, for the arches of marriages that skip a card. */
export const TREE_PAD_Y = 30
export const TREE_LANE = 8
/** Gap between two spouses: it holds the knot of their marriage. */
export const COUPLE_GAP = 88
/** Width the tree aims for: the room a full-screen dialog leaves at 1440 px. */
export const TREE_BUDGET = 1280
const SIBLING_GAP = 20
const GROUP_GAP = 44
/** Offset between the lines of a pair married more than once. */
const MARRIAGE_AGAIN_STEP = 13
/** Width and spacing of the empty slot that keeps a long stem clear of the cards. */
const SLOT_W = 12
const SLOT_GAP = 16

const cmp = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0)
const pairKey = (a: string, b: string) => (a < b ? `${a}|${b}` : `${b}|${a}`)

/**
 * A couple (or a lone noble) with the siblings dispatched beside it, laid out
 * side by side; or an empty slot a long stem passes through.
 */
interface Unit {
  id: string
  /** The couple as arranged, spouses side by side. */
  core: string[]
  /** Every noble shown, left to right: the core and its siblings. */
  codes: string[]
  /** Left edge of each noble relative to the unit. */
  offset: Map<string, number>
  width: number
  ord: number
  /** Row within its tree, then the row of the whole layout. */
  row: number
  x: number
  slot: boolean
}

/** A stem hop from a point of one unit to a point of a unit on the next row. */
interface Join {
  from: Unit
  fromOff: number
  to: Unit
  toOff: number
}

/** The units of a marriage tree, arranged on rows of their own. */
interface TreeBlock {
  units: Unit[]
  slots: Unit[]
  depth: number
  width: number
  x: number
}

/** Puts the siblings before and after the core of a unit and measures it. */
function dress(unit: Unit, before: string[], after: string[]) {
  unit.codes = [...before, ...unit.core, ...after]
  const core = new Set(unit.core)
  unit.offset = new Map()
  let at = 0
  unit.codes.forEach((code, index) => {
    if (index > 0) {
      const spouses = core.has(code) && core.has(unit.codes[index - 1])
      at += MEDALLION_W + (spouses ? COUPLE_GAP : SIBLING_GAP)
    }
    unit.offset.set(code, at)
  })
  unit.width = at + MEDALLION_W
}

/**
 * Orders the nobles of a marriage component so that spouses stand side by
 * side: a path (a noble with two spouses) is laid out end to end; anything
 * else is walked around its best-married noble, and the marriages left apart
 * are drawn arched.
 */
function arrangeCouple(
  codes: string[],
  neighbours: Map<string, Set<string>>,
  ordOf: Map<string, number>,
): string[] {
  if (codes.length < 2) return codes
  const byOrd = (a: string, b: string) => ordOf.get(a)! - ordOf.get(b)!
  const next = (code: string) => [...neighbours.get(code)!].sort(byOrd)
  const degree = (code: string) => neighbours.get(code)!.size
  const edges = codes.reduce((sum, code) => sum + degree(code), 0) / 2
  const seen = new Set<string>()
  const order: string[] = []
  const walk = (code: string) => {
    if (seen.has(code)) return
    seen.add(code)
    order.push(code)
    for (const other of next(code)) walk(other)
  }
  if (codes.every((code) => degree(code) <= 2) && edges === codes.length - 1) {
    walk(codes.find((code) => degree(code) === 1)!)
    return order
  }
  // A hub keeps two spouses beside it; the other marriages arch over the cards between.
  const hub = [...codes].sort((a, b) => degree(b) - degree(a) || byOrd(a, b))[0]
  const [first, ...others] = next(hub)
  seen.add(first)
  order.push(first)
  seen.add(hub)
  order.push(hub)
  for (const other of others) walk(other)
  for (const code of codes) walk(code)
  return order
}

/**
 * Lays the nobles out as a genealogical tree:
 *
 * 1. Marriages (ended ones too) join nobles into couples that stand side by
 *    side on one row; a noble with two spouses stands between them.
 * 2. A claim hangs its heir on the row below the couple that carries it (the
 *    target and the spouse of the claim). A claim that would make a couple
 *    descend from itself is left out of the rows (`hangs: false`). A couple
 *    gets the deepest row any of its heirs requires. A couple and the heirs
 *    hanging under it, generation after generation, form a marriage tree.
 * 3. A noble with no claim and no marriage is listed in a line of its house,
 *    in succession order. The nobles of a house who come before its first
 *    noble of a tree (in the living succession) stand in a line above all the
 *    trees; those who come after its last one stand in a line below them. A
 *    noble between two of them stands in the tree, as a sibling beside the
 *    closest earlier one: on the outer side of a couple. The deceased of a
 *    house with a tree stand below. Lines of several houses share a row when
 *    they fit, and a line wider than `TREE_BUDGET` wraps.
 * 4. Each tree is ordered by barycentre sweeps (down by the parents'
 *    position, up by the heirs'), keeping the order with the fewest stem
 *    crossings; ties go to the house order, then the succession order.
 *    Heirs centre under their couple's knot and couples over their heirs, no
 *    two cards overlapping. A stem to an heir more than one row below crosses
 *    the rows between through an empty slot.
 * 5. Trees are packed on shelves no wider than `TREE_BUDGET` (a wider tree has
 *    a shelf of its own), the lines above first and the lines below last;
 *    every shelf and line is centred.
 */
export function layoutTree(
  houses: House[],
  links: LineageLink[],
  focus: PlayerId | null,
  claims: ClaimRef[] = [],
): TreeLayout {
  const sortedHouses = [...houses].sort((a, b) => cmp(a.player.id, b.player.id))
  const houseOf = new Map<string, House>()
  for (const house of sortedHouses)
    for (const member of house.members) houseOf.set(member.code, house)
  const columns = cachedColumns(sortedHouses, links, focus, claims, houseOf)

  const ordOf = new Map<string, number>()
  const houseIndex = new Map<string, number>()
  columns.forEach((house, h) =>
    house.members.forEach((member, m) => {
      ordOf.set(member.code, h * 10000 + m)
      houseIndex.set(member.code, h)
    }),
  )
  const allCodes = [...ordOf.keys()].sort((a, b) => ordOf.get(a)! - ordOf.get(b)!)

  // Couples: the components of the marriage graph.
  const usable = links
    .filter((link) => ordOf.has(link.a) && ordOf.has(link.b) && link.a !== link.b)
    .sort((x, y) => cmp(x.key, y.key))
  const parent = new Map(allCodes.map((code) => [code, code]))
  const find = (code: string): string => {
    let root = code
    while (parent.get(root) !== root) root = parent.get(root)!
    return root
  }
  const neighbours = new Map<string, Set<string>>(allCodes.map((code) => [code, new Set()]))
  for (const link of usable) {
    neighbours.get(link.a)!.add(link.b)
    neighbours.get(link.b)!.add(link.a)
    const ra = find(link.a)
    const rb = find(link.b)
    if (ra === rb) continue
    if (ordOf.get(ra)! < ordOf.get(rb)!) parent.set(rb, ra)
    else parent.set(ra, rb)
  }
  const members = new Map<string, string[]>()
  for (const code of allCodes) {
    const root = find(code)
    members.set(root, [...(members.get(root) ?? []), code])
  }
  const unitOf = new Map<string, Unit>()
  const units: Unit[] = []
  for (const codes of members.values()) {
    const core = arrangeCouple(codes, neighbours, ordOf)
    const unit: Unit = {
      id: core[0],
      core,
      codes: core,
      offset: new Map(),
      width: 0,
      ord: ordOf.get(core[0])!,
      row: 0,
      x: 0,
      slot: false,
    }
    dress(unit, [], [])
    for (const code of core) unitOf.set(code, unit)
    units.push(unit)
  }
  const offsetOf = (code: string) => unitOf.get(code)!.offset.get(code)!
  const marriagesOf = (): TreeMarriage[] =>
    usable.map((link) => {
      const [left, right] =
        offsetOf(link.a) <= offsetOf(link.b) ? [link.a, link.b] : [link.b, link.a]
      return {
        link,
        left,
        right,
        aligned: offsetOf(right) - offsetOf(left) === MEDALLION_W + COUPLE_GAP,
      }
    })
  const sideBySide = new Map<string, TreeMarriage>()
  for (const marriage of marriagesOf())
    if (marriage.aligned && !sideBySide.has(pairKey(marriage.left, marriage.right)))
      sideBySide.set(pairKey(marriage.left, marriage.right), marriage)

  // Claims, in an order that does not depend on the input order.
  interface Hang {
    index: number
    claim: ClaimRef
    heir: Unit
    parent: Unit
    couple?: TreeMarriage
    hangs: boolean
    via: Unit[]
  }
  const hangs: Hang[] = []
  claims.forEach((claim, index) => {
    if (!ordOf.has(claim.heir) || !ordOf.has(claim.target) || claim.heir === claim.target)
      return
    hangs.push({
      index,
      claim,
      heir: unitOf.get(claim.heir)!,
      parent: unitOf.get(claim.target)!,
      couple: sideBySide.get(pairKey(claim.target, claim.spouse)),
      hangs: true,
      via: [],
    })
  })
  const canonical = [...hangs].sort(
    (x, y) =>
      Number(Boolean(x.claim.ended)) - Number(Boolean(y.claim.ended)) ||
      (x.claim.turn ?? 0) - (y.claim.turn ?? 0) ||
      ordOf.get(x.claim.heir)! - ordOf.get(y.claim.heir)! ||
      ordOf.get(x.claim.target)! - ordOf.get(y.claim.target)! ||
      cmp(x.claim.spouse, y.claim.spouse) ||
      x.index - y.index,
  )
  const heirsOf = new Map<Unit, Unit[]>()
  const parentsOf = new Map<Unit, Unit[]>()
  const reaches = (from: Unit, to: Unit): boolean => {
    const seen = new Set<Unit>()
    const stack = [from]
    while (stack.length > 0) {
      const unit = stack.pop()!
      if (unit === to) return true
      if (seen.has(unit)) continue
      seen.add(unit)
      stack.push(...(heirsOf.get(unit) ?? []))
    }
    return false
  }
  for (const hang of canonical) {
    if (hang.heir === hang.parent || reaches(hang.heir, hang.parent)) {
      hang.hangs = false
      continue
    }
    heirsOf.set(hang.parent, [...(heirsOf.get(hang.parent) ?? []), hang.heir])
    parentsOf.set(hang.heir, [...(parentsOf.get(hang.heir) ?? []), hang.parent])
  }
  const depth = new Map<Unit, number>()
  const rowFor = (unit: Unit): number => {
    const known = depth.get(unit)
    if (known !== undefined) return known
    const row = Math.max(0, ...(parentsOf.get(unit) ?? []).map((up) => rowFor(up) + 1))
    depth.set(unit, row)
    return row
  }
  for (const unit of units) unit.row = rowFor(unit)

  // Marriage trees: the couples joined by the claims that hang.
  const inTree = (unit: Unit) =>
    unit.core.length > 1 || heirsOf.has(unit) || parentsOf.has(unit)
  const treeParent = new Map(units.map((unit) => [unit, unit]))
  const treeRoot = (unit: Unit): Unit => {
    let root = unit
    while (treeParent.get(root) !== root) root = treeParent.get(root)!
    return root
  }
  for (const [up, down] of heirsOf)
    for (const unit of down) {
      const a = treeRoot(up)
      const b = treeRoot(unit)
      if (a === b) continue
      if (a.ord < b.ord) treeParent.set(b, a)
      else treeParent.set(a, b)
    }
  const blocks = new Map<Unit, TreeBlock>()
  const blockOf = new Map<Unit, TreeBlock>()
  for (const unit of units) {
    if (!inTree(unit)) continue
    const root = treeRoot(unit)
    const block = blocks.get(root) ?? { units: [], slots: [], depth: 0, width: 0, x: 0 }
    block.units.push(unit)
    block.depth = Math.max(block.depth, unit.row + 1)
    blocks.set(root, block)
    blockOf.set(unit, block)
  }
  const trees = [...blocks.values()]

  // Free nobles: above the trees of their house, below them, or beside a sibling in a tree.
  const above: string[][] = []
  const below: string[][] = []
  const attached = new Map<string, string[]>()
  for (const house of columns) {
    const living = house.members.filter((member) => !member.dead)
    const anchors = living
      .map((member, index) => ({ code: member.code, index }))
      .filter(({ code }) => inTree(unitOf.get(code)!))
    const lineAbove: string[] = []
    const lineBelow: string[] = []
    let nextLiving = 0
    for (const member of house.members) {
      // Position among the living; the dead have none.
      const index = member.dead ? -1 : nextLiving++
      if (inTree(unitOf.get(member.code)!)) continue
      const earlier = anchors.filter((anchor) => anchor.index < index)
      if (anchors.length === 0) lineAbove.push(member.code)
      else if (member.dead || index < 0) lineBelow.push(member.code)
      else if (earlier.length === 0) lineAbove.push(member.code)
      else if (earlier.length === anchors.length) lineBelow.push(member.code)
      else {
        const anchor = earlier[earlier.length - 1].code
        attached.set(anchor, [...(attached.get(anchor) ?? []), member.code])
      }
    }
    if (lineAbove.length > 0) above.push(lineAbove)
    if (lineBelow.length > 0) below.push(lineBelow)
  }
  const dressings = new Map<Unit, { before: string[]; after: string[] }>()
  for (const [anchor, siblings] of [...attached].sort(
    (p, q) => ordOf.get(p[0])! - ordOf.get(q[0])!,
  )) {
    const unit = unitOf.get(anchor)!
    const dressing = dressings.get(unit) ?? { before: [], after: [] }
    // Beside the spouse's side the couple is not opened: siblings stand on the outer side.
    if (unit.core.length > 1 && unit.core[0] === anchor)
      dressing.before.push(...[...siblings].reverse())
    else dressing.after.push(...siblings)
    dressings.set(unit, dressing)
  }
  for (const [unit, { before, after }] of dressings) {
    dress(unit, before, after)
    for (const code of [...before, ...after]) unitOf.set(code, unit)
  }

  // Joins between neighbouring rows, with an empty slot on every row a stem skips.
  const slots: Unit[] = []
  const joins: Join[] = []
  for (const hang of canonical) {
    if (!hang.hangs) continue
    const { claim } = hang
    const block = blockOf.get(hang.parent)!
    let from = hang.parent
    let fromOff = hang.couple
      ? offsetOf(hang.couple.left) + MEDALLION_W + COUPLE_GAP / 2
      : offsetOf(claim.target) + MEDALLION_W / 2
    for (let row = hang.parent.row + 1; row < hang.heir.row; row++) {
      const slot: Unit = {
        id: `~${hang.index}-${row}`,
        core: [],
        codes: [],
        offset: new Map(),
        width: SLOT_W,
        ord: 1e9 + slots.length,
        row,
        x: 0,
        slot: true,
      }
      slots.push(slot)
      block.slots.push(slot)
      blockOf.set(slot, block)
      hang.via.push(slot)
      joins.push({ from, fromOff, to: slot, toOff: SLOT_W / 2 })
      from = slot
      fromOff = SLOT_W / 2
    }
    joins.push({
      from,
      fromOff,
      to: hang.heir,
      toOff: offsetOf(claim.heir) + MEDALLION_W / 2,
    })
  }
  const ins = new Map<Unit, Join[]>()
  const outs = new Map<Unit, Join[]>()
  const joinsOfTree = new Map<TreeBlock, Join[]>()
  for (const join of joins) {
    const block = blockOf.get(join.from)!
    const inTree = joinsOfTree.get(block) ?? []
    inTree.push(join)
    joinsOfTree.set(block, inTree)
    const into = ins.get(join.to) ?? []
    into.push(join)
    ins.set(join.to, into)
    const outOf = outs.get(join.from) ?? []
    outOf.push(join)
    outs.set(join.from, outOf)
  }

  const gapOf = (a: Unit, b: Unit) => {
    if (a.slot || b.slot) return SLOT_GAP
    return houseIndex.get(a.codes[a.codes.length - 1]) === houseIndex.get(b.codes[0])
      ? SIBLING_GAP
      : GROUP_GAP
  }
  /** Where a unit wants to stand to line up with its parents or its heirs. */
  const desired = (unit: Unit, byHeirs: boolean) => {
    const wanted = byHeirs
      ? (outs.get(unit) ?? []).map((j) => j.to.x + j.toOff - j.fromOff)
      : (ins.get(unit) ?? []).map((j) => j.from.x + j.fromOff - j.toOff)
    return wanted.length === 0
      ? undefined
      : wanted.reduce((sum, value) => sum + value, 0) / wanted.length
  }

  /** Orders and positions the rows of one tree, from x = 0. */
  const arrange = (tree: TreeBlock) => {
    const everyone = [...tree.units, ...tree.slots]
    const treeJoins = joinsOfTree.get(tree) ?? []
    let state: Unit[][] = Array.from({ length: tree.depth }, () => [])
    for (const unit of everyone) state[unit.row]?.push(unit)
    for (const row of state) row.sort((a, b) => a.ord - b.ord)
    const assign = (rows: Unit[][]) => {
      for (const row of rows) {
        let prev: Unit | null = null
        for (const unit of row) {
          unit.x = prev ? prev.x + prev.width + gapOf(prev, unit) : 0
          prev = unit
        }
      }
      const relax = (row: Unit[], byHeirs: boolean) => {
        let prev: Unit | null = null
        for (const unit of row) {
          const floor = prev ? prev.x + prev.width + gapOf(prev, unit) : 0
          unit.x = Math.max(floor, desired(unit, byHeirs) ?? unit.x)
          prev = unit
        }
      }
      for (let pass = 0; pass < 4; pass++) {
        for (let row = 1; row < rows.length; row++) relax(rows[row], false)
        for (let row = rows.length - 2; row >= 0; row--) relax(rows[row], true)
      }
      const lowest = Math.min(...everyone.map((unit) => unit.x))
      for (const unit of everyone) unit.x -= lowest
    }
    const crossings = () => {
      let total = 0
      for (let i = 0; i < treeJoins.length; i++)
        for (let j = i + 1; j < treeJoins.length; j++) {
          if (treeJoins[i].from.row !== treeJoins[j].from.row) continue
          const above =
            treeJoins[i].from.x + treeJoins[i].fromOff - treeJoins[j].from.x - treeJoins[j].fromOff
          const below =
            treeJoins[i].to.x + treeJoins[i].toOff - treeJoins[j].to.x - treeJoins[j].toOff
          if (above * below < 0) total++
        }
      return total
    }
    const resort = (rows: Unit[][], row: number, byHeirs: boolean) => {
      const keyed = rows[row].map((unit, index) => ({
        unit,
        index,
        key: Math.round(((desired(unit, byHeirs) ?? unit.x) + unit.width / 2) * 1000) / 1000,
      }))
      keyed.sort((p, q) => p.key - q.key || p.index - q.index)
      rows[row] = keyed.map((entry) => entry.unit)
    }
    const copy = (rows: Unit[][]) => rows.map((row) => [...row])
    const signature = (rows: Unit[][]) =>
      rows.map((row) => row.map((unit) => unit.id).join(',')).join('/')

    assign(state)
    let best = copy(state)
    let bestCrossings = crossings()
    for (let sweep = 0; sweep < 8 && bestCrossings > 0; sweep++) {
      const before = signature(state)
      for (let row = 1; row < state.length; row++) {
        resort(state, row, false)
        assign(state)
      }
      if (crossings() < bestCrossings) {
        bestCrossings = crossings()
        best = copy(state)
      }
      for (let row = state.length - 2; row >= 0; row--) {
        resort(state, row, true)
        assign(state)
      }
      if (crossings() < bestCrossings) {
        bestCrossings = crossings()
        best = copy(state)
      }
      if (signature(state) === before) break
    }
    state = best
    assign(state)
    tree.width = Math.max(...tree.units.map((unit) => unit.x + unit.width))
  }
  for (const tree of trees) arrange(tree)

  // Shelves of trees, between the lines above and the lines below.
  const budget = Math.max(TREE_BUDGET, ...trees.map((tree) => tree.width))
  const perLine = Math.max(1, Math.floor((budget + SIBLING_GAP) / (MEDALLION_W + SIBLING_GAP)))
  const lineWidth = (count: number) => count * MEDALLION_W + (count - 1) * SIBLING_GAP
  /** Lines of several houses share a row when they fit; a line wider than the budget wraps. */
  const packLines = (lines: string[][]) => {
    const rows: string[][][] = []
    let open = false
    let used = 0
    for (const line of lines) {
      if (line.length > perLine) {
        for (let at = 0; at < line.length; at += perLine)
          rows.push([line.slice(at, at + perLine)])
        open = false
        continue
      }
      const width = lineWidth(line.length)
      if (open && used + GROUP_GAP + width <= budget) {
        rows[rows.length - 1].push(line)
        used += GROUP_GAP + width
      } else {
        rows.push([line])
        used = width
        open = true
      }
    }
    return rows
  }
  const rowWidth = (row: string[][]) =>
    row.reduce((sum, line) => sum + lineWidth(line.length), 0) + GROUP_GAP * (row.length - 1)
  const shelves: TreeBlock[][] = []
  let shelfWidth = 0
  for (const tree of trees) {
    if (shelves.length > 0 && shelfWidth + GROUP_GAP + tree.width <= budget) {
      shelves[shelves.length - 1].push(tree)
      shelfWidth += GROUP_GAP + tree.width
    } else {
      shelves.push([tree])
      shelfWidth = tree.width
    }
  }
  const shelfWidths = shelves.map(
    (shelf) => shelf.reduce((sum, tree) => sum + tree.width, 0) + GROUP_GAP * (shelf.length - 1),
  )
  const topRows = packLines(above)
  const bottomRows = packLines(below)
  let width = Math.max(
    0,
    ...shelfWidths,
    ...topRows.map(rowWidth),
    ...bottomRows.map(rowWidth),
  )

  const x = new Map<string, number>()
  const rowOf = new Map<string, number>()
  let row = 0
  const placeLines = (lineRows: string[][][]) => {
    for (const lineRow of lineRows) {
      let left = Math.floor((width - rowWidth(lineRow)) / 2)
      for (const line of lineRow) {
        for (const code of line) {
          x.set(code, left)
          rowOf.set(code, row)
          left += MEDALLION_W + SIBLING_GAP
        }
        left += GROUP_GAP - SIBLING_GAP
      }
      row++
    }
  }
  placeLines(topRows)
  shelves.forEach((shelf, index) => {
    let left = Math.floor((width - shelfWidths[index]) / 2)
    for (const tree of shelf) {
      for (const unit of [...tree.units, ...tree.slots]) {
        unit.x += left
        unit.row += row
        unit.codes.forEach((code) => {
          x.set(code, unit.x + unit.offset.get(code)!)
          rowOf.set(code, unit.row)
        })
      }
      left += tree.width + GROUP_GAP
    }
    row += Math.max(...shelf.map((tree) => tree.depth))
  })
  placeLines(bottomRows)
  const rows = row

  // Claims that do not hang and would cross cards run in a margin outside them:
  // a vertical line per stem, on the side nearer to it, 8px apart.
  const rowOfCode = (code: string) => rowOf.get(code)!
  const centreOf = (code: string) => x.get(code)! + MEDALLION_W / 2
  const startOf = (hang: Hang) =>
    hang.couple
      ? x.get(hang.couple.left)! + MEDALLION_W + COUPLE_GAP / 2
      : centreOf(hang.claim.target)
  const detouring = [...hangs]
    .filter((hang) => {
      if (hang.hangs) return false
      const from = rowOfCode(hang.claim.target)
      const to = rowOfCode(hang.claim.heir)
      return to < from || to > from + 1
    })
    .sort((a, b) => a.index - b.index)
  const sideOf = new Map<Hang, 'left' | 'right'>()
  let leftCount = 0
  let rightCount = 0
  for (const hang of detouring) {
    const middle = (startOf(hang) + centreOf(hang.claim.heir)) / 2
    const side = middle < width / 2 ? 'left' : 'right'
    sideOf.set(hang, side)
    if (side === 'left') leftCount++
    else rightCount++
  }
  const GUTTER_STEP = 8
  const leftMargin = leftCount > 0 ? GUTTER_STEP * leftCount + GUTTER_STEP : 0
  const rightMargin = rightCount > 0 ? GUTTER_STEP * rightCount + GUTTER_STEP : 0
  if (leftMargin > 0) for (const [code, at] of x) x.set(code, at + leftMargin)
  const gutterX = new Map<Hang, number>()
  let leftSeen = 0
  let rightSeen = 0
  for (const hang of detouring) {
    gutterX.set(
      hang,
      sideOf.get(hang) === 'left'
        ? leftMargin - 12 - GUTTER_STEP * leftSeen++
        : leftMargin + width + 12 + GUTTER_STEP * rightSeen++,
    )
  }
  width += leftMargin + rightMargin

  const marriages = marriagesOf()
  const order: string[][] = Array.from({ length: rows }, () => [])
  for (const [code, at] of rowOf) order[at].push(code)
  for (const codes of order) codes.sort((a, b) => x.get(a)! - x.get(b)! || cmp(a, b))

  // Stems and the lanes of their bars.
  const stems: ClaimStem[] = [...hangs]
    .sort((a, b) => a.index - b.index)
    .map((hang) => {
      const { claim } = hang
      const couple = hang.couple && ([hang.couple.left, hang.couple.right] as [string, string])
      const startX = hang.couple
        ? x.get(hang.couple.left)! + MEDALLION_W + COUPLE_GAP / 2
        : x.get(claim.target)! + MEDALLION_W / 2
      const rowFrom = rowOf.get(claim.target)!
      const rowTo = rowOf.get(claim.heir)!
      return {
        claim: hang.index,
        heir: claim.heir,
        target: claim.target,
        couple,
        hangs: hang.hangs,
        rowFrom,
        rowTo,
        xs: [
          startX,
          ...(hang.hangs ? hang.via.map((slot) => slot.x + SLOT_W / 2) : []),
          x.get(claim.heir)! + MEDALLION_W / 2,
        ],
        lanes: [],
        detour: gutterX.has(hang)
          ? {
              x: gutterX.get(hang)!,
              entryBand: rowTo > rowFrom ? rowTo - 1 : rowTo,
              lanes: [0, 0],
            }
          : undefined,
      }
    })
  interface Bar {
    band: number
    key: string
    lo: number
    hi: number
    hops: Array<[ClaimStem, number]>
    /** Which bar of a detour this is, when it is one. */
    detour?: [ClaimStem, 0 | 1]
  }
  const bars = new Map<string, Bar>()
  for (const stem of stems) {
    const key = stem.couple ? pairKey(...stem.couple) : stem.target
    if (stem.detour) {
      const { x: gx, entryBand } = stem.detour
      const ends = stem.xs
      const own = `~${stem.claim}`
      for (const [slot, band, from, to] of [
        [0, stem.rowFrom, ends[0], gx],
        [1, entryBand, gx, ends[ends.length - 1]],
      ] as const) {
        bars.set(`${band}:${own}:${slot}`, {
          band,
          key: own,
          lo: Math.min(from, to),
          hi: Math.max(from, to),
          hops: [],
          detour: [stem, slot],
        })
      }
      continue
    }
    const hops = stem.xs.length - 1
    stem.lanes = new Array<number>(hops).fill(0)
    for (let hop = 0; hop < hops; hop++) {
      const band = stem.rowFrom + hop
      const id = `${band}:${key}`
      const lo = Math.min(stem.xs[hop], stem.xs[hop + 1])
      const hi = Math.max(stem.xs[hop], stem.xs[hop + 1])
      const bar = bars.get(id) ?? { band, key, lo, hi, hops: [] }
      bar.lo = Math.min(bar.lo, lo)
      bar.hi = Math.max(bar.hi, hi)
      bar.hops.push([stem, hop])
      bars.set(id, bar)
    }
  }
  const lanes = new Array<number>(Math.max(rows, 1)).fill(0)
  const ends = new Map<number, number[]>()
  const byPlace = [...bars.values()]
    .filter((bar) => bar.hi > bar.lo)
    .sort((p, q) => p.band - q.band || p.lo - q.lo || cmp(p.key, q.key))
  for (const bar of byPlace) {
    const taken = ends.get(bar.band) ?? []
    let lane = taken.findIndex((end) => end + 6 < bar.lo)
    if (lane < 0) lane = taken.length
    taken[lane] = bar.hi
    ends.set(bar.band, taken)
    lanes[bar.band] = Math.max(lanes[bar.band] ?? 0, lane + 1)
    for (const [stem, hop] of bar.hops) stem.lanes[hop] = lane
    if (bar.detour) bar.detour[0].detour!.lanes[bar.detour[1]] = lane
  }

  return { rows, rowOf, x, width, order, marriages, stems, lanes }
}

/**
 * Codes of the nobles shown when a house is focused: its own, their spouses
 * (ended marriages included) and the nobles tied to them by a claim, in either
 * direction. `null` without a focus: everyone is shown.
 */
export function visibleNobles(
  houses: House[],
  links: LineageLink[],
  claims: Array<Pick<ClaimRef, 'heir' | 'target' | 'spouse'>>,
  focus: PlayerId | null,
): Set<string> | null {
  if (!focus) return null
  const own = new Set(
    houses
      .filter((h) => h.player.id === focus)
      .flatMap((h) => h.members.map((m) => m.code)),
  )
  const shown = new Set(own)
  for (const link of links) {
    if (own.has(link.a)) shown.add(link.b)
    if (own.has(link.b)) shown.add(link.a)
  }
  for (const claim of claims) {
    if (own.has(claim.heir)) {
      shown.add(claim.target)
      shown.add(claim.spouse)
    } else if (own.has(claim.target) || own.has(claim.spouse)) {
      shown.add(claim.heir)
      shown.add(claim.target)
      shown.add(claim.spouse)
    }
  }
  // A claim that stays drawn stands on its couple: the spouse is shown too, and
  // so is whatever claim that spouse then completes.
  for (let grown = true; grown; ) {
    grown = false
    for (const claim of claims) {
      if (!shown.has(claim.heir) || !shown.has(claim.target)) continue
      if (shown.has(claim.spouse)) continue
      shown.add(claim.spouse)
      grown = true
    }
  }
  return shown
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

/**
 * `orderColumns`, remembered per inputs that decide it (houses and their
 * members' codes, links, claims, focus): a simulated action changes the nobles'
 * data, not the order of the columns, and the search is costly.
 */
const columnCache = new Map<string, PlayerId[]>()
const COLUMN_CACHE_SIZE = 16

function cachedColumns(
  houses: House[],
  links: LineageLink[],
  focus: PlayerId | null,
  claims: ClaimRef[],
  houseOf: Map<string, House>,
): House[] {
  const signature = JSON.stringify([
    focus,
    houses.map((house) => [house.player.id, house.members.map((m) => m.code)]),
    links.map((link) => [link.key, link.a, link.b, link.active, link.weight ?? 0]),
    claims.map((claim) => [claim.heir, claim.target]),
  ])
  const known = columnCache.get(signature)
  if (known) {
    const byId = new Map(houses.map((house) => [house.player.id, house]))
    return known.map((id) => byId.get(id)!)
  }
  const columns = orderColumns(houses, links, focus, claims, houseOf)
  columnCache.set(signature, columns.map((house) => house.player.id))
  if (columnCache.size > COLUMN_CACHE_SIZE)
    columnCache.delete(columnCache.keys().next().value!)
  return columns
}

const MAX_PERMUTED_HOUSES = 7

/** The column order with the fewest skipped columns, keeping `start` when equally good. */
function leastSkippingOrder(
  start: House[],
  pairs: Array<{ a: string; b: string; weight: number }>,
  houseOf: Map<string, House>,
): House[] {
  if (start.length < 3) return start
  // Weight of every pair of houses, summed once.
  const between = new Map<string, { a: PlayerId; b: PlayerId; weight: number }>()
  for (const { a, b, weight } of pairs) {
    const ha = houseOf.get(a)?.player.id
    const hb = houseOf.get(b)?.player.id
    if (ha === undefined || hb === undefined) continue
    const key = ha < hb ? `${ha}|${hb}` : `${hb}|${ha}`
    const entry = between.get(key) ?? { a: ha, b: hb, weight: 0 }
    entry.weight += weight
    between.set(key, entry)
  }
  const crossPairs = [...between.values()]
  const cost = (order: House[]) => {
    const column = new Map(order.map((house, index) => [house.player.id, index]))
    let total = 0
    for (const { a, b, weight } of crossPairs)
      total += weight * Math.max(0, Math.abs(column.get(a)! - column.get(b)!) - 1)
    return total
  }
  let best = start
  let bestCost = cost(start)
  if (start.length <= MAX_PERMUTED_HOUSES) {
    const startAt = new Map(start.map((house, index) => [house, index]))
    const displacement = (order: House[]) =>
      order.reduce((sum, house, index) => sum + Math.abs(index - startAt.get(house)!), 0)
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
  boxes: Map<string, TreeBox>
  marriages: PlacedMarriage[]
  claims: PlacedClaim[]
}

/** Width of a character in average characters: capitals are wide, thin letters narrow. */
function charUnits(char: string): number {
  if ('mwMW@%'.includes(char)) return 1.4
  if (char >= 'A' && char <= 'Z') return 1.2
  if ("iljtfrI.,;:'!|()· ".includes(char)) return 0.6
  return 1
}

/**
 * Lines taken by a text wrapped to `perLine` average characters: words fill a
 * line in turn (a hyphen lets a word break), a word longer than a line breaks
 * wherever it must. Character widths vary, so capitals count for more than thin
 * letters; the count errs on the high side.
 */
export function wrapLines(text: string, perLine = MEDALLION_LINE_CHARS): number {
  const widthOf = (piece: string) =>
    [...piece].reduce((sum, char) => sum + charUnits(char), 0)
  let lines = 1
  let used = 0
  for (const word of text.split(/\s+/).filter(Boolean)) {
    let gap = used > 0 ? 0.6 : 0
    for (const piece of word.match(/[^-]*-|[^-]+/g) ?? []) {
      const width = widthOf(piece)
      if (used + gap + width <= perLine) {
        used += gap + width
      } else if (width <= perLine) {
        lines++
        used = width
      } else {
        if (used > 0) lines++
        const extra = Math.ceil(width / perLine) - 1
        lines += extra
        used = width - extra * perLine
      }
      gap = 0
    }
  }
  return lines
}

/** Lines taken by the given texts once wrapped to `perLine` characters. */
export function countLines(texts: string[], perLine = MEDALLION_LINE_CHARS): number {
  return texts.reduce((sum, text) => sum + wrapLines(text, perLine), 0)
}

/**
 * Turns a layout into pixels. A row is as tall as its tallest cartouche, so
 * spouses on one row share the height of their marriage line; the height of a
 * cartouche comes from the number of text lines under its name
 * (`medallionLines`), never from the DOM. The channel under a row grows with
 * the lanes its stems' bars take.
 */
export function placeTree(
  layout: TreeLayout,
  medallionLines: (code: string) => number,
): TreePlacement {
  const { rows, rowOf } = layout
  const heightOf = (code: string) => {
    const extra = medallionLines(code)
    return MEDALLION_BASE_H + extra * MEDALLION_LINE_H + (extra > 0 ? 4 : 0)
  }

  const rowHeight = new Array<number>(rows).fill(0)
  for (const [code, row] of rowOf) rowHeight[row] = Math.max(rowHeight[row], heightOf(code))
  const rowTop: number[] = []
  const channel = (row: number) => Math.max(56, 36 + TREE_LANE * (layout.lanes[row] ?? 0))
  let top = TREE_PAD_Y
  for (let row = 0; row < rows; row++) {
    rowTop.push(top)
    top += rowHeight[row] + (row < rows - 1 ? channel(row) : 0)
  }
  const height = top + 8

  const boxes = new Map<string, TreeBox>()
  for (const [code, row] of rowOf)
    boxes.set(code, {
      x: layout.x.get(code)!,
      y: rowTop[row],
      w: MEDALLION_W,
      h: heightOf(code),
    })
  const lineY = (code: string) => boxes.get(code)!.y + MEDALLION_ANCHOR_Y
  const barY = (row: number, lane: number) =>
    rowTop[row] + rowHeight[row] + 16 + TREE_LANE * lane

  const arches = new Map<number, number>()
  // A pair married again draws its next marriages beside the first.
  const sameCouple = new Map<string, number>()
  const marriages: PlacedMarriage[] = layout.marriages.map(
    ({ link, aligned, left, right }) => {
      const a = boxes.get(left)!
      const b = boxes.get(right)!
      const again = sameCouple.get(pairKey(left, right)) ?? 0
      sameCouple.set(pairKey(left, right), again + 1)
      if (aligned) {
        const dy = again * MARRIAGE_AGAIN_STEP
        return {
          link,
          aligned,
          d: `M${a.x + a.w},${lineY(left) + dy} L${b.x},${lineY(right) + dy}`,
        }
      }
      const row = rowOf.get(left)!
      const taken = arches.get(row) ?? 0
      arches.set(row, taken + 1)
      const archY = a.y - 10 - 6 * taken
      const x1 = a.x + a.w / 2 + again * MARRIAGE_AGAIN_STEP
      const x2 = b.x + b.w / 2 + again * MARRIAGE_AGAIN_STEP
      return { link, aligned, d: `M${x1},${a.y} V${archY} H${x2} V${b.y}` }
    },
  )

  const claims: PlacedClaim[] = layout.stems.map((stem) => {
    const heir = boxes.get(stem.heir)!
    const target = boxes.get(stem.target)!
    const start = stem.couple ? lineY(stem.couple[0]) : target.y + target.h
    let d = `M${stem.xs[0]},${start}`
    if (stem.detour) {
      const { x: gx, entryBand, lanes } = stem.detour
      d += ` V${barY(stem.rowFrom, lanes[0])} H${gx}`
      d += ` V${barY(entryBand, lanes[1])} H${stem.xs[stem.xs.length - 1]}`
      d += ` V${stem.rowTo > stem.rowFrom ? heir.y : heir.y + heir.h}`
      return { claim: stem.claim, d }
    }
    for (let hop = 0; hop < stem.xs.length - 1; hop++) {
      if (stem.xs[hop] === stem.xs[hop + 1]) continue
      d += ` V${barY(stem.rowFrom + hop, stem.lanes[hop])} H${stem.xs[hop + 1]}`
    }
    // An heir on a row of the couple or above is reached from below.
    d += ` V${stem.rowTo > stem.rowFrom ? heir.y : heir.y + heir.h}`
    return { claim: stem.claim, d }
  })
  return { width: layout.width, height, boxes, marriages, claims }
}
