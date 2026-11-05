import { useId, useState } from 'react'
import { IconChevronDown, IconChevronUp, IconScale } from '@tabler/icons-react'

import { ProjectedConsumptionSummary } from '@/components/ProjectedConsumptionSummary'
import { ProjectedIncomeSummary } from '@/components/ProjectedIncomeSummary'
import { useLanguage } from '@/i18n/LanguageContext'
import { treasuryOf } from '@/lib/treasury'
import { cn } from '@/lib/utils'
import type { PlayerId, StateData } from '@/types'

interface TreasuryBarProps {
  state: StateData | null
  playerId: PlayerId | null
  /** The viewer's drafted special-card orders, whose tax effects count. */
  specialDraft?: string
}

/**
 * One-line treasury of the command post (charte-graphique.md § Poste de
 * commandement): stock, income per action turn, consumption and famine risk,
 * unfolding on the income and consumption detail.
 */
export function TreasuryBar({ state, playerId, specialDraft }: TreasuryBarProps) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)
  const detailId = useId()
  if (!state || !playerId) return null
  const treasury = treasuryOf(state, playerId, specialDraft)
  if (!treasury) return null
  const winter = state.season === 'winter'
  const atRisk = treasury.armiesAtRisk

  return (
    <section
      aria-label={t('treasury.title')}
      className="rounded-md border border-line bg-paper-100 text-sm"
    >
      <button
        type="button"
        aria-expanded={open}
        aria-controls={detailId}
        className="flex min-h-11 w-full items-center gap-2 px-3 py-1.5 text-left lg:min-h-0"
        onClick={() => setOpen((current) => !current)}
      >
        <IconScale aria-hidden="true" className="size-4 shrink-0 text-ink-500" />
        <span
          data-treasury-summary
          className="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-1.5 font-semibold text-ink-900"
        >
          <span>{t('treasury.stock', { amount: treasury.stock })}</span>
          <span aria-hidden="true">·</span>
          <span>{t('treasury.income', { amount: treasury.income })}</span>
          {!winter && (
            <>
              <span aria-hidden="true">·</span>
              <span>{t('treasury.consumption', { amount: treasury.consumption })}</span>
              <span aria-hidden="true">·</span>
              <span
                data-famine-risk={atRisk > 0 || undefined}
                className={cn(atRisk > 0 ? 'text-warning-700' : 'font-normal text-ink-500')}
              >
                {atRisk === 0
                  ? t('treasury.famineNone')
                  : t(atRisk === 1 ? 'treasury.famineArmy' : 'treasury.famineArmies', {
                      count: atRisk,
                    })}
              </span>
            </>
          )}
        </span>
        {open ? (
          <IconChevronUp aria-hidden="true" className="size-4 shrink-0 text-ink-500" />
        ) : (
          <IconChevronDown aria-hidden="true" className="size-4 shrink-0 text-ink-500" />
        )}
      </button>
      {open && (
        <div id={detailId} className="space-y-3 border-t border-line-soft px-3 py-2">
          <ProjectedIncomeSummary
            state={state}
            playerId={playerId}
            specialDraft={specialDraft}
            treasury={treasury}
          />
          <ProjectedConsumptionSummary state={state} playerId={playerId} />
        </div>
      )}
    </section>
  )
}
