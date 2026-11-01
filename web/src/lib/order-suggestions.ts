import type { NobleCardKind, PlayerId, Region, StateData, Territory } from '@/types'

/** A code the order editor can offer, with the name that explains it. */
export interface CodeSuggestion {
  code: string
  name: string
  kind: 'territory' | 'noble' | 'card'
}

/** The word being typed: its bounds in the text and its letters. */
export interface TypedWord {
  start: number
  end: number
  text: string
}

/** Looks up the codes for a word typed after `textBefore`, the text of the zone before it. */
export type CodeSuggester = (word: string, textBefore: string) => CodeSuggestion[]

const LETTER = /\p{L}/u

/**
 * The word the cursor stands at the end of, when it is long enough to look a
 * code up and is not in a comment. Two letters by default, since a single
 * letter is a verb; an explicit request (Ctrl+Space) takes even an empty word.
 */
export function wordAtCursor(value: string, cursor: number, minLength = 2): TypedWord | null {
  if (cursor < value.length && LETTER.test(value[cursor])) return null
  let start = cursor
  while (start > 0 && LETTER.test(value[start - 1])) start--
  if (cursor - start < minLength) return null
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

/** A card of the hand, with its code and its name in the player's language. */
export interface HandCard {
  code: string
  name: string
}

/** What the suggestions are drawn from. */
export interface SuggestionSources {
  territories: readonly Territory[]
  state: Pick<StateData, 'territories' | 'nobles' | 'revoltTargets'>
  player: PlayerId
  regions?: readonly Region[]
  nobleCards?: ReadonlyArray<HandCard & { kind: NobleCardKind }>
  specialCards?: readonly HandCard[]
}

interface IndexedCode extends CodeSuggestion {
  folded: string
  /** Owned by the player: an army or a controlled territory, a noble, a card. */
  own: boolean
}

/** Codes the editors can offer, and the lists each kind of place draws from. */
export interface CodeIndex {
  codes: IndexedCode[]
  neighbours: Map<string, readonly string[]>
  /** Territories holding an army, the player's first. */
  armies: string[]
  slots: Record<Exclude<SlotKind, 'territory' | 'adjacent'>, string[]>
}

export function buildCodeIndex({
  territories,
  state,
  player,
  regions = [],
  nobleCards = [],
  specialCards = [],
}: SuggestionSources): CodeIndex {
  const owned = state.territories.filter(
    (territory) => territory.owner === player || territory.army?.owner === player,
  )
  const villages = new Set(territories.filter((territory) => territory.village).map((t) => t.id))
  const settled = state.territories
    .filter(
      (territory) =>
        villages.has(territory.id) ||
        territory.infrastructures.some((infra) => infra.type === 'castle' || infra.type === 'village'),
    )
  const withArmy = state.territories.filter((territory) => territory.army)
  const ownArmies = withArmy.filter((territory) => territory.army?.owner === player)
  const foreignArmies = withArmy.filter((territory) => territory.army?.owner !== player)
  const ids = (list: ReadonlyArray<{ id: string }>) => list.map((item) => item.id)
  const codesOf = (list: ReadonlyArray<{ code: string }>) => list.map((item) => item.code)
  const ownNobles = state.nobles.filter((noble) => noble.owner === player)
  const otherNobles = state.nobles.filter((noble) => noble.owner !== player)
  const ownIds = new Set(ids(owned))

  const card = (entry: HandCard): IndexedCode => ({
    code: entry.code,
    name: entry.name,
    kind: 'card',
    folded: fold(entry.name),
    own: true,
  })
  return {
    codes: [
      ...territories.map<IndexedCode>((territory) => ({
        code: territory.id,
        name: territory.name,
        kind: 'territory',
        folded: fold(territory.name),
        own: ownIds.has(territory.id),
      })),
      ...state.nobles.map<IndexedCode>((noble) => ({
        code: noble.code,
        name: noble.name,
        kind: 'noble',
        folded: fold(noble.name),
        own: noble.owner === player,
      })),
      ...nobleCards.map(card),
      ...specialCards.map(card),
    ],
    neighbours: new Map(territories.map((territory) => [territory.id, territory.adjacencies])),
    armies: ids([...ownArmies, ...foreignArmies]),
    slots: {
      own: ids(owned),
      ownSettlement: ids(settled.filter((territory) => territory.owner === player)),
      foreignSettlement: ids(
        settled.filter((territory) => territory.owner && territory.owner !== player),
      ),
      ownArmy: ids(ownArmies),
      foreignArmy: ids(foreignArmies),
      region: regions.map((region) => region.seed),
      revoltTarget: [...(state.revoltTargets ?? [])],
      noble: codesOf([...ownNobles, ...otherNobles]),
      ownNoble: codesOf(ownNobles),
      otherNoble: codesOf(otherNobles),
      nobleCard: codesOf(nobleCards.filter((entry) => entry.kind === 'noble')),
      dignityCard: codesOf(nobleCards.filter((entry) => entry.kind === 'dignity')),
      handCard: codesOf([...nobleCards, ...specialCards]),
      specialCard: codesOf(specialCards),
    },
  }
}

/** Words of an order line, without its comment, brackets and hyphens. */
function lineWords(line: string): string[] {
  return line
    .replace(/#.*$/, '')
    .replace(/[()-]/g, ' ')
    .trim()
    .toUpperCase()
    .split(/\s+/)
    .filter(Boolean)
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
export function chainSlotCodes(index: CodeIndex, textBefore: string): string[] | undefined {
  const lines = textBefore.split('\n')
  const current = lineWords(lines.pop() ?? '')
  const position = chainPosition(lines)
  const around = (territory: string) => index.neighbours.get(territory) ?? []
  if (current.length === 0 || (current.length === 1 && STAYING_VERBS.has(current[0]))) {
    return position ? [position, ...around(position)] : index.armies
  }
  const [origin, verb] = current
  if (current.length === 2 && (MOVING_VERBS.has(verb) || verb === 'S')) return [...around(origin)]
  if (verb === 'D') return [origin, ...around(origin)]
  // `XXX S YYY - ZZZ`: the attack of YYY goes to one of its neighbours.
  if (verb === 'S' && current.length === 3 && /-\s*$/.test(textBefore)) return [...around(current[2])]
  return undefined
}

/** Places of the winter orders after their verb (specs/gdd.md § Phase d'hiver). */
export const WINTER_SLOTS: Record<string, SlotKind[]> = {
  'R N': ['nobleCard', 'ownSettlement'],
  'C N': ['ownNoble', 'otherNoble'],
  'D N': ['noble', 'dignityCard', 'region'],
  'S R': ['ownNoble'],
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
  'K R': ['ownNoble'],
  'V R': ['noble'],
  'W X': ['ownNoble', 'territory', 'adjacent'],
  'W R': ['ownNoble', 'territory'],
  'W F': ['ownNoble', 'territory'],
  'W G': ['ownNoble', 'territory', 'noble'],
  'M N': ['ownNoble', 'otherNoble'],
}

/** Places of the special card orders after `P` and the card (specs/ordres-speciaux.md). */
export const SPECIAL_SLOTS: Record<string, SlotKind[]> = {
  P: ['specialCard'],
  'D C': ['specialCard'],
  'P BT': ['region'],
  'P RA': ['region'],
  'P RE': ['revoltTarget'],
  'P AG': ['ownNoble', 'region'],
  'P AP': ['ownNoble', 'region'],
  'P TX': ['ownNoble', 'territory'],
  'P DI': ['ownNoble', 'region'],
  'P SO': ['noble'],
  'P AS': ['ownNoble', 'otherNoble'],
  'P JU': ['ownNoble'],
  'P EM': ['ownArmy', 'adjacent'],
}

/** Orders whose last place repeats: the territories of a new fief. */
const REPEATING = new Set(['T F'])

/**
 * Codes a sheet order expects at the typed word, from its verb and the words
 * already written after it: the longest verb of `table` the line starts with
 * gives the places, the number of words after it the place being typed.
 */
export function sheetSlotCodes(
  index: CodeIndex,
  table: Record<string, SlotKind[]>,
  textBefore: string,
): string[] | undefined {
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
    return before ? [...(index.neighbours.get(before) ?? [])] : undefined
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
  if (slot?.includes(upper)) return []
  if (upper.length === 3 && index.codes.some((entry) => entry.code === upper)) return []
  if (word === '' && !slot) return []
  const folded = fold(word)
  const slotRank = new Map((slot ?? []).map((code, position) => [code, position]))
  const rank = (entry: IndexedCode) =>
    slotRank.has(entry.code)
      ? (slotRank.get(entry.code) ?? 0) - slotRank.size
      : entry.own
        ? 1
        : 2
  const seen = new Set<string>()
  return index.codes
    .filter((entry) =>
      word === ''
        ? slotRank.has(entry.code)
        : entry.code.startsWith(upper) || entry.folded.startsWith(folded),
    )
    .sort((a, b) => rank(a) - rank(b) || a.code.localeCompare(b.code))
    .filter((entry) => !seen.has(entry.code) && Boolean(seen.add(entry.code)))
    .slice(0, limit)
    .map(({ code, name, kind }) => ({ code, name, kind }))
}

/** Suggester of a chain: the codes its orders expect at the cursor come first. */
export function chainSuggester(index: CodeIndex): CodeSuggester {
  return (word, textBefore) => suggestCodes(index, word, chainSlotCodes(index, textBefore))
}

/** Suggester of a sheet of orders (winter or special cards) read through `table`. */
export function sheetSuggester(index: CodeIndex, table: Record<string, SlotKind[]>): CodeSuggester {
  return (word, textBefore) => suggestCodes(index, word, sheetSlotCodes(index, table, textBefore))
}
