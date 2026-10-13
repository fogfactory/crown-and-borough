import type { CrownKind } from '@/lib/heraldry'
import { religiousTitlesByHolder } from '@/lib/lineage'
import type { FiefTitle, StateData } from '@/types'

/**
 * What a noble's pawn wears (graphic charter, Nobles): the crown of its
 * highest secular title, or a courtesy crown, plus its religious attributes.
 * Dignities never show on the map.
 */
export interface NobleAttributes {
  crown?: CrownKind
  /** Courtesy title of a fief holder's or the king's spouse; never with `crown`. */
  courtesyCrown?: CrownKind
  /** Bishop: a crosier held beside the pawn. */
  crosier: boolean
  /** Cardinal: red robe and hat; pope: white robe and tiara (the pope's win). */
  robe?: 'cardinal' | 'pope'
  excommunicated: boolean
}

/** Secular ranks from the highest down. */
const CROWN_RANK: CrownKind[] = ['king', 'duchy', 'marquisate', 'county', 'barony']

function highest(kinds: CrownKind[]): CrownKind | undefined {
  return CROWN_RANK.find((kind) => kinds.includes(kind))
}

/** Pawn attributes of every noble who wears at least one, by noble code. */
export function nobleAttributesByCode(state: StateData): Map<string, NobleAttributes> {
  const religious = religiousTitlesByHolder({
    bishoprics: state.bishoprics,
    cardinals: state.cardinals,
    pope: state.pope,
    king: state.king,
    queen: state.queen,
    queenDowager: state.queenDowager,
  })
  const fiefTitlesByHolder = new Map<string, FiefTitle[]>()
  for (const fief of state.fiefs ?? []) {
    if (!fief.holder) continue
    fiefTitlesByHolder.set(fief.holder, [
      ...(fiefTitlesByHolder.get(fief.holder) ?? []),
      fief.title,
    ])
  }
  const secularCrown = (code: string | undefined): CrownKind | undefined => {
    if (!code) return undefined
    const kinds: CrownKind[] = [...(fiefTitlesByHolder.get(code) ?? [])]
    if (state.king === code) kinds.push('king')
    return highest(kinds)
  }
  const excommunicated = new Set((state.excommunicated ?? []).map((entry) => entry.noble))

  const attributes = new Map<string, NobleAttributes>()
  for (const noble of state.nobles) {
    const held = religious.get(noble.code) ?? []
    const titles = new Set(held.map((title) => title.title))
    const crown = secularCrown(noble.code)
    // A courtesy title follows the spouse's real title; the queen keeps hers
    // as a dowager.
    const courtesyCrown = crown
      ? undefined
      : titles.has('queen') || titles.has('dowager_queen')
        ? 'king'
        : secularCrown(noble.spouse)
    const entry: NobleAttributes = {
      crown,
      courtesyCrown,
      crosier: titles.has('bishop'),
      robe: titles.has('pope') ? 'pope' : titles.has('cardinal') ? 'cardinal' : undefined,
      excommunicated: excommunicated.has(noble.code),
    }
    if (
      entry.crown ||
      entry.courtesyCrown ||
      entry.crosier ||
      entry.robe ||
      entry.excommunicated
    ) {
      attributes.set(noble.code, entry)
    }
  }
  return attributes
}
