import type { Claim, Fief, Noble, SimulationAction, StateData } from '@/types'

/** One way a noble can die in the simulation (specs/titres.md § Lisibilité et simulateur). */
export interface KillVariant {
  key: string
  /** Second noble dying in the same settlement. */
  partner?: string
  action: SimulationAction
}

/** Who holds a fief once an action is applied, read from the projected state. */
export interface FiefOutcome {
  fief: string
  /** Name of the titulaire; absent while the fief is vacant. */
  holder?: string
  /** Name of the exiled noble, if the fief is usurped. */
  exile?: string
}

/** The fiefs a noble's death settles: the ones they hold or are exiled from. */
export function affectedFiefs(code: string, fiefs: Fief[]): Fief[] {
  return fiefs.filter((fief) => fief.holder === code || fief.exiled === code)
}

/** The fiefs the noble is exiled from. */
export function exiledFiefs(code: string, fiefs: Fief[]): Fief[] {
  return fiefs.filter((fief) => fief.exiled === code)
}

/** The fiefs the noble holds as a usurper. */
export function usurpedFiefs(code: string, fiefs: Fief[]): Fief[] {
  return fiefs.filter((fief) => fief.exiled && fief.holder === code)
}

/**
 * The outcomes a death offers beyond the plain one, derived from the current
 * (hypothetical) state: the death of the other title holder of a usurped
 * fief, and the absence of a garrison when a claim bears on a titulaire. The
 * plain kill is always first; a garrison noble is picked by a click instead.
 */
export function killVariants(
  code: string,
  nobles: Noble[],
  fiefs: Fief[],
  claims: Claim[],
): KillVariant[] {
  const variants: KillVariant[] = [
    { key: 'plain', action: { type: 'kill', noble: code } },
  ]
  const affected = affectedFiefs(code, fiefs)
  const partners = new Set<string>()
  for (const fief of affected) {
    if (!fief.exiled || !fief.holder) continue
    const other = fief.holder === code ? fief.exiled : fief.holder
    if (nobles.some((n) => n.code === other)) partners.add(other)
  }
  for (const partner of partners) {
    variants.push({
      key: `together:${partner}`,
      partner,
      action: { type: 'kill', noble: code, alsoKill: partner },
    })
  }
  if (hasClaimedHolding(code, fiefs, claims)) {
    variants.push({
      key: 'noGarrison',
      action: { type: 'kill', noble: code, noGarrison: true },
    })
  }
  return variants
}

/** A claim bears on the noble and they hold a fief of their own: a garrison matters. */
export function hasClaimedHolding(code: string, fiefs: Fief[], claims: Claim[]): boolean {
  return (
    claims.some((claim) => claim.target === code && !claim.ended) &&
    fiefs.some((fief) => fief.holder === code && !fief.exiled)
  )
}

/** Reads the holder and exile of the given fiefs (by capital) in a projected state. */
export function fiefOutcomes(
  state: StateData,
  capitals: string[],
  nameOf: (code: string) => string,
): FiefOutcome[] {
  return capitals.map((capital) => {
    const fief = (state.fiefs ?? []).find((f) => f.capital === capital)
    return {
      fief: capital,
      holder: fief?.holder ? nameOf(fief.holder) : undefined,
      exile: fief?.exiled ? nameOf(fief.exiled) : undefined,
    }
  })
}

/** Whether two outcome lists read the same. */
export function sameOutcomes(a: FiefOutcome[], b: FiefOutcome[]): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}
