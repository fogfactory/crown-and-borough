import { royalIssuerOf } from '@/lib/royal'
import type { PlayerId, StateData } from '@/types'

/** Effect of the tax cards drafted this turn on the viewer's projected income. */
export interface DraftedTaxEffects {
  /** Extra income, by fief capital, of the seigneurial taxes (the fief's income doubles). */
  seigneurial: Record<string, number>
  /** Supplement of the royal taxes, created for the player's capital. */
  royalSupplement: number
  /** Mill production of other players diverted to the player's capital by the drafted tithes. */
  tithe: number
  /** A tithe is drafted: its line shows even when it diverts nothing. */
  titheDrafted: boolean
  /** A royal tax is drafted: its line shows even when it creates nothing. */
  royalDrafted: boolean
}

/** One drafted Tax card line, `P TX|ST|DI|TT [HHH] [XXX]`. */
export interface DraftedTaxLine {
  /** Written with the explicit tithe code (`DI`, `TT`). */
  tithe: boolean
  /** The issuing noble; absent for `P TX XXX` on a vacant fief. */
  issuer?: string
  /** The fief capital or bishopric seed; absent for the pope's tithe `P DI HHH`. */
  target?: string
}

const TAX_LINE = /^P (TX|ST|DI|TT)(?: ([A-Z]{3}))?(?: ([A-Z]{3}))?(?:\s|$)/

/** Reads one line of the special-orders draft, null when it is no Tax card line. */
export function parseDraftTaxLine(raw: string): DraftedTaxLine | null {
  const match = TAX_LINE.exec(raw.trim())
  if (!match) return null
  const [, code, first, second] = match
  const tithe = code === 'DI' || code === 'TT'
  if (second) return { tithe, issuer: first, target: second }
  return tithe ? { tithe, issuer: first } : { tithe, target: first }
}

/**
 * Projects the effect of the drafted Tax cards (tithes aside, see
 * draftedTitheIncome): a seigneurial tax on a fief of the player doubles its
 * territory income; a royal tax, issued by the king or the regent on any other
 * fief, creates the same amount again for the player's capital. The lord's tax
 * wins over a royal tax on the same fief (titres.md "Taxe seigneuriale"); each
 * fief is taxed at most once.
 */
export function draftedTaxEffects(
  state: StateData,
  playerId: PlayerId,
  specialDraft: string | undefined,
): DraftedTaxEffects {
  const effects: DraftedTaxEffects = {
    seigneurial: {},
    royalSupplement: 0,
    tithe: 0,
    titheDrafted: false,
    royalDrafted: false,
  }
  if (!specialDraft) return effects
  const fiefs = state.fiefs ?? []
  const royalIssuer = royalIssuerOf(state)
  const royalIssuerIsOurs = royalIssuer?.owner === playerId
  const royal = new Set<string>()
  for (const raw of specialDraft.split('\n')) {
    const line = parseDraftTaxLine(raw)
    if (!line || line.tithe || !line.target) continue
    const target = line.target
    const fief = fiefs.find((candidate) => candidate.capital === target)
    if (!fief) continue
    const ownsIt =
      fief.owner === playerId && (!line.issuer || !fief.holder || fief.holder === line.issuer)
    if (ownsIt) {
      effects.seigneurial[target] = fief.projectedIncome ?? 0
    } else if (royalIssuerIsOurs && line.issuer === royalIssuer?.code) {
      royal.add(target)
    }
  }
  for (const target of royal) {
    if (target in effects.seigneurial) continue
    effects.royalDrafted = true
    effects.royalSupplement += fiefs.find((fief) => fief.capital === target)?.projectedIncome ?? 0
  }
  const tithe = draftedTitheIncome(state, playerId, specialDraft)
  effects.tithe = tithe.amount
  effects.titheDrafted = tithe.drafted
  return effects
}

/**
 * Seeds of the bishoprics the drafted tithes cover: every bishopric for the
 * pope's untargeted tithe, the named one for a cardinal or a bishop. A `P TX
 * HHH XXX` on a fief capital is a tax, not a tithe.
 */
export function draftedTitheSeeds(
  state: StateData,
  playerId: PlayerId,
  specialDraft: string,
): Set<string> {
  const fiefCapitals = new Set((state.fiefs ?? []).map((fief) => fief.capital))
  const bishoprics = state.bishoprics ?? []
  const seeds = new Set<string>()
  for (const raw of specialDraft.split('\n')) {
    const line = parseDraftTaxLine(raw)
    if (!line?.issuer) continue
    const cleric = state.nobles.find((noble) => noble.code === line.issuer && noble.owner === playerId)
    if (!cleric?.religiousTitle) continue
    if (cleric.religiousTitle === 'pope') {
      if (line.tithe) for (const bishopric of bishoprics) seeds.add(bishopric.region)
    } else if (line.target && bishoprics.some((bishopric) => bishopric.region === line.target)) {
      if (line.tithe || !fiefCapitals.has(line.target)) seeds.add(line.target)
    }
  }
  return seeds
}

/**
 * Mill production the drafted tithes would divert to the player's capital: the
 * mills of the tithed bishoprics held by someone else, since the player's own
 * mills are already in the mill projection. Other players' competing tithes
 * are not visible here.
 */
function draftedTitheIncome(
  state: StateData,
  playerId: PlayerId,
  specialDraft: string,
): { amount: number; drafted: boolean } {
  const seeds = draftedTitheSeeds(state, playerId, specialDraft)
  let amount = 0
  for (const bishopric of state.bishoprics ?? []) {
    if (!seeds.has(bishopric.region)) continue
    for (const territory of state.territories) {
      if (bishopric.territories.includes(territory.id) && territory.owner !== playerId) {
        amount += territory.millProduction ?? 0
      }
    }
  }
  return { amount, drafted: seeds.size > 0 }
}
