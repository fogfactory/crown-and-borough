import { useLanguage } from '@/i18n/LanguageContext'
import type { PlayerId, StateData } from '@/types'

interface ProjectedIncomeSummaryProps {
  state: StateData | null
  playerId: PlayerId | null
}

/**
 * Command post line showing the selected player's normal territory income
 * for the next action turn (never in winter): territories and villages
 * already credit the player's capital before ravitaillement, so this is a
 * steady-state projection, not last turn's actual result.
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

  return (
    <div className="rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 py-2 text-sm">
      <p className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
        {t('app.projectedIncome')}
      </p>
      <p className="mt-1 font-medium">
        {t('app.projectedIncomeAmount', { amount: player.projectedIncome ?? 0 })}
        {player.capitalTerritory
          ? ' ' + t('app.projectedIncomeDestination', { destination: player.capitalTerritory })
          : ''}
      </p>
    </div>
  )
}
