import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { FiefTitle, PlayerId, StateData } from '@/types'

interface ProjectedIncomeSummaryProps {
  state: StateData | null
  playerId: PlayerId | null
}

const FIEF_TITLE_KEYS: Record<FiefTitle, MessageKey> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

/**
 * Command post lines showing the selected player's normal territory and mill
 * income for the next action turn (never in winter): territories, villages,
 * and mills already credit their destinations before ravitaillement, so this
 * is a steady-state projection, not last turn's actual result. The two are
 * shown separately because mills do not route through the capital and can
 * credit several different settlements, unlike territory income.
 *
 * Territory income splits into one line per fief the player holds (credited
 * to that fief's own capital, see titres.md #196) plus a remaining line for
 * income routed outside any fief, since a single "→ capital" line would
 * otherwise hide where a fief's territories actually pay.
 */
export function ProjectedIncomeSummary({ state, playerId }: ProjectedIncomeSummaryProps) {
  const { t } = useLanguage()
  if (!state || !playerId) {
    return null
  }
  const player = state.players.find((candidate) => candidate.id === playerId)
  if (!player) {
    return null
  }
  const playerFiefs = (state.fiefs ?? []).filter((fief) => fief.owner === playerId)
  const fiefIncomeTotal = playerFiefs.reduce(
    (total, fief) => total + (fief.projectedIncome ?? 0),
    0,
  )
  const nonFiefIncome = (player.projectedIncome ?? 0) - fiefIncomeTotal
  const totalResources = state.territories.reduce(
    (total, territory) => total + (territory.owner === playerId ? territory.resources : 0),
    0,
  )

  return (
    <div className="rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 py-2 text-sm">
      <p className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
        {t('app.projectedIncome')}
      </p>
      {playerFiefs.length === 0 ? (
        <p className="mt-1 font-medium">
          {t('app.projectedTerritoryIncomeAmount', {
            amount: player.projectedIncome ?? 0,
          })}
          {player.capitalTerritory
            ? ' ' +
              t('app.projectedIncomeDestination', {
                destination: player.capitalTerritory,
              })
            : ''}
        </p>
      ) : (
        <>
          <p className="mt-1 font-medium">
            {t('app.projectedNonFiefIncomeAmount', { amount: nonFiefIncome })}
            {player.capitalTerritory
              ? ' ' +
                t('app.projectedIncomeDestination', {
                  destination: player.capitalTerritory,
                })
              : ''}
          </p>
          {playerFiefs.map((fief) => (
            <p key={fief.capital} className="mt-1 font-medium">
              {t('app.projectedFiefIncomeAmount', {
                title: t(FIEF_TITLE_KEYS[fief.title]),
                capital: fief.capital,
                amount: fief.projectedIncome ?? 0,
              })}
            </p>
          ))}
        </>
      )}
      <p className="mt-1 font-medium">
        {t('app.projectedMillIncomeAmount', { amount: player.projectedMillIncome ?? 0 })}
      </p>
      <p className="mt-2 border-t border-[#d4c4b0] pt-2 font-medium">
        {t('app.currentResourcesAmount', { amount: totalResources })}
      </p>
    </div>
  )
}
