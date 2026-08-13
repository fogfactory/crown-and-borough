import type { ChangeEvent } from 'react'
import { BookOpen, Snowflake } from 'lucide-react'

import { Button } from '@/components/ui/button'
import type { RulesSection } from '@/components/RulesPanel'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { Noble, PlayerId, StateData } from '@/types'

interface OrdersPanelProps {
  state: StateData
  player: PlayerId
  chainDrafts: Record<string, string>
  winterDraft: string
  submitted: boolean
  submitting: boolean
  error: string | null
  onChainChange: (noble: string, text: string) => void
  onWinterChange: (text: string) => void
  onSubmit: () => void
  onOpenRules: (section: RulesSection) => void
}

function ownedNobles(state: StateData, player: PlayerId): Noble[] {
  return state.nobles.filter((noble) => noble.owner === player)
}

function chainPlaceholder(): string {
  return 'XXX A YYY'
}

function RulesButton({
  section,
  onOpenRules,
}: {
  section: RulesSection
  onOpenRules: (section: RulesSection) => void
}) {
  const { t } = useLanguage()

  return (
    <Button
      type="button"
      variant="outline"
      className="w-full border-[#b7a786] bg-[#fffaf0] text-[#594b3c] hover:bg-[#f3ead9] hover:text-[#30291f]"
      onClick={() => onOpenRules(section)}
    >
      <BookOpen aria-hidden="true" className="size-4" />
      {t('orders.rulesShortcut')}
    </Button>
  )
}

function OrderError({ error }: { error: string | null }) {
  if (!error) return null

  return (
    <p
      role="alert"
      className="rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-xs text-[#8d321e]"
    >
      {error}
    </p>
  )
}

export function OrdersPanel({
  state,
  player,
  chainDrafts,
  winterDraft,
  submitted,
  submitting,
  error,
  onChainChange,
  onWinterChange,
  onSubmit,
  onOpenRules,
}: OrdersPanelProps) {
  const { t } = useLanguage()
  const handleChainChange =
    (noble: Noble) => (event: ChangeEvent<HTMLTextAreaElement>) => {
      onChainChange(noble.code, event.target.value)
    }

  if (state.season === 'winter') {
    return (
      <section className="space-y-3 rounded-xl border border-[#9bbbd3] bg-[#eaf3ff]/80 p-4 shadow-inner shadow-[#b8d3e8]/40">
        <div>
          <h3 className="flex items-center gap-2 font-serif text-lg font-bold text-[#2c5b7d]">
            <Snowflake aria-hidden="true" className="size-5 text-[#5c94bd]" />
            <span>{t('orders.winterTitle')}</span>
          </h3>
          <p className="mt-1 text-xs leading-relaxed text-[#55738a]">
            {t('orders.winterDescription')}
          </p>
        </div>
        <textarea
          value={winterDraft}
          onChange={(event) => onWinterChange(event.target.value)}
          className="min-h-36 w-full resize-y rounded-lg border border-[#9bbbd3] bg-[#f7fbff] p-3 font-mono text-xs text-[#263f52] outline-none transition focus:border-[#5c94bd] focus:ring-2 focus:ring-[#5c94bd]/20"
          placeholder={t('orders.winterPlaceholder')}
          aria-label={t('orders.winterAria', { player })}
        />
        {submitted && (
          <p className="text-xs text-[#376341]">{t('orders.submittedEditable')}</p>
        )}
        <Button
          type="button"
          variant="outline"
          className="w-full border-[#6f9fc1] bg-[#d8ebfa] text-[#244c68] hover:bg-[#c8e1f2] hover:text-[#1d3e56]"
          disabled={submitting}
          onClick={onSubmit}
        >
          {submitting
            ? t('orders.sending')
            : submitted
              ? t('orders.editWinter')
              : t('orders.submitWinter')}
        </Button>
        <OrderError error={error} />
        <RulesButton section="winter-orders" onOpenRules={onOpenRules} />
      </section>
    )
  }

  const nobles = ownedNobles(state, player)
  const hasEmittingNoble = nobles.some((noble) => noble.status !== 'dungeon')
  return (
    <section className="space-y-3 border-t border-[#b7a786]/50 pt-5">
      <div>
        <h3 className="font-serif text-lg font-semibold">{t('orders.actionTitle')}</h3>
        <p className="mt-1 text-xs leading-relaxed text-[#806f57]">
          {t('orders.actionDescription')}
        </p>
      </div>
      {nobles.length === 0 ? (
        <p className="rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] p-3 text-sm italic text-[#806f57]">
          {t('orders.noNobleAvailable')}
        </p>
      ) : (
        nobles.map((noble) => (
          <label key={noble.code} className="block space-y-1.5">
            <span className="flex items-center justify-between text-xs font-semibold uppercase tracking-[0.12em] text-[#806f57]">
              <span>
                {noble.code} · {noble.name}
              </span>
              <span
                className={
                  noble.status === 'dungeon' ? 'text-[#a84632]' : 'text-[#376341]'
                }
              >
                {t(`orders.nobleStatus.${noble.status}` as MessageKey)}
              </span>
            </span>
            <textarea
              value={chainDrafts[noble.code] ?? ''}
              onChange={handleChainChange(noble)}
              disabled={noble.status === 'dungeon'}
              className="min-h-32 w-full resize-y rounded-lg border border-[#b7a786] bg-[#f8f0e2] p-3 font-mono text-xs text-[#30291f] outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20 disabled:cursor-not-allowed disabled:opacity-50"
              placeholder={chainPlaceholder()}
              aria-label={t('orders.chainAria', { noble: noble.code })}
            />
          </label>
        ))
      )}
      {submitted && (
        <p className="text-xs text-[#376341]">{t('orders.submittedEditable')}</p>
      )}
      {!hasEmittingNoble && (
        <p className="rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] p-3 text-sm italic text-[#806f57]">
          {t('orders.noEmittingNoble')}
        </p>
      )}
      <Button
        type="button"
        className="w-full"
        disabled={submitting || !hasEmittingNoble}
        onClick={onSubmit}
      >
        {submitting
          ? t('orders.sending')
          : submitted
            ? t('orders.edit')
            : t('orders.submit')}
      </Button>
      <OrderError error={error} />
      <RulesButton section="action-orders" onOpenRules={onOpenRules} />
    </section>
  )
}
