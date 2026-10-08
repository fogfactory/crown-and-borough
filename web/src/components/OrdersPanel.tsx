import { useState, type ChangeEvent } from 'react'
import { IconBook, IconSnowflake } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { RulesSection } from '@/components/RulesPanel'
import { formatCardHand, formatCardLabel } from '@/lib/card-hand'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import { DEFAULT_HAND_LIMIT } from '@/types'
import type {
  Dignity,
  Noble,
  NobleCard,
  OrdersPreview,
  OrdersPreviewError,
  PlayerId,
  StateData,
  VoiceSource,
  WinterLinePreview,
} from '@/types'

interface OrdersPanelProps {
  state: StateData
  player: PlayerId
  chainDrafts: Record<string, string>
  winterDraft: string
  /** Server dry run of the drafts; null until the first preview arrives. */
  preview?: OrdersPreview | null
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

// Dignities a married lady cannot receive (the others suit a married lady).
const MARRIAGE_EXCLUDED_DIGNITIES: Dignity[] = ['d_arc', 'abbess', 'herbalist', 'chevalier_d_eon']

// A lady dignity needs a lady; the bastard is open to any noble.
function canReceiveDignity(noble: Noble, dignity: Dignity): boolean {
  if ((noble.dignities ?? []).includes(dignity)) return false
  if (dignity === 'bastard') return true
  if (noble.sex !== 'female') return false
  return !noble.spouse || !MARRIAGE_EXCLUDED_DIGNITIES.includes(dignity)
}

function ownedNobles(state: StateData, player: PlayerId): Noble[] {
  return state.nobles.filter((noble) => noble.owner === player)
}

// Nobles of other players married to one of the player's living nobles: the
// possible targets of a claim.
function claimTargets(state: StateData, player: PlayerId): Noble[] {
  const byCode = new Map(state.nobles.map((noble) => [noble.code, noble]))
  const targets = new Map<string, Noble>()
  for (const marriage of state.marriages ?? []) {
    const spouseA = byCode.get(marriage.nobleA)
    const spouseB = byCode.get(marriage.nobleB)
    if (!spouseA || !spouseB) continue
    for (const [mine, other] of [
      [spouseA, spouseB],
      [spouseB, spouseA],
    ]) {
      if (mine.owner === player && other.owner !== player) targets.set(other.code, other)
    }
  }
  return [...targets.values()]
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

function PreviewErrors({
  errors,
  label,
}: {
  errors: Array<{ line?: number; message: string }>
  label: string
}) {
  const { t } = useLanguage()
  if (errors.length === 0) return null

  return (
    <ul
      role="alert"
      aria-label={label}
      className="max-h-32 list-disc space-y-1 overflow-y-auto rounded-md border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 pl-7 text-xs text-[#8d321e]"
    >
      {errors.map((error, index) => (
        <li key={`${error.line ?? 0}-${index}`}>
          {error.line
            ? t('error.line', { line: error.line, message: error.message })
            : error.message}
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

/** Appends one order line to a winter draft, on its own line. */
export function appendDraftLine(draft: string, line: string): string {
  const trimmed = draft.replace(/\s+$/, '')
  return trimmed === '' ? `${line}\n` : `${trimmed}\n${line}\n`
}

function draftLines(draft: string): string[] {
  return draft
    .split('\n')
    .map((line) => line.replace(/#.*$/, '').trim().toUpperCase())
    .filter((line) => line !== '')
}

/** True when the draft already carries a `T N` draw order. */
export function draftHasNobleDraw(draft: string): boolean {
  return draftLines(draft).some((line) => /^T\s+N$/.test(line))
}

/** Counts the `D C CCC` lines of the draft: each frees a hand slot. */
function draftNobleDiscardCount(draft: string): number {
  return draftLines(draft).filter((line) => /^D\s+C\s+[A-Z]{3}$/.test(line)).length
}

/** Counts the `C N HHH CCC` lines of the draft: each consumes a claim card. */
function draftClaimCount(draft: string): number {
  return draftLines(draft).filter((line) => /^C\s+N\s+/.test(line)).length
}

function draftMentionsCard(draft: string, code: string): boolean {
  const upper = code.toUpperCase()
  return draftLines(draft).some((line) => {
    const fields = line.split(/\s+/)
    return (
      (fields[0] === 'R' && fields[1] === 'N' && fields[2] === upper) ||
      (fields[0] === 'D' && fields[1] === 'N' && fields[3] === upper) ||
      (fields[0] === 'D' && fields[1] === 'C' && fields[2] === upper)
    )
  })
}

function NobleCardRow({
  card,
  player,
  state,
  winterDraft,
  onWinterChange,
  cardsOnly = false,
}: {
  card: NobleCard
  player: PlayerId
  state: StateData
  winterDraft: string
  onWinterChange: (text: string) => void
  /** Outside winter only cards can be played: no discard. */
  cardsOnly?: boolean
}) {
  const { t } = useLanguage()
  const [target, setTarget] = useState('')
  const [claimTarget, setClaimTarget] = useState('')
  const [abbeyRegion, setAbbeyRegion] = useState('')
  const isClaim = card.kind === 'claim'
  const claimCardCount = (state.nobleHand ?? []).filter((c) => c.kind === 'claim').length
  const used = isClaim
    ? draftClaimCount(winterDraft) >= claimCardCount
    : draftMentionsCard(winterDraft, card.code)
  const isDignity = card.kind === 'dignity'
  const needsRegion = card.dignity === 'abbess'
  const claimOptions = isClaim
    ? claimTargets(state, player).map((noble) => ({
        value: noble.code,
        label: `${noble.code} · ${noble.name}`,
      }))
    : []
  const selectedClaim = claimOptions.some((option) => option.value === claimTarget)
    ? claimTarget
    : (claimOptions[0]?.value ?? '')
  const options =
    isDignity || isClaim
      ? ownedNobles(state, player)
          .filter((noble) =>
            isClaim
              ? !(noble.dignities ?? []).includes('bastard')
              : canReceiveDignity(noble, card.dignity ?? 'bastard'),
          )
          .map((noble) => ({ value: noble.code, label: `${noble.code} · ${noble.name}` }))
      : state.territories
          .filter(
            (territory) =>
              territory.owner === player &&
              territory.army?.owner === player &&
              territory.infrastructures.some(
                (infra) => infra.type === 'castle' || infra.type === 'village',
              ),
          )
          .map((territory) => ({ value: territory.id, label: territory.id }))
  const selected = options.some((option) => option.value === target)
    ? target
    : (options[0]?.value ?? '')
  const label = isClaim
    ? t('orders.nobleHandClaim', { code: card.code })
    : isDignity
      ? t('orders.nobleHandDignity', {
          dignity: t(`dignity.${card.dignity ?? 'bastard'}` as MessageKey),
          code: card.code,
        })
      : t('orders.nobleHandNoble', {
          name: card.name ?? card.code,
          code: card.code,
          sex: t(
            card.sex === 'female' ? 'orders.nobleHandFemale' : 'orders.nobleHandMale',
          ),
        })
  const discard = () => {
    onWinterChange(appendDraftLine(winterDraft, `D C ${card.code}`))
  }
  const play = () => {
    if (selected === '' || (isClaim && selectedClaim === '')) return
    if (needsRegion && abbeyRegion.trim().length !== 3) return
    const line = isClaim
      ? `C N ${selected} ${selectedClaim}`
      : isDignity
        ? `D N ${selected} ${card.code}${needsRegion ? ` ${abbeyRegion.toUpperCase()}` : ''}`
        : `R N ${card.code} ${selected}`
    onWinterChange(appendDraftLine(winterDraft, line))
  }
  return (
    <li className="flex flex-wrap items-center gap-2 text-xs text-[#263f52]">
      <span className="font-medium">{label}</span>
      <select
        value={selected}
        onChange={(event) => setTarget(event.target.value)}
        className="rounded border border-[#9bbbd3] bg-white px-1 py-0.5"
        aria-label={t(
          isClaim
            ? 'orders.nobleHandClaimHeirAria'
            : isDignity
              ? 'orders.nobleHandDignityTargetAria'
              : 'orders.nobleHandTerritoryAria',
          { code: card.code },
        )}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      {needsRegion && (
        <input
          value={abbeyRegion}
          maxLength={3}
          onChange={(event) => setAbbeyRegion(event.target.value)}
          placeholder={t('orders.abbeyRegionPlaceholder')}
          aria-label={t('orders.abbeyRegionAria', { code: card.code })}
          className="w-16 rounded border border-[#9bbbd3] bg-white px-1 py-0.5 uppercase"
        />
      )}
      {isClaim && (
        <select
          value={selectedClaim}
          onChange={(event) => setClaimTarget(event.target.value)}
          className="rounded border border-[#9bbbd3] bg-white px-1 py-0.5"
          aria-label={t('orders.nobleHandClaimTargetAria', { code: card.code })}
        >
          {claimOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      )}
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={
          used ||
          selected === '' ||
          (isClaim && selectedClaim === '') ||
          (needsRegion && abbeyRegion.trim().length !== 3)
        }
        onClick={play}
      >
        {t(
          isClaim
            ? 'orders.nobleHandPlayClaim'
            : isDignity
              ? 'orders.nobleHandPlayDignity'
              : 'orders.nobleHandPlayTerritory',
        )}
      </Button>
      {!cardsOnly && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={used}
          aria-label={t('orders.nobleHandDiscardAria', { code: card.code })}
          onClick={discard}
        >
          {t('orders.nobleHandDiscard')}
        </Button>
      )}
    </li>
  )
}

function NobleDeckSection({
  state,
  player,
  winterDraft,
  onWinterChange,
  cardsOnly = false,
}: {
  state: StateData
  player: PlayerId
  winterDraft: string
  onWinterChange: (text: string) => void
  /** Outside winter: play the cards of the hand, no draw nor discard. */
  cardsOnly?: boolean
}) {
  const { t } = useLanguage()
  const hand = state.nobleHand ?? []
  const deckSize = state.nobleDeckSize ?? 0
  const drawn = draftHasNobleDraw(winterDraft)
  const handLimit = state.handLimit ?? DEFAULT_HAND_LIMIT
  const specialCount = (state.specialHand ?? []).length
  const handFull =
    specialCount + hand.length - draftNobleDiscardCount(winterDraft) >= handLimit
  const drawDisabled = drawn || deckSize === 0 || handFull
  return (
    <section className="space-y-2 rounded-lg border border-[#9bbbd3] bg-[#f7fbff] p-3">
      <h4 className="font-serif text-base font-semibold text-[#2c5b7d]">
        {t('orders.nobleDeckTitle')}
      </h4>
      {(state.spiedHands ?? []).map((hand) => (
        <p key={hand.player} className="text-xs font-medium text-[#2c5b7d]">
          {t('orders.spiedHand', {
            player: hand.player,
            cards: [
              ...hand.specialHand.map((kind) => t(`card.${kind}` as MessageKey)),
              ...hand.nobleHand.map((card) => card.name ?? card.code),
            ].join(', '),
          })}
        </p>
      ))}
      {(state.calamityForecast ?? []).length > 0 && (
        <p className="text-xs font-medium text-[#2c5b7d]">
          {t('orders.calamityForecast', {
            kinds: (state.calamityForecast ?? [])
              .map((kind) => t(`card.${kind}` as MessageKey))
              .join(', '),
          })}
        </p>
      )}
      <p className="text-xs leading-relaxed text-[#55738a]">
        {t(
          cardsOnly
            ? 'orders.nobleCardsActionDescription'
            : 'orders.nobleDeckDescription',
        )}
      </p>
      {!cardsOnly && (
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={drawDisabled}
            onClick={() => onWinterChange(appendDraftLine(winterDraft, 'T N'))}
          >
            {t('orders.nobleDraw')}
          </Button>
          <span className="text-xs text-[#55738a]">
            {t('orders.nobleDeckSize', { count: deckSize })}
          </span>
          {drawn && (
            <span className="text-xs text-[#55738a]">{t('orders.nobleDrawUsed')}</span>
          )}
          {!drawn && deckSize === 0 && (
            <span className="text-xs text-[#8d321e]">{t('orders.nobleDeckEmpty')}</span>
          )}
          {!drawn && deckSize > 0 && handFull && (
            <span className="text-xs text-[#8d321e]">{t('orders.nobleHandFull')}</span>
          )}
        </div>
      )}
      <p className="text-xs text-[#55738a]">
        {t('orders.handCounter', {
          count: specialCount + hand.length,
          limit: handLimit,
          special: specialCount,
          noble: hand.length,
        })}
      </p>
      <p className="text-xs font-semibold text-[#2c5b7d]">{t('orders.nobleHand')}</p>
      {hand.length === 0 ? (
        <p className="text-xs text-[#55738a]">{t('orders.nobleHandEmpty')}</p>
      ) : (
        <ul className="space-y-1" aria-label={t('orders.nobleHand')}>
          {hand.map((card) => (
            <NobleCardRow
              key={card.id}
              card={card}
              player={player}
              state={state}
              winterDraft={winterDraft}
              onWinterChange={onWinterChange}
              cardsOnly={cardsOnly}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

function voiceSourceLabel(source: VoiceSource, t: Translate): string {
  if (source.kind === 'title') {
    return t('orders.voiceTitle', {
      name: source.nobleName ?? source.noble ?? '',
      title: t(`orders.voiceTitle.${source.title ?? 'bishop'}` as MessageKey),
      votes: source.votes,
    })
  }
  return t(source.kind === 'seat' ? 'orders.voiceSeat' : 'orders.voiceTerritory', {
    territory: source.territory ?? '',
    votes: source.votes,
  })
}

function OpenElectionsSection({
  state,
  winterDraft,
  onWinterChange,
}: {
  state: StateData
  winterDraft: string
  onWinterChange: (value: string) => void
}) {
  const { t } = useLanguage()
  const elections = state.openElections ?? []
  if (elections.length === 0) return null
  const lines = draftLines(winterDraft)

  return (
    <section className="space-y-2 rounded-lg border border-[#9bbbd3] bg-[#f7fbff] p-3">
      <h4 className="font-serif text-base font-semibold text-[#2c5b7d]">
        {t('orders.electionsTitle')}
      </h4>
      <p className="text-xs leading-relaxed text-[#55738a]">
        {t('orders.electionsTimingNote')}
      </p>
      <TooltipProvider delayDuration={100}>
        {elections.map((election) => {
          const name =
            state.bishoprics?.find((bishopric) => bishopric.region === election.region)
              ?.name ?? election.region ?? ''
          const isPope = election.kind === 'pope'
          const seat = election.seat ?? ''
          return (
            <details
              key={`${election.kind}-${election.region ?? ''}`}
              open
              className="rounded-md border border-[#c9dcea] bg-white/60 px-2 py-1.5 text-xs text-[#2c5b7d]"
            >
              <summary className="cursor-pointer font-semibold">
                {isPope
                  ? t('orders.electionPopeHeading')
                  : t('orders.electionBishopHeading', { name, seat })}
              </summary>
              <div className="mt-1.5 space-y-1.5">
                <p className="text-[#55738a]">
                  {isPope
                    ? t('orders.electionPopeRule', { required: election.required ?? 0 })
                    : t('orders.electionBishopRule', { seat })}
                </p>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span
                      tabIndex={0}
                      className="inline-block cursor-help border-b border-dotted border-[#5c94bd] font-medium"
                    >
                      {t('orders.electionVoices', { voices: election.voices })}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent className="flex-col items-start">
                    {election.voiceSources.length === 0 ? (
                      <span>{t('orders.electionNoVoice')}</span>
                    ) : (
                      election.voiceSources.map((source, index) => (
                        <span key={index}>{voiceSourceLabel(source, t)}</span>
                      ))
                    )}
                  </TooltipContent>
                </Tooltip>
                {election.candidates.length > 0 ? (
                  <div className="space-y-1">
                    <p className="text-[#55738a]">{t('orders.electionCandidateHint')}</p>
                    <div className="flex flex-wrap gap-1.5">
                      {election.candidates.map((candidate) => {
                        const line = isPope
                          ? `K P ${candidate.code}`
                          : `K E ${candidate.code} ${seat}`
                        const added = lines.includes(line.toUpperCase())
                        const commented = `${line} # ${t('orders.electionCandidacyComment', { name: candidate.name })}`
                        return (
                          <Button
                            key={candidate.code}
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={added}
                            title={added ? t('orders.electionCandidateAdded') : line}
                            onClick={() =>
                              onWinterChange(appendDraftLine(winterDraft, commented))
                            }
                          >
                            {candidate.name} ({candidate.code})
                          </Button>
                        )
                      })}
                    </div>
                  </div>
                ) : (
                  <p className="text-[#55738a]">{t('orders.electionNoCandidate')}</p>
                )}
              </div>
            </details>
          )
        })}
      </TooltipProvider>
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
  diagnostics: WinterLinePreview[]
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
          className={
            diagnostic.reason === 'insufficient_resources' ? undefined : 'font-semibold'
          }
        >
          {t('error.line', {
            line: diagnostic.line,
            message: t(
              `reports.reason.${diagnostic.reason ?? 'insufficient_resources'}` as MessageKey,
            ),
          })}
        </li>
      ))}
    </ul>
  )
}

/** Chain errors, as opposed to line errors of the winter sheet. */
function chainErrors(errors: OrdersPreviewError[]): OrdersPreviewError[] {
  return errors.filter((error) => error.noble)
}

export function OrdersPanel({
  state,
  player,
  chainDrafts,
  winterDraft,
  preview = null,
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
    const winterLines = preview?.winter ?? []
    const winterErrors = winterLines
      .filter((line) => line.status === 'invalid')
      .map((line) => ({ line: line.line, message: line.message ?? '' }))
    const winterDiagnostics = winterLines.filter((line) => line.status === 'rejected')
    const winterEstimate = preview?.winterCost ?? null
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
        <OpenElectionsSection
          state={state}
          winterDraft={winterDraft}
          onWinterChange={onWinterChange}
        />
        <NobleDeckSection
          state={state}
          player={player}
          winterDraft={winterDraft}
          onWinterChange={onWinterChange}
        />
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
        <PreviewErrors errors={winterErrors} label={t('orders.winterErrorsAria')} />
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
      {(state.nobleHand ?? []).length > 0 && (
        <>
          <NobleDeckSection
            state={state}
            player={player}
            winterDraft={winterDraft}
            onWinterChange={onWinterChange}
            cardsOnly
          />
          <textarea
            value={winterDraft}
            onChange={(event) => onWinterChange(event.target.value)}
            className="min-h-16 w-full resize-y rounded-lg border border-[#9bbbd3] bg-[#f7fbff] p-3 font-mono text-xs text-[#263f52] outline-none transition focus:border-[#5c94bd] focus:ring-2 focus:ring-[#5c94bd]/20"
            placeholder={t('orders.nobleCardsPlaceholder')}
            aria-label={t('orders.nobleCardsAria', { player })}
          />
          <PreviewErrors
            errors={(preview?.winter ?? [])
              .filter((line) => line.status === 'invalid')
              .map((line) => ({ line: line.line, message: line.message ?? '' }))}
            label={t('orders.winterErrorsAria')}
          />
          <WinterOrderDiagnostics
            diagnostics={(preview?.winter ?? []).filter(
              (line) => line.status === 'rejected',
            )}
            t={t}
          />
        </>
      )}
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
      <PreviewErrors
        errors={chainErrors(preview?.errors ?? [])}
        label={t('orders.chainErrorsAria')}
      />
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
        disabled={
          submitting ||
          (!hasEmittingNoble && specialDraft.trim() === '' && winterDraft.trim() === '')
        }
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
