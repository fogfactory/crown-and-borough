import type { ChangeEvent } from 'react'
import { IconBook, IconSnowflake } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import type { RulesSection } from '@/components/RulesPanel'
import { formatCardHand, formatCardLabel } from '@/lib/card-hand'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import { estimateWinterCost } from '@/lib/winter-cost'
import { parseWinterDraftDetailed, type WinterParseError } from '@/lib/winter-parse'
import { simulateWinterDraft, type WinterSimulationOutcome } from '@/lib/winter-overlay'
import type { MapData, Noble, PlayerId, StateData, WinterCosts } from '@/types'

interface OrdersPanelProps {
  state: StateData
  player: PlayerId
  chainDrafts: Record<string, string>
  winterDraft: string
  winterCosts?: WinterCosts | null
  map?: MapData
  specialDraft: string
  submitted: boolean
  submitting: boolean
  error: string | null
  draftDiffers?: {
    chains?: Record<string, boolean>
    winter?: boolean
  }
  onChainChange: (noble: string, text: string) => void
  onWinterChange: (text: string) => void
  onSpecialChange: (text: string) => void
  onSubmit: () => void
  onOpenRules: (section: RulesSection) => void
  onRestoreFromServer?: (target?: string) => void
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
      <IconBook aria-hidden="true" className="size-4" />
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

function WinterOrderErrors({ errors, t }: { errors: WinterParseError[]; t: Translate }) {
  if (errors.length === 0) return null

  return (
    <ul
      role="alert"
      aria-label={t('orders.winterErrorsAria')}
      className="max-h-32 list-disc space-y-1 overflow-y-auto rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 pl-7 text-xs text-[#8d321e]"
    >
      {errors.map((error) => (
        <li key={`${error.line}-${error.key}`}>
          {t('error.line', {
            line: error.line,
            message: t(error.key, error.values),
          })}
        </li>
      ))}
    </ul>
  )
}

function CalamityWarnings({ state }: { state: StateData }) {
  const { t } = useLanguage()
  const announcements = state.announcements ?? []
  if (announcements.length === 0) return null

  return (
    <div
      role="alert"
      className="space-y-1 rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-xs text-[#8d321e]"
    >
      <p className="font-semibold">{t('orders.calamityWarningTitle')}</p>
      <ul className="list-disc space-y-0.5 pl-4">
        {announcements.map((announcement, index) => (
          <li
            key={`${announcement.year}-${announcement.season}-${announcement.kind}-${index}`}
          >
            {t('orders.calamityWarningItem', {
              card: formatCardLabel(announcement.kind, t),
              season: t(SEASON_LABEL_KEYS[announcement.season]),
              region: announcement.region,
            })}
          </li>
        ))}
      </ul>
    </div>
  )
}

function DeckOrdersSection({
  state,
  specialDraft,
  onSpecialChange,
}: {
  state: StateData
  specialDraft: string
  onSpecialChange: (text: string) => void
}) {
  const { t } = useLanguage()
  const hand = state.specialHand ?? []
  return (
    <section className="space-y-2 rounded-lg border border-[#c8b0d9] bg-[#fbf5ff] p-3">
      <h4 className="font-serif text-base font-semibold text-[#684b7d]">
        {t('orders.deckTitle')}
      </h4>
      <p className="text-xs leading-relaxed text-[#806f57]">
        {t('orders.deckDescription')}
      </p>
      <p className="text-xs text-[#684b7d]">
        {t('orders.deckHand')}: {formatCardHand(hand, t)}
      </p>
      <CalamityWarnings state={state} />
      <textarea
        value={specialDraft}
        onChange={(event) => onSpecialChange(event.target.value)}
        className="min-h-20 w-full resize-y rounded-lg border border-[#c8b0d9] bg-white p-3 font-mono text-xs text-[#30291f] outline-none focus:border-[#8a5ba6] focus:ring-2 focus:ring-[#8a5ba6]/20"
        placeholder={t('orders.deckPlaceholder')}
        aria-label={t('orders.deckAria')}
      />
    </section>
  )
}

function DeckHandSummary({ state }: { state: StateData }) {
  const { t } = useLanguage()
  const hand = state.specialHand ?? []

  return (
    <section className="space-y-2 rounded-lg border border-[#c8b0d9] bg-[#fbf5ff] p-3">
      <h4 className="font-serif text-base font-semibold text-[#684b7d]">
        {t('orders.deckTitle')}
      </h4>
      <p className="text-xs leading-relaxed text-[#806f57]">
        {t('orders.deckWinterDescription')}
      </p>
      <p className="text-xs text-[#684b7d]">
        {t('orders.deckHand')}: {formatCardHand(hand, t)}
      </p>
      <CalamityWarnings state={state} />
    </section>
  )
}

function WinterOrderDiagnostics({
  diagnostics,
  t,
}: {
  diagnostics: WinterSimulationOutcome[]
  t: Translate
}) {
  if (diagnostics.length === 0) return null

  return (
    <ul
      role="status"
      aria-label={t('orders.winterDiagnosticsAria')}
      className="max-h-32 list-disc space-y-1 overflow-y-auto rounded-md border border-[#e07a30]/40 bg-[#fdf0e3] px-3 py-2 pl-7 text-xs text-[#8a5216]"
    >
      {diagnostics.map((diagnostic) => (
        <li
          key={`${diagnostic.line}-${diagnostic.reason ?? ''}`}
          className={diagnostic.valid ? undefined : 'font-semibold'}
        >
          {t('error.line', {
            line: diagnostic.line,
            message: diagnostic.reason
              ? t(`reports.reason.${diagnostic.reason}` as MessageKey)
              : t('reports.reason.insufficient_resources'),
          })}
        </li>
      ))}
    </ul>
  )
}

export function OrdersPanel({
  state,
  player,
  chainDrafts,
  winterDraft,
  winterCosts,
  map,
  specialDraft,
  submitted,
  submitting,
  error,
  draftDiffers,
  onChainChange,
  onWinterChange,
  onSpecialChange,
  onSubmit,
  onOpenRules,
  onRestoreFromServer,
}: OrdersPanelProps) {
  const { t } = useLanguage()
  const handleChainChange =
    (noble: Noble) => (event: ChangeEvent<HTMLTextAreaElement>) => {
      onChainChange(noble.code, event.target.value)
    }

  if (state.season === 'winter') {
    const parsedWinterDraft = parseWinterDraftDetailed(winterDraft, {
      map,
      nobles: state.nobles,
    })
    const winterSimulation = map
      ? simulateWinterDraft(state, player, winterDraft, map, winterCosts)
      : null
    const winterDiagnostics = (winterSimulation?.outcomes ?? [])
      .filter(
        (outcome) =>
          (outcome.valid && outcome.warning) ||
          (!outcome.valid && outcome.reason && !outcome.reason.startsWith('error.')),
      )
      .sort((first, second) => first.line - second.line)
    const winterEstimate = winterCosts
      ? estimateWinterCost(state, player, winterCosts, winterDraft, map)
      : null
    return (
      <section className="space-y-3 rounded-xl border border-[#9bbbd3] bg-[#eaf3ff]/80 p-4 shadow-inner shadow-[#b8d3e8]/40">
        <div>
          <h3 className="flex items-center gap-2 font-serif text-lg font-bold text-[#2c5b7d]">
            <IconSnowflake aria-hidden="true" className="size-5 text-[#5c94bd]" />
            <span>{t('orders.winterTitle')}</span>
          </h3>
          <p className="mt-1 text-xs leading-relaxed text-[#55738a]">
            {t('orders.winterDescription')}
          </p>
        </div>
        {draftDiffers?.winter && (
          <div className="flex items-center justify-between gap-2 rounded-lg border border-[#815f1e]/40 bg-[#f8e8ae]/60 px-3 py-2 text-xs text-[#6d5118]">
            <span>{t('orders.draftDiffers')}</span>
            {onRestoreFromServer && (
              <button
                type="button"
                className="font-semibold underline hover:text-[#4a360f]"
                onClick={() => onRestoreFromServer('winter')}
              >
                {t('orders.restoreFromServer')}
              </button>
            )}
          </div>
        )}
        <DeckHandSummary state={state} />
        <textarea
          value={winterDraft}
          onChange={(event) => onWinterChange(event.target.value)}
          className="min-h-28 w-full resize-y rounded-lg border border-[#9bbbd3] bg-[#f7fbff] p-3 font-mono text-xs text-[#263f52] outline-none transition focus:border-[#5c94bd] focus:ring-2 focus:ring-[#5c94bd]/20"
          placeholder={t('orders.winterPlaceholder')}
          aria-label={t('orders.winterAria', { player })}
        />
        {winterEstimate && (
          <p
            role="status"
            aria-live="polite"
            className={`text-xs font-semibold ${winterEstimate.spent <= winterEstimate.available ? 'text-[#376341]' : 'text-[#8d321e]'}`}
          >
            {t('orders.winterCostEstimate', {
              spent: winterEstimate.spent,
              available: winterEstimate.available,
            })}
          </p>
        )}
        <WinterOrderErrors errors={parsedWinterDraft.errors} t={t} />
        <WinterOrderDiagnostics diagnostics={winterDiagnostics} t={t} />
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
      <DeckOrdersSection
        state={state}
        specialDraft={specialDraft}
        onSpecialChange={onSpecialChange}
      />
      {nobles.length === 0 ? (
        <p className="rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] p-3 text-sm italic text-[#806f57]">
          {t('orders.noNobleAvailable')}
        </p>
      ) : (
        nobles.map((noble) => (
          <div key={noble.code} className="space-y-1.5">
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
            {draftDiffers?.chains?.[noble.code] && (
              <div className="flex items-center justify-between gap-2 rounded-lg border border-[#815f1e]/40 bg-[#f8e8ae]/60 px-3 py-2 text-xs text-[#6d5118]">
                <span>{t('orders.draftDiffers')}</span>
                {onRestoreFromServer && (
                  <button
                    type="button"
                    className="font-semibold underline hover:text-[#4a360f]"
                    onClick={() => onRestoreFromServer(noble.code)}
                  >
                    {t('orders.restoreFromServer')}
                  </button>
                )}
              </div>
            )}
            <textarea
              value={chainDrafts[noble.code] ?? ''}
              onChange={handleChainChange(noble)}
              disabled={noble.status === 'dungeon'}
              className="min-h-24 w-full resize-y rounded-lg border border-[#b7a786] bg-[#f8f0e2] p-3 font-mono text-xs text-[#30291f] outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20 disabled:cursor-not-allowed disabled:opacity-50"
              placeholder={chainPlaceholder()}
              aria-label={t('orders.chainAria', { noble: noble.code })}
            />
          </div>
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
        disabled={submitting || (!hasEmittingNoble && specialDraft.trim() === '')}
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
