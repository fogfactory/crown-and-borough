import type { Noble, PlayerId, StateData, Territory } from '@/types'

/** A code the order editor can offer, with the name that explains it. */
export interface CodeSuggestion {
  code: string
  name: string
  kind: 'territory' | 'noble'
}

/** The word being typed: its bounds in the text and its letters. */
export interface TypedWord {
  start: number
  end: number
  text: string
}

const LETTER = /\p{L}/u

/**
 * The word the cursor stands at the end of, when it is long enough to look a
 * code up (two letters: a single letter is a verb) and is not in a comment.
 */
export function wordAtCursor(value: string, cursor: number): TypedWord | null {
  if (cursor < value.length && LETTER.test(value[cursor])) return null
  let start = cursor
  while (start > 0 && LETTER.test(value[start - 1])) start--
  if (cursor - start < 2) return null
  const lineStart = value.lastIndexOf('\n', start - 1) + 1
  if (value.slice(lineStart, start).includes('#')) return null
  return { start, end: cursor, text: value.slice(start, cursor) }
}

function fold(text: string): string {
  return text.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase()
}

interface IndexedCode extends CodeSuggestion {
  folded: string
  /** Owned by the player: an army or a controlled territory, or a noble. */
  own: boolean
}

/** Codes the editor can offer: the territories of the map, then the known nobles. */
export interface CodeIndex {
  codes: IndexedCode[]
  neighbours: Map<string, readonly string[]>
}

export function buildCodeIndex(
  territories: readonly Territory[],
  state: Pick<StateData, 'territories' | 'nobles'>,
  player: PlayerId,
): CodeIndex {
  const owned = new Set(
    state.territories
      .filter((territory) => territory.owner === player || territory.army?.owner === player)
      .map((territory) => territory.id),
  )
  const territoryCodes = territories.map<IndexedCode>((territory) => ({
    code: territory.id,
    name: territory.name,
    kind: 'territory',
    folded: fold(territory.name),
    own: owned.has(territory.id),
  }))
  const nobleCodes = state.nobles.map<IndexedCode>((noble: Noble) => ({
    code: noble.code,
    name: noble.name,
    kind: 'noble',
    folded: fold(noble.name),
    own: noble.owner === player,
  }))
  return {
    codes: [...territoryCodes, ...nobleCodes],
    neighbours: new Map(territories.map((territory) => [territory.id, territory.adjacencies])),
  }
}

/** Last territory code written on the line before the typed word. */
function contextTerritory(index: CodeIndex, lineBefore: string): string | undefined {
  const words = lineBefore.toUpperCase().match(/\p{L}{3}(?!\p{L})/gu) ?? []
  return words.reverse().find((word) => index.neighbours.has(word))
}

/**
 * Codes that complete the typed word, by code or by name: the neighbours of
 * the last territory of the line first, then the player's own territories and
 * nobles, then the others. A word that already is a whole code needs nothing.
 * The suggestions only help typing; the preview judges the order.
 */
export function suggestCodes(
  index: CodeIndex,
  word: string,
  lineBefore: string,
  limit = 8,
): CodeSuggestion[] {
  const upper = word.toUpperCase()
  if (index.codes.some((entry) => entry.code === upper)) return []
  const folded = fold(word)
  const context = contextTerritory(index, lineBefore)
  const neighbours = new Set(context ? index.neighbours.get(context) : [])
  const rank = (entry: IndexedCode) =>
    entry.kind === 'territory' && neighbours.has(entry.code) ? 0 : entry.own ? 1 : 2
  return index.codes
    .filter((entry) => entry.code.startsWith(upper) || entry.folded.startsWith(folded))
    .sort((a, b) => rank(a) - rank(b) || a.code.localeCompare(b.code))
    .slice(0, limit)
    .map(({ code, name, kind }) => ({ code, name, kind }))
}
