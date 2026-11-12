import { shortCode } from '@/lib/card-hand'
import { isSettlement } from '@/lib/supply'
import { cleanOrderLine } from '@/lib/winter-draft'
import type { CardKind, NobleCardKind, PlayerId, Region, StateData, Territory, WinterAids } from '@/types'

/** A code the order editor can offer, with the name that explains it. */
export interface CodeSuggestion {
  code: string
  name: string
  kind: 'territory' | 'noble' | 'card'
}

/** The word being typed: its bounds in the text and its letters. */
interface TypedWord {
  start: number
  end: number
  text: string
}

/** Looks up the codes for a word typed after `textBefore`, the text of the zone before it. */
export type CodeSuggester = (word: string, textBefore: string) => CodeSuggestion[]

const LETTER = /\p{L}/u

/**
 * The word the cursor stands at the end of, possibly empty, unless the cursor
 * is inside a word or a comment. The editor decides how long it must be.
 */
export function wordAtCursor(value: string, cursor: number): TypedWord | null {
  if (cursor < value.length && LETTER.test(value[cursor])) return null
  let start = cursor
  while (start > 0 && LETTER.test(value[start - 1])) start--
  const lineStart = value.lastIndexOf('\n', start - 1) + 1
  if (value.slice(lineStart, start).includes('#')) return null
  return { start, end: cursor, text: value.slice(start, cursor) }
}

function fold(text: string): string {
  return text.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase()
}

/**
 * What an order expects at one of its places, as far as suggestions go:
 * territories by who holds them or what stands there, nobles by owner, or the
 * cards of the hand. `adjacent` is a neighbour of the territory just before.
 * The title orders read their places in the winter aids the server sends.
 */
export type SlotKind =
  | 'territory'
  | 'own'
  | 'ownSettlement'
  | 'foreignSettlement'
  | 'ownArmy'
  | 'foreignArmy'
  | 'adjacent'
  | 'region'
  | 'revoltTarget'
  | 'noble'
  | 'ownNoble'
  | 'otherNoble'
  | 'nobleCard'
  | 'dignityCard'
  | 'handCard'
  | 'specialCard'
  | 'calamity'
  | 'excommunicable'
  | 'liftable'
  | 'dissolvable'
  | 'buyableCardinal'
  | 'inquirer'
  | 'trialJudge'
  | 'witch'
  | 'royalIssuer'

/** A card of the hand, with its code and its name in the player's language. */
interface HandCard {
  code: string
  name: string
}

/** What the suggestions are drawn from. */
interface SuggestionSources {
  territories: readonly Territory[]
  state: Pick<StateData, 'territories' | 'nobles' | 'revoltTargets'>
  player: PlayerId
  regions?: readonly Region[]
  nobleCards?: ReadonlyArray<HandCard & { kind: NobleCardKind }>
  specialCards?: readonly HandCard[]
  /** Calamities a Witch can call (`S R NNN CAL`). */
  calamities?: readonly HandCard[]
  aids?: WinterAids
}

interface IndexedCode extends CodeSuggestion {
  folded: string
  /** Owned by the player: an army or a controlled territory, a noble, a card. */
  own: boolean
}

/** Codes the editors can offer, and the lists each kind of place draws from. */
interface CodeIndex {
  codes: IndexedCode[]
  byCode: Map<string, IndexedCode>
  neighbours: Map<string, readonly string[]>
  /** `army`: every territory holding an army, the player's first. */
  slots: Record<Exclude<SlotKind, 'territory' | 'adjacent'> | 'army', readonly string[]>
}

export function buildCodeIndex({
  territories,
  state,
  player,
  regions = [],
  nobleCards = [],
  specialCards = [],
  calamities = [],
  aids,
}: SuggestionSources): CodeIndex {
  const villages = new Set(territories.filter((territory) => territory.village).map((t) => t.id))
  const settled = state.territories.filter(
    (territory) => villages.has(territory.id) || isSettlement(territory),
  )
  const owned = state.territories.filter(
    (territory) => territory.owner === player || territory.army?.owner === player,
  )
  const ownArmies = state.territories.filter((territory) => territory.army?.owner === player)
  const foreignArmies = state.territories.filter(
    (territory) => territory.army && territory.army.owner !== player,
  )
  const ownNobles = state.nobles.filter((noble) => noble.owner === player)
  const otherNobles = state.nobles.filter((noble) => noble.owner !== player)
  const ids = (list: ReadonlyArray<{ id: string }>) => list.map((item) => item.id)
  const codesOf = (list: ReadonlyArray<{ code: string }>) => list.map((item) => item.code)
  const ownIds = ids(owned)
  const ownSet = new Set(ownIds)

  const entry = (code: string, name: string, kind: CodeSuggestion['kind'], own: boolean) => ({
    code,
    name,
    kind,
    folded: fold(name),
    own,
  })
  const codes: IndexedCode[] = [
    ...territories.map((t) => entry(t.id, t.name, 'territory', ownSet.has(t.id))),
    ...state.nobles.map((noble) => entry(noble.code, noble.name, 'noble', noble.owner === player)),
    ...[...nobleCards, ...specialCards, ...calamities].map((card) =>
      entry(card.code, card.name, 'card', true),
    ),
  ]
  return {
    codes,
    // The first entry of a code wins: a territory before a card of the same code.
    byCode: new Map([...codes].reverse().map((code) => [code.code, code])),
    neighbours: new Map(territories.map((territory) => [territory.id, territory.adjacencies])),
    slots: {
      army: ids([...ownArmies, ...foreignArmies]),
      own: ownIds,
      ownSettlement: ids(settled.filter((territory) => territory.owner === player)),
      foreignSettlement: ids(
        settled.filter((territory) => territory.owner && territory.owner !== player),
      ),
      ownArmy: ids(ownArmies),
      foreignArmy: ids(foreignArmies),
      region: regions.map((region) => region.seed),
      revoltTarget: state.revoltTargets ?? [],
      noble: codesOf([...ownNobles, ...otherNobles]),
      ownNoble: codesOf(ownNobles),
      otherNoble: codesOf(otherNobles),
      nobleCard: codesOf(nobleCards.filter((card) => card.kind === 'noble')),
      dignityCard: codesOf(nobleCards.filter((card) => card.kind === 'dignity')),
      handCard: codesOf([...nobleCards, ...specialCards]),
      specialCard: codesOf(specialCards),
      calamity: codesOf(calamities),
      excommunicable: codesOf(aids?.excommunicable ?? []),
      liftable: aids?.liftable ?? [],
      dissolvable: (aids?.dissolvable ?? []).map((marriage) => marriage.noble),
      buyableCardinal: aids?.buyableCardinals ?? [],
      inquirer: aids?.inquirers ?? [],
      trialJudge: aids?.trialJudges ?? [],
      witch: (aids?.rituals ?? []).map((ritual) => ritual.noble),
      royalIssuer: aids?.royal ? [aids.royal.issuer] : [],
    },
  }
}

/** Words of an order line, without its comment, brackets and hyphens. */
function lineWords(line: string): string[] {
  return cleanOrderLine(line).replace(/[()-]/g, ' ').split(/\s+/).filter(Boolean)
}

const STAYING_VERBS = new Set(['H', 'P'])
const MOVING_VERBS = new Set(['A', 'J'])

/** Where the army of a chain stands once `lines` are carried out, as far as their words tell. */
function chainPosition(lines: string[]): string | undefined {
  let position: string | undefined
  for (const line of lines) {
    const words = lineWords(line)
    // A lone word is the header of the noble, not an order.
    if (words.length < 2) continue
    if (STAYING_VERBS.has(words[0])) position = words[1]
    // After a dispersal the chain follows the army of its last destination (engine rule, specs/gdd.md).
    else if (words[1] === 'D' && words.length > 2) position = words[words.length - 1].split('*')[0]
    else position = MOVING_VERBS.has(words[1]) && words[2] ? words[2] : words[0]
  }
  return position
}

/**
 * Territories a chain order expects at the typed word, likeliest first, from
 * the words before it: the army's position at the head of a line (on the first
 * order, every army), a neighbour after a move, an attack or a support, the
 * position or a neighbour in a dispersion. Undefined where the words say
 * nothing (a verb, a number, a transfer target): any code may come.
 */
function chainSlotCodes(index: CodeIndex, textBefore: string): readonly string[] | undefined {
  const lines = textBefore.split('\n')
  const current = lineWords(lines.pop() ?? '')
  const position = chainPosition(lines)
  const around = (territory: string) => index.neighbours.get(territory) ?? []
  if (current.length === 0 || (current.length === 1 && STAYING_VERBS.has(current[0]))) {
    return position ? [position, ...around(position)] : index.slots.army
  }
  const [origin, verb] = current
  if (current.length === 2 && (MOVING_VERBS.has(verb) || verb === 'S')) return around(origin)
  if (verb === 'D') return [origin, ...around(origin)]
  // `XXX S YYY - ZZZ`: the attack of YYY goes to one of its neighbours.
  if (verb === 'S' && current.length === 3 && /-\s*$/.test(textBefore)) return around(current[2])
  return undefined
}

/**
 * Places of the winter orders after their verb (specs/gdd.md § Phase d'hiver,
 * titres.md, politique.md). The preview judges the orders: these tables only
 * steer the suggestions.
 */
export const WINTER_SLOTS: Record<string, SlotKind[]> = {
  'R N': ['nobleCard', 'ownSettlement'],
  'C N': ['ownNoble', 'otherNoble'],
  'D N': ['noble', 'dignityCard', 'region'],
  'S R': ['witch', 'calamity'],
  'D C': ['handCard'],
  'A N': ['ownArmy'],
  'R T': ['own'],
  'C M': ['own'],
  'C C': ['own'],
  'C D': ['own'],
  'E C': ['ownSettlement'],
  'O N': ['otherNoble'],
  'P N': ['otherNoble'],
  'H N': ['ownNoble', 'foreignArmy'],
  G: ['ownSettlement', 'foreignSettlement'],
  'T F': ['ownNoble', 'ownSettlement', 'own'],
  'T A': ['ownNoble', 'ownSettlement'],
  'T X': ['ownNoble', 'ownSettlement', 'adjacent'],
  'T R': ['ownNoble', 'ownSettlement'],
  'T C': ['ownNoble'],
  'T D': ['ownNoble', 'ownSettlement'],
  'K R': ['ownNoble'],
  'V R': ['noble'],
  'V C': ['ownNoble'],
  'W X': ['royalIssuer', 'territory', 'adjacent'],
  'W R': ['royalIssuer', 'territory'],
  'W F': ['royalIssuer', 'territory'],
  'W G': ['royalIssuer', 'territory', 'noble'],
  'M N': ['ownNoble', 'otherNoble'],
  'X E': ['excommunicable'],
  'X L': ['liftable'],
  'X D': ['dissolvable'],
  'N C': ['buyableCardinal'],
  Q: ['inquirer', 'noble'],
  J: ['trialJudge', 'noble'],
}

/** `P` and a card code, as the special card orders write it. */
const play = (kind: CardKind) => `P ${shortCode(kind)}`

/** Places of the special card orders after `P` and the card (specs/ordres-speciaux.md). */
export const SPECIAL_SLOTS: Record<string, SlotKind[]> = {
  P: ['specialCard'],
  'D C': ['specialCard'],
  [play('fair_weather')]: ['region'],
  [play('abundant_harvest')]: ['region'],
  [play('revolt')]: ['revoltTarget'],
  [play('seigneurial_tax')]: ['ownNoble', 'territory'],
  [play('tunnel')]: ['noble'],
  [play('assassination')]: ['ownNoble', 'otherNoble'],
  [play('justice')]: ['ownNoble'],
  [play('ambush')]: ['ownArmy', 'adjacent'],
  // Revolt appeasement and tithe: no card of their own.
  'P AG': ['ownNoble', 'region'],
  'P AP': ['ownNoble', 'region'],
  'P DI': ['ownNoble', 'region'],
}

/** Orders whose last place repeats: the territories of a new fief. */
const REPEATING = new Set(['T F'])

/**
 * Codes a sheet order expects at the typed word, from its verb and the words
 * already written after it: the longest verb of `table` the line starts with
 * gives the places, the number of words after it the place being typed.
 */
function sheetSlotCodes(
  index: CodeIndex,
  table: Record<string, SlotKind[]>,
  textBefore: string,
): readonly string[] | undefined {
  const words = lineWords(textBefore.slice(textBefore.lastIndexOf('\n') + 1))
  const verb = [words.slice(0, 2).join(' '), words[0]].find(
    (candidate) => candidate !== undefined && candidate in table,
  )
  if (!verb) return undefined
  const places = table[verb]
  const args = words.slice(verb.split(' ').length)
  const place = REPEATING.has(verb) ? places[Math.min(args.length, places.length - 1)] : places[args.length]
  if (!place || place === 'territory') return undefined
  if (place === 'adjacent') {
    const before = args.at(-1)
    return before ? index.neighbours.get(before) : undefined
  }
  return index.slots[place]
}

/**
 * Codes that complete the typed word, by code or by name: the codes `slot`
 * expects first, in its order, then the player's own codes, then the others.
 * An empty word lists the slot alone. A word that already is a whole code of
 * the slot, or a whole three-letter code, needs nothing. The suggestions only
 * help typing; the preview judges the order.
 */
export function suggestCodes(
  index: CodeIndex,
  word: string,
  slot: readonly string[] | undefined,
  limit = 12,
): CodeSuggestion[] {
  const upper = word.toUpperCase()
  const slotRank = new Map((slot ?? []).map((code, position) => [code, position]))
  if (slotRank.has(upper) || (upper.length === 3 && index.byCode.has(upper))) return []
  const pick = ({ code, name, kind }: IndexedCode): CodeSuggestion => ({ code, name, kind })
  if (word === '') {
    // The slot is already in rank order.
    return [...slotRank.keys()]
      .flatMap((code) => index.byCode.get(code) ?? [])
      .slice(0, limit)
      .map(pick)
  }
  const folded = fold(word)
  // Rank once per code: the slot in its order, then own codes, then the others.
  const others = slotRank.size
  const seen = new Set<string>()
  return index.codes
    .filter((entry) => entry.code.startsWith(upper) || entry.folded.startsWith(folded))
    .filter((entry) => !seen.has(entry.code) && Boolean(seen.add(entry.code)))
    .map((entry) => ({ entry, rank: slotRank.get(entry.code) ?? others + (entry.own ? 0 : 1) }))
    .sort((a, b) => a.rank - b.rank || (a.entry.code < b.entry.code ? -1 : 1))
    .slice(0, limit)
    .map(({ entry }) => pick(entry))
}

/** Suggester of a chain: the codes its orders expect at the cursor come first. */
export function chainSuggester(index: CodeIndex): CodeSuggester {
  return (word, textBefore) => suggestCodes(index, word, chainSlotCodes(index, textBefore))
}

/** Suggester of a sheet of orders (winter or special cards) read through `table`. */
export function sheetSuggester(index: CodeIndex, table: Record<string, SlotKind[]>): CodeSuggester {
  return (word, textBefore) => suggestCodes(index, word, sheetSlotCodes(index, table, textBefore))
}
