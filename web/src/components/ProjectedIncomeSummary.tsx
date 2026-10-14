import { TerritoryRef } from '@/components/MapLinks'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { richTranslate } from '@/lib/rich-text'
import { treasuryOf, type Treasury } from '@/lib/treasury'
import type { FiefTitle, PlayerId, StateData } from '@/types'

interface ProjectedIncomeSummaryProps {
  state: StateData | null
  playerId: PlayerId | null
  /** The viewer's drafted special-card orders, whose tax effects are included. */
  specialDraft?: string
  /** Treasury already computed by the caller (the treasury bar). */
  treasury?: Treasury | null
}

const FIEF_TITLE_KEYS: Record<FiefTitle, MessageKey> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

/**
 * Income detail of the treasury bar (charte § Poste de commandement), showing the selected player's normal territory and mill
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
export function ProjectedIncomeSummary({
  state,
  playerId,
  specialDraft,
  treasury: givenTreasury,
}: ProjectedIncomeSummaryProps) {
  const { t } = useLanguage()
  if (!state || !playerId) {
    return null
  }
  const player = state.players.find((candidate) => candidate.id === playerId)
  const treasury = givenTreasury ?? treasuryOf(state, playerId, specialDraft)
  if (!player || !treasury) {
    return null
  }
  const playerFiefs = (state.fiefs ?? []).filter((fief) => fief.owner === playerId)
  const fiefIncomeTotal = playerFiefs.reduce(
    (total, fief) => total + (fief.projectedIncome ?? 0),
    0,
  )
  const nonFiefIncome = (player.projectedIncome ?? 0) - fiefIncomeTotal
  const { taxes } = treasury
  const royalRent = player.projectedRoyalRent ?? 0

  return (
    <div className="text-sm">
      <p className="text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
        {t('app.projectedIncome')}
      </p>
      {playerFiefs.length === 0 ? (
        <p className="mt-1 font-medium">
          {t('app.projectedTerritoryIncomeAmount', {
            amount: player.projectedIncome ?? 0,
          })}
          {player.capitalTerritory ? (
            <>
              {' '}
              {richTranslate(t, 'app.projectedIncomeDestination', {
                destination: <TerritoryRef id={player.capitalTerritory} />,
              })}
            </>
          ) : (
            ''
          )}
        </p>
      ) : (
        <>
          <p className="mt-1 font-medium">
            {t('app.projectedNonFiefIncomeAmount', { amount: nonFiefIncome })}
            {player.capitalTerritory ? (
              <>
                {' '}
                {richTranslate(t, 'app.projectedIncomeDestination', {
                  destination: <TerritoryRef id={player.capitalTerritory} />,
                })}
              </>
            ) : (
              ''
            )}
          </p>
          {playerFiefs.map((fief) => (
            <p key={fief.capital} className="mt-1 font-medium">
              {richTranslate(
                t,
                fief.capital in taxes.seigneurial
                  ? 'app.projectedFiefIncomeTaxedAmount'
                  : 'app.projectedFiefIncomeAmount',
                {
                  title: t(FIEF_TITLE_KEYS[fief.title]),
                  capital: <TerritoryRef id={fief.capital} />,
                  amount:
                    (fief.projectedIncome ?? 0) + (taxes.seigneurial[fief.capital] ?? 0),
                },
              )}
            </p>
          ))}
        </>
      )}
      <p className="mt-1 font-medium">
        {t('app.projectedMillIncomeAmount', { amount: player.projectedMillIncome ?? 0 })}
      </p>
      {royalRent > 0 && (
        <p className="mt-1 font-medium">
          {t('app.projectedRoyalRentAmount', { amount: royalRent })}
        </p>
      )}
      {taxes.titheDrafted && (
        <p className="mt-1 font-medium">
          {t('app.projectedTitheAmount', { amount: taxes.tithe })}
        </p>
      )}
      {taxes.royalDrafted && (
        <p className="mt-1 font-medium">
          {t('app.projectedRoyalTaxAmount', { amount: taxes.royalSupplement })}
        </p>
      )}
      <p className="mt-1 border-t border-line-soft pt-1 font-bold">
        {t('app.projectedTotalAmount', { amount: treasury.income })}
      </p>
    </div>
  )
}
