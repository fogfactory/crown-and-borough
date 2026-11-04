import { TerritoryRef } from '@/components/MapLinks'
import { INFRASTRUCTURE_LABEL_KEYS } from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import { richTranslate } from '@/lib/rich-text'
import type { PlayerId, StateData } from '@/types'

interface ProjectedConsumptionSummaryProps {
  state: StateData | null
  playerId: PlayerId | null
}

/**
 * Consumption detail of the treasury bar (charte § Poste de commandement),
 * showing the selected player's projected ration
 * consumption for the next action turn, and the armies that would starve if
 * nothing changes before resolution (see the FAQ for the two things this
 * can't foresee: an undrawn calamity card, and any order change before
 * submission). Never shown in winter, since ravitaillement does not happen
 * then.
 */
export function ProjectedConsumptionSummary({ state, playerId }: ProjectedConsumptionSummaryProps) {
  const { t } = useLanguage()
  if (!state || !playerId || state.season === 'winter') {
    return null
  }
  const player = state.players.find((candidate) => candidate.id === playerId)
  if (!player) {
    return null
  }
  const armiesAtRisk = player.armiesAtRisk ?? []

  return (
    <div className="text-sm">
      <p className="text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
        {t('app.projectedConsumption')}
      </p>
      <p className="mt-1 font-medium">
        {t('app.projectedConsumptionAmount', { amount: player.projectedConsumption ?? 0 })}
      </p>
      <p className="mt-2 text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
        {t('app.armiesAtRiskTitle')}
      </p>
      {armiesAtRisk.length > 0 ? (
        <ul className="mt-1 space-y-0.5">
          {armiesAtRisk.map((risk) => (
            <li key={risk.territoryId}>
              {richTranslate(t, 'app.armyAtRisk', {
                territory: <TerritoryRef id={risk.territoryId} />,
                size: risk.size,
                deficit: risk.deficit,
                supplied: risk.suppliedStrength,
              })}
              {risk.pillageInfrastructure
                ? t(risk.savedByPillage ? 'app.armyAtRiskSavedByPillage' : 'app.armyAtRiskPillage', {
                    infrastructure: t(INFRASTRUCTURE_LABEL_KEYS[risk.pillageInfrastructure]),
                  })
                : ''}
              {risk.suppliedStrength === 0 && risk.expectedLoss > 0
                ? t('app.armyAtRiskVanishes')
                : risk.expectedLoss > 0
                  ? t('app.armyAtRiskLoss', { count: risk.expectedLoss })
                  : ''}
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-1 italic text-ink-500">{t('app.noArmyAtRisk')}</p>
      )}
    </div>
  )
}
