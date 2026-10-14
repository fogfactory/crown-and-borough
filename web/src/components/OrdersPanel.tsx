import { type ChangeEvent } from 'react'
import { IconBook, IconSnowflake } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import { OrderLauncher } from '@/components/OrderDialog'
import { AppeasementOrders, SpecialCardOrders, assassinationLauncher, justiceLauncher, TitleOrdersSection, shortCode } from '@/components/TitleOrders'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { RulesSection } from '@/components/RulesPanel'
import { SectionHeading } from '@/components/SectionHeading'
import { TerritoryRef } from '@/components/MapLinks'
import { richTranslate } from '@/lib/rich-text'
import { formatCardLabel, formatCardName } from '@/lib/card-hand'
import { CardPictogram } from '@/components/CardPictogram'
import { HandFan, type FanCard } from '@/components/HandFan'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { appendDraftLine, draftLines } from '@/lib/winter-draft'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import { DEFAULT_HAND_LIMIT } from '@/types'
import type {
  CardKind,
  Dignity,
  Noble,
  NobleCard,
  OrdersPreview,
  OrdersPreviewError,
  PlayerId,
  Region,
  Territory,
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
  /** Regions of the map (seed villages), for the orders that name one. */
  regions?: Region[]
  /** Map territories, for the commune names of the Tax card targets. */
  mapTerritories?: Territory[]
  specialDraft: string
  draftDiffers?: {
    chains?: Record<string, boolean>
    winter?: boolean
  }
  onChainChange: (noble: string, text: string) => void
  onWinterChange: (text: string) => void
  onSpecialChange: (text: string) => void
  onOpenRules: (section: RulesSection) => void
  onRestoreFromServer?: (target?: string) => void
}

// Dignities a married lady cannot receive (the others suit a married lady).
const MARRIAGE_EXCLUDED_DIGNITIES: Dignity[] = ['d_arc', 'abbess', 'herbalist', 'chevalier_d_eon']

// A lady dignity needs a lady; the bastard is open to any noble.
// Dignities only their owner knows (the others see a lady without them).
const HIDDEN_DIGNITIES: Dignity[] = ['chevalier_d_eon', 'correspondent', 'spy', 'poisoner', 'witch']

function isHiddenDignity(dignity: Dignity): boolean {
  return HIDDEN_DIGNITIES.includes(dignity)
}

function canReceiveDignity(noble: Noble, dignity: Dignity): boolean {
  const held = noble.dignities ?? []
  if (held.includes(dignity)) return false
  if (dignity === 'bastard') return true
  // A lady carries at most one visible and one hidden dignity of the ladies.
  if (
    held.some(
      (other) =>
        other !== 'bastard' &&
        other !== 'cardinal' &&
        isHiddenDignity(other) === isHiddenDignity(dignity),
    )
  ) {
    return false
  }
  if (noble.sex !== 'female') return false
  return !noble.spouse || !MARRIAGE_EXCLUDED_DIGNITIES.includes(dignity)
}

function ownedNobles(state: StateData, player: PlayerId): Noble[] {
  return state.nobles.filter((noble) => noble.owner === player)
}

function hasEmittingNobleOf(state: StateData, player: PlayerId): boolean {
  return ownedNobles(state, player).some((noble) => noble.status !== 'dungeon')
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

export { appendDraftLine }

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
      className="w-full border-line bg-paper-50 text-ink-700 hover:bg-paper-100 hover:text-ink-900"
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
      className="rounded-md border border-vermilion-700/30 bg-vermilion-100 px-3 py-2 text-xs text-vermilion-700"
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
      className="max-h-32 list-disc space-y-1 overflow-y-auto rounded-md border border-vermilion-700/30 bg-vermilion-100 px-3 py-2 pl-7 text-xs text-vermilion-700"
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
    // A forecast is not an error: warning tones, not the vermilion of alerts.
    <div
      role="status"
      data-calamity-forecast
      className="space-y-1 rounded-md border border-warning-700/40 bg-warning-100 px-3 py-2 text-xs text-warning-700"
    >
      <p className="font-semibold">{t('orders.calamityWarningTitle')}</p>
      <ul className="list-disc space-y-0.5 pl-4">
        {announcements.map((announcement, index) => (
          <li
            key={`${announcement.year}-${announcement.season}-${announcement.kind}-${index}`}
          >
            {richTranslate(t, announcement.ritual ? 'orders.calamityWarningItemOmens' : 'orders.calamityWarningItem', {
              card: formatCardLabel(announcement.kind, t),
              season: t(SEASON_LABEL_KEYS[announcement.season]),
              region: <TerritoryRef id={announcement.region} />,
            })}
          </li>
        ))}
      </ul>
    </div>
  )
}

/** The special-card sheet: appeasement orders and the typed card orders. */
function DeckOrdersSection({
  state,
  player,
  specialDraft,
  onSpecialChange,
  onOpenRules,
}: {
  state: StateData
  player: PlayerId
  onOpenRules?: (section: RulesSection) => void
  specialDraft: string
  onSpecialChange: (text: string) => void
}) {
  const { t } = useLanguage()
  return (
    <section className="space-y-2 rounded-lg border border-line bg-paper-100 p-3">
      <SectionHeading
        title={t('orders.deckTitle')}
        topic={t('orders.deckTitle')}
        summary={t('orders.deckSummary')}
        help={t('orders.deckDescription')}
        rulesSection="action-orders"
        onOpenRules={onOpenRules}
        titleClassName="font-display text-base font-semibold text-ink-700"
      />
      <AppeasementOrders
        state={state}
        player={player}
        draft={specialDraft}
        onChange={onSpecialChange}
      />
      <textarea
        value={specialDraft}
        onChange={(event) => onSpecialChange(event.target.value)}
        className="min-h-20 w-full resize-y rounded-lg border border-line bg-paper-50 p-3 font-mono text-xs text-ink-900 outline-none focus:border-ink-700 focus:ring-2 focus:ring-ink-700/20"
        placeholder={t('orders.deckPlaceholder')}
        aria-label={t('orders.deckAria')}
      />
    </section>
  )
}

/** Number of `D C CODE` lines of the draft discarding this card. */
function draftDiscardCount(lines: string[], code: string): number {
  const discard = new RegExp(`^D\\s+C\\s+${code}(\\s|$)`)
  return lines.filter((line) => discard.test(line)).length
}

/** True when the draft already carries a `T N` draw order. */
export function draftHasNobleDraw(draft: string): boolean {
  return draftLines(draft).some((line) => /^T\s+N$/.test(line))
}

/** Counts the `D C CCC` lines of the draft: each frees a hand slot. */
function draftNobleDiscardCount(draft: string): number {
  return draftLines(draft).filter((line) => /^D\s+C\s+[A-Z]{2,3}$/.test(line)).length
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

/** Play and discard controls of a noble, claim or dignity card of the hand. */
function NobleCardActions({
  card,
  player,
  state,
  regions,
  winterDraft,
  onWinterChange,
  cardsOnly = false,
}: {
  card: NobleCard
  player: PlayerId
  state: StateData
  regions: Region[]
  winterDraft: string
  onWinterChange: (text: string) => void
  /** Outside winter only cards can be played: no discard. */
  cardsOnly?: boolean
}) {
  const { t } = useLanguage()
  const isClaim = card.kind === 'claim'
  const claimCardCount = (state.nobleHand ?? []).filter((c) => c.kind === 'claim').length
  const used = isClaim
    ? draftClaimCount(winterDraft) >= claimCardCount
    : draftMentionsCard(winterDraft, card.code)
  const isDignity = card.kind === 'dignity'
  const needsRegion = card.dignity === 'abbess'
  // A visible dignity can be played on any eligible noble, an opponent's
  // included (own nobles first); a claim needs one of the player's own nobles as heir.
  const nobleOptions = [...state.nobles]
    .filter((noble) =>
      isClaim
        ? noble.owner === player && !(noble.dignities ?? []).includes('bastard')
        : canReceiveDignity(noble, card.dignity ?? 'bastard') &&
          // A hidden dignity goes on one of your own ladies only, so the
          // order cannot be used to probe other players' hidden dignities.
          (noble.owner === player || !isHiddenDignity(card.dignity ?? 'bastard')),
    )
    .sort((a, b) => Number(b.owner === player) - Number(a.owner === player))
    .map((noble) => ({
      value: noble.code,
      label: `${noble.name} (${noble.code})${noble.owner === player ? '' : ` · ${noble.owner}`}`,
    }))
  const claimOptions = isClaim
    ? claimTargets(state, player).map((noble) => ({
        value: noble.code,
        label: `${noble.name} (${noble.code})`,
      }))
    : []
  const territoryOptions = state.territories
    .filter(
      (territory) =>
        territory.owner === player &&
        territory.army?.owner === player &&
        territory.infrastructures.some(
          (infra) => infra.type === 'castle' || infra.type === 'village',
        ),
    )
    .map((territory) => ({ value: territory.id, label: territory.id }))
  const regionOptions = regions.map((region) => ({
    value: region.seed,
    label: region.name ? `${region.name} (${region.seed})` : region.seed,
  }))
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
  const nobleName = (code: string) =>
    state.nobles.find((noble) => noble.code === code)?.name ?? code
  const fields = isClaim
    ? [
        { key: 'heir', label: t('orders.field.heir'), options: nobleOptions },
        { key: 'target', label: t('orders.field.claimed'), options: claimOptions },
      ]
    : isDignity
      ? [
          { key: 'noble', label: t('orders.field.noble'), options: nobleOptions },
          ...(needsRegion
            ? [
                regionOptions.length > 0
                  ? { key: 'region', label: t('orders.field.region'), options: regionOptions }
                  : {
                      key: 'region',
                      label: t('orders.field.region'),
                      placeholder: t('orders.abbeyRegionPlaceholder'),
                      maxLength: 3,
                    },
              ]
            : []),
        ]
      : [{ key: 'territory', label: t('orders.field.territory'), options: territoryOptions }]
  const missingOptions = fields.some((field) => field.options?.length === 0)
  const buildOrder = (values: Record<string, string>) => {
    if (isClaim) {
      if (!values.heir || !values.target) return null
      return {
        line: `C N ${values.heir} ${values.target}`,
        comment: t('orders.comment.claim', {
          name: nobleName(values.target),
          heir: nobleName(values.heir),
        }),
      }
    }
    if (isDignity) {
      if (!values.noble) return null
      if (needsRegion && values.region.trim().length !== 3) return null
      return {
        line: `D N ${values.noble} ${card.code}${needsRegion ? ` ${values.region.toUpperCase()}` : ''}`,
        comment: t('orders.comment.dignity', {
          dignity: t(`dignity.${card.dignity ?? 'bastard'}` as MessageKey),
          name: nobleName(values.noble),
        }),
      }
    }
    if (!values.territory) return null
    return {
      line: `R N ${card.code} ${values.territory}`,
      comment: t('orders.comment.recruit', {
        name: card.name ?? card.code,
        territory: values.territory,
      }),
    }
  }
  return (
    <>
      {missingOptions ? (
        <span className="text-xs italic text-ink-500">{t('hand.noTarget')}</span>
      ) : (
        <OrderLauncher
          label={t('hand.play')}
          title={label}
          disabled={used}
          fields={fields}
          buildOrder={buildOrder}
          onConfirm={(line) => onWinterChange(appendDraftLine(winterDraft, line))}
        />
      )}
      {!cardsOnly && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={used}
          title={t('orders.nobleHandDiscardAria', { code: card.code })}
          onClick={() =>
            onWinterChange(
              appendDraftLine(
                winterDraft,
                `D C ${card.code} # ${t('orders.comment.discard', { name: card.name ?? card.code })}`,
              ),
            )
          }
        >
          {t('hand.discard')}
        </Button>
      )}
    </>
  )
}

/**
 * Controls of a special card of the hand: in winter, play the cards played in
 * every season and discard (`D C KIND`); in an action season, the card's own
 * play orders.
 */
function SpecialCardActions({
  state,
  player,
  kind,
  copies,
  winterDraft,
  onWinterChange,
  cardsOnly,
  regions,
  mapTerritories,
  specialDraft,
  onSpecialChange,
}: {
  state: StateData
  player: PlayerId
  kind: CardKind
  copies: number
  winterDraft: string
  onWinterChange: (text: string) => void
  cardsOnly: boolean
  regions: Region[]
  mapTerritories?: Territory[]
  specialDraft?: string
  onSpecialChange?: (text: string) => void
}) {
  const { t } = useLanguage()
  if (cardsOnly) {
    if (specialDraft === undefined || !onSpecialChange) return null
    return (
      <SpecialCardOrders
        only={kind}
        state={state}
        player={player}
        mapTerritories={mapTerritories}
        regions={regions}
        draft={specialDraft}
        onChange={onSpecialChange}
      />
    )
  }
  const card = formatCardLabel(kind, t)
  const code = shortCode(kind)
  const discarded = draftDiscardCount(draftLines(winterDraft), code)
  const cardLabel = copies > 1 ? `${card} ×${copies}` : card
  // The Assassination and Justice cards are played in every season, winter
  // included: the label is the button that plays them.
  const launcherProps = {
    state,
    t,
    lines: draftLines(winterDraft),
    copies,
    add: (line: string) => onWinterChange(appendDraftLine(winterDraft, line)),
    label: t('hand.play'),
  }
  const playButton =
    kind === 'assassination'
      ? assassinationLauncher({ ...launcherProps, player })
      : kind === 'justice'
        ? justiceLauncher(launcherProps)
        : null
  return (
    <>
      {playButton}
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={discarded >= copies}
        title={t('orders.discardSpecialCard', { card: cardLabel })}
        onClick={() =>
          onWinterChange(
            appendDraftLine(
              winterDraft,
              `D C ${code} # ${t('orders.comment.discardSpecial', { card })}`,
            ),
          )
        }
      >
        {t('hand.discard')}
      </Button>
    </>
  )
}

/**
 * The hand of cards (charte-graphique.md § Poste de commandement): special,
 * noble, claim and dignity cards as a fan; a picked card shows its play and
 * discard controls. Winter adds the noble draw; the deck and the hand limit
 * are written under the fan.
 */
function HandSection({
  state,
  player,
  regions,
  mapTerritories,
  winterDraft,
  onWinterChange,
  specialDraft,
  onSpecialChange,
  cardsOnly = false,
  onOpenRules,
}: {
  state: StateData
  player: PlayerId
  regions: Region[]
  mapTerritories?: Territory[]
  winterDraft: string
  onWinterChange: (text: string) => void
  /** Action seasons: the special-card sheet, where special cards are played. */
  specialDraft?: string
  onSpecialChange?: (text: string) => void
  /** Outside winter: play the cards of the hand, no draw nor discard. */
  cardsOnly?: boolean
  onOpenRules?: (section: RulesSection) => void
}) {
  const { t } = useLanguage()
  const hand = state.nobleHand ?? []
  const deckSize = state.nobleDeckSize ?? 0
  const drawn = draftHasNobleDraw(winterDraft)
  const handLimit = state.handLimit ?? DEFAULT_HAND_LIMIT
  const specialHand = state.specialHand ?? []
  const specialKinds = [...new Set(specialHand)]
  const specialCount = specialHand.length
  const handFull =
    specialCount + hand.length - draftNobleDiscardCount(winterDraft) >= handLimit
  const drawDisabled = drawn || deckSize === 0 || handFull
  const lines = draftLines(winterDraft)
  const nobleName = (card: NobleCard) =>
    card.kind === 'claim'
      ? t('hand.claimCard')
      : card.kind === 'dignity'
        ? t(`dignity.${card.dignity ?? 'bastard'}` as MessageKey)
        : (card.name ?? card.code)

  const cards: FanCard[] = [
    ...specialKinds.map((kind): FanCard => {
      const copies = specialHand.filter((card) => card === kind).length
      const code = shortCode(kind)
      const discarded = draftDiscardCount(lines, code)
      return {
        key: `special:${kind}`,
        family: 'special',
        name: formatCardName(kind, t),
        code,
        icon: <CardPictogram kind={kind} />,
        copies,
        discarded: discarded > 0 && discarded >= copies,
        actions: (
          <SpecialCardActions
            state={state}
            player={player}
            kind={kind}
            copies={copies}
            winterDraft={winterDraft}
            onWinterChange={onWinterChange}
            cardsOnly={cardsOnly}
            regions={regions}
            mapTerritories={mapTerritories}
            specialDraft={specialDraft}
            onSpecialChange={onSpecialChange}
          />
        ),
      }
    }),
    ...hand.map(
      (card): FanCard => ({
        key: `noble:${card.id}`,
        family: card.kind === 'dignity' ? 'dignity' : 'noble',
        name: nobleName(card),
        code: card.code,
        icon: <CardPictogram kind={card.kind} />,
        discarded: draftDiscardCount(lines, card.code) > 0,
        actions: (
          <NobleCardActions
            card={card}
            player={player}
            state={state}
            regions={regions}
            winterDraft={winterDraft}
            onWinterChange={onWinterChange}
            cardsOnly={cardsOnly}
          />
        ),
      }),
    ),
  ]

  return (
    <section className="space-y-2 rounded-lg border border-line bg-paper-100 p-3">
      <SectionHeading
        title={t('orders.handTitle')}
        topic={t('orders.handTitle')}
        summary={t(cardsOnly ? 'orders.nobleCardsActionDescription' : 'orders.nobleDeckSummary')}
        help={t('orders.nobleDeckDescription')}
        rulesSection={cardsOnly ? 'action-orders' : 'winter-orders'}
        onOpenRules={onOpenRules}
        titleClassName="font-display text-base font-semibold text-ink-700"
      />
      <CalamityWarnings state={state} />
      {(state.spiedHands ?? []).map((spied) => (
        <p key={spied.player} className="text-xs font-medium text-ink-700">
          {t('orders.spiedHand', {
            player: spied.player,
            cards: [
              ...spied.specialHand.map((kind) => t(`card.${kind}` as MessageKey)),
              ...spied.nobleHand.map((card) => card.name ?? card.code),
            ].join(', '),
          })}
        </p>
      ))}
      {(state.calamityForecast ?? []).length > 0 && (
        <p className="text-xs font-medium text-warning-700">
          {t('orders.calamityForecast', {
            kinds: (state.calamityForecast ?? [])
              .map((kind) => t(`card.${kind}` as MessageKey))
              .join(', '),
          })}
        </p>
      )}
      <HandFan
        cards={cards}
        label={t('orders.hand')}
        footer={
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-ink-500">
            <span data-hand-counter>
              {t('hand.counter', {
                deck: deckSize,
                count: specialCount + hand.length,
                limit: handLimit,
              })}
            </span>
            {!cardsOnly && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={drawDisabled}
                onClick={() => onWinterChange(appendDraftLine(winterDraft, 'T N'))}
              >
                {t('orders.nobleDraw')}
              </Button>
            )}
            {!cardsOnly && drawn && <span>{t('orders.nobleDrawUsed')}</span>}
            {!cardsOnly && !drawn && deckSize === 0 && (
              <span className="text-warning-700">{t('orders.nobleDeckEmpty')}</span>
            )}
            {!cardsOnly && !drawn && deckSize > 0 && handFull && (
              <span className="text-warning-700">{t('orders.nobleHandFull')}</span>
            )}
          </div>
        }
      />
    </section>
  )
}

function voiceSourceLabel(source: VoiceSource, t: Translate): string {
  if (source.kind === 'fief') {
    return t('orders.voiceFief', {
      name: source.nobleName ?? source.noble ?? '',
      votes: source.votes,
    })
  }
  if (source.kind === 'title') {
    return t('orders.voiceTitle', {
      name: source.nobleName ?? source.noble ?? '',
      title: t(`orders.voiceTitle.${source.title ?? 'bishop'}` as MessageKey),
      votes: source.votes,
    })
  }
  if (source.kind === 'abbey') {
    return t('orders.voiceAbbey', {
      name: source.nobleName ?? source.noble ?? '',
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
  onOpenRules,
}: {
  state: StateData
  winterDraft: string
  onWinterChange: (value: string) => void
  onOpenRules?: (section: RulesSection) => void
}) {
  const { t } = useLanguage()
  const elections = state.openElections ?? []
  if (elections.length === 0) return null
  const lines = draftLines(winterDraft)

  return (
    <section className="space-y-2 rounded-lg border border-winter-700/30 bg-winter-100 p-3">
      <SectionHeading
        tone="winter"
        title={t('orders.electionsTitle')}
        topic={t('orders.electionsTitle')}
        summary={t('orders.electionsSummary')}
        help={t('orders.electionsTimingNote')}
        rulesSection="winter-orders"
        onOpenRules={onOpenRules}
        titleClassName="font-display text-base font-semibold text-winter-700"
      />
      <TooltipProvider delayDuration={100}>
        {elections.map((election) => {
          const name =
            state.bishoprics?.find((bishopric) => bishopric.region === election.region)
              ?.name ?? election.region ?? ''
          const isPope = election.kind === 'pope'
          const isKing = election.kind === 'king'
          const noSeat = isPope || isKing
          const kind = isKing ? 'R' : 'P'
          const seat = election.seat ?? ''
          const filable = election.candidates.filter((candidate) => {
            const line = noSeat ? `K ${kind} ${candidate.code}` : `K E ${candidate.code} ${seat}`
            return !lines.includes(line.toUpperCase())
          })
          return (
            <details
              key={`${election.kind}-${election.region ?? ''}`}
              open
              className="rounded-md border border-winter-700/30 bg-paper-50 px-2 py-1.5 text-xs text-winter-700"
            >
              <summary className="cursor-pointer font-semibold">
                {isKing
                  ? t('orders.electionKingHeading')
                  : isPope
                    ? t('orders.electionPopeHeading')
                    : t('orders.electionBishopHeading', { name, seat })}
              </summary>
              <div className="mt-1.5 space-y-1.5">
                <p className="text-winter-700">
                  {isKing
                    ? t('orders.electionKingRule', { minVotes: election.minVotes ?? 0 })
                    : isPope
                      ? t('orders.electionPopeRule', { required: election.required ?? 0 })
                      : t('orders.electionBishopRule', { seat })}
                </p>
                {isKing && (
                  <p className="text-winter-700">
                    {t('orders.electionKingClergy', {
                      count: election.voiceSources.filter((source) => source.title).length,
                      cardinal: election.voiceSources.some(
                        (source) => source.title === 'cardinal' || source.title === 'pope',
                      )
                        ? t('orders.electionKingCardinalYes')
                        : t('orders.electionKingCardinalNo'),
                    })}
                  </p>
                )}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span
                      tabIndex={0}
                      className="inline-block cursor-help border-b border-dotted border-winter-700 font-medium"
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
                <div className="flex flex-wrap gap-1.5">
                  <OrderLauncher
                    label={t('orders.electionCandidacyLaunch')}
                    title={`${t('orders.electionCandidacyLaunch')} · ${
                      isKing
                        ? t('orders.electionKingHeading')
                        : isPope
                          ? t('orders.electionPopeHeading')
                          : name
                    }`}
                    disabled={filable.length === 0}
                    fields={[
                      {
                        key: 'noble',
                        label: t('orders.field.candidate'),
                        options: filable.map((candidate) => ({
                          value: candidate.code,
                          label: `${candidate.name} (${candidate.code})`,
                        })),
                      },
                    ]}
                    buildOrder={(values) => {
                      const candidate = filable.find((entry) => entry.code === values.noble)
                      if (!candidate) return null
                      return {
                        line: noSeat
                          ? `K ${kind} ${candidate.code}`
                          : `K E ${candidate.code} ${seat}`,
                        comment: t('orders.electionCandidacyComment', { name: candidate.name }),
                      }
                    }}
                    onConfirm={(line) => onWinterChange(appendDraftLine(winterDraft, line))}
                  />
                  <OrderLauncher
                    label={t('orders.electionVoteLaunch')}
                    title={`${t('orders.electionVoteLaunch')} · ${
                      isKing
                        ? t('orders.electionKingHeading')
                        : isPope
                          ? t('orders.electionPopeHeading')
                          : name
                    }`}
                    disabled={election.voices === 0 || election.eligible.length === 0}
                    fields={[
                      {
                        key: 'noble',
                        label: t('orders.field.votedCandidate'),
                        options: election.eligible.map((candidate) => ({
                          value: candidate.code,
                          label: `${candidate.name} (${candidate.code}) · ${
                            state.players.find((entry) => entry.id === candidate.owner)
                              ?.name ?? candidate.owner
                          }`,
                        })),
                      },
                    ]}
                    buildOrder={(values) => {
                      const code = values.noble.trim().toUpperCase()
                      if (!/^[A-Z]{3}$/.test(code)) return null
                      return {
                        line: noSeat ? `V ${kind} ${code}` : `V E ${code} ${seat}`,
                        comment: t('orders.comment.vote', {
                          name:
                            state.nobles.find((noble) => noble.code === code)?.name ?? code,
                        }),
                      }
                    }}
                    onConfirm={(line) => onWinterChange(appendDraftLine(winterDraft, line))}
                  />
                </div>
                {election.candidates.length === 0 && (
                  <p className="text-winter-700">{t('orders.electionNoCandidate')}</p>
                )}
              </div>
            </details>
          )
        })}
      </TooltipProvider>
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
      className="max-h-32 list-disc space-y-1 overflow-y-auto rounded-md border border-warning-700/40 bg-warning-100 px-3 py-2 pl-7 text-xs text-warning-700"
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
  regions = [],
  mapTerritories,
  specialDraft,
  draftDiffers,
  onChainChange,
  onWinterChange,
  onSpecialChange,
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
      <section className="space-y-3 rounded-xl border border-winter-700/30 bg-winter-100/80 p-4 shadow-inner shadow-winter-700/40">
        <SectionHeading
          level="h3"
          tone="winter"
          title={
            <>
              <IconSnowflake aria-hidden="true" className="size-5 text-winter-700" />
              <span>{t('orders.winterTitle')}</span>
            </>
          }
          topic={t('orders.winterTitle')}
          summary={t('orders.winterSummary')}
          help={t('orders.winterDescription')}
          rulesSection="winter-orders"
          onOpenRules={onOpenRules}
          titleClassName="flex items-center gap-2 font-display text-lg font-bold text-winter-700"
        />
        {draftDiffers?.winter && (
          <div className="flex items-center justify-between gap-2 rounded-lg border border-brass-700/40 bg-brass-300/30 px-3 py-2 text-xs text-brass-700">
            <span>{t('orders.draftDiffers')}</span>
            {onRestoreFromServer && (
              <button
                type="button"
                className="font-semibold underline hover:text-brass-700"
                onClick={() => onRestoreFromServer('winter')}
              >
                {t('orders.restoreFromServer')}
              </button>
            )}
          </div>
        )}
        <TitleOrdersSection
          state={state}
          player={player}
          winterDraft={winterDraft}
          onWinterChange={onWinterChange}
          onOpenRules={onOpenRules}
        />
        <OpenElectionsSection
          state={state}
          winterDraft={winterDraft}
          onWinterChange={onWinterChange}
          onOpenRules={onOpenRules}
        />
        <HandSection
          state={state}
          player={player}
          regions={regions}
          mapTerritories={mapTerritories}
          winterDraft={winterDraft}
          onWinterChange={onWinterChange}
          onOpenRules={onOpenRules}
        />
        <textarea
          value={winterDraft}
          onChange={(event) => onWinterChange(event.target.value)}
          className="min-h-28 w-full resize-y rounded-lg border border-winter-700/30 bg-winter-100 p-3 font-mono text-xs text-winter-700 outline-none transition focus:border-winter-700 focus:ring-2 focus:ring-winter-700/20"
          placeholder={t('orders.winterPlaceholder')}
          aria-label={t('orders.winterAria', { player })}
        />
        {winterEstimate && (
          <p
            role="status"
            aria-live="polite"
            className={`text-xs font-semibold ${winterEstimate.spent <= winterEstimate.available ? 'text-success-700' : 'text-vermilion-700'}`}
          >
            {t('orders.winterCostEstimate', {
              spent: winterEstimate.spent,
              available: winterEstimate.available,
            })}
          </p>
        )}
        <PreviewErrors errors={winterErrors} label={t('orders.winterErrorsAria')} />
        <WinterOrderDiagnostics diagnostics={winterDiagnostics} t={t} />
        <RulesButton section="winter-orders" onOpenRules={onOpenRules} />
      </section>
    )
  }

  const nobles = ownedNobles(state, player)
  const hasEmittingNoble = hasEmittingNobleOf(state, player)
  return (
    <section className="space-y-3 border-t border-line/50 pt-5">
      <SectionHeading
        level="h3"
        title={t('orders.actionTitle')}
        topic={t('orders.actionTitle')}
        summary={t('orders.actionSummary')}
        help={t('orders.actionDescription')}
        rulesSection="action-orders"
        onOpenRules={onOpenRules}
        titleClassName="font-display text-lg font-semibold"
      />
      <HandSection
        state={state}
        player={player}
        regions={regions}
        mapTerritories={mapTerritories}
        winterDraft={winterDraft}
        onWinterChange={onWinterChange}
        specialDraft={specialDraft}
        onSpecialChange={onSpecialChange}
        cardsOnly
        onOpenRules={onOpenRules}
      />
      <DeckOrdersSection
        state={state}
        player={player}
        specialDraft={specialDraft}
        onSpecialChange={onSpecialChange}
        onOpenRules={onOpenRules}
      />
      {(state.nobleHand ?? []).length > 0 && (
        <>
          <textarea
            value={winterDraft}
            onChange={(event) => onWinterChange(event.target.value)}
            className="min-h-16 w-full resize-y rounded-lg border border-winter-700/30 bg-winter-100 p-3 font-mono text-xs text-winter-700 outline-none transition focus:border-winter-700 focus:ring-2 focus:ring-winter-700/20"
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
        <p className="rounded-lg border border-dashed border-line bg-paper-100 p-3 text-sm italic text-ink-500">
          {t('orders.noNobleAvailable')}
        </p>
      ) : (
        nobles.map((noble) => (
          <div key={noble.code} className="space-y-1.5">
            <span className="flex items-center justify-between text-xs font-semibold uppercase tracking-[0.12em] text-ink-500">
              <span>
                {noble.name} ({noble.code})
              </span>
              <span
                className={
                  noble.status === 'dungeon' ? 'text-brass-700' : 'text-success-700'
                }
              >
                {t(`orders.nobleStatus.${noble.status}` as MessageKey)}
              </span>
            </span>
            {draftDiffers?.chains?.[noble.code] && (
              <div className="flex items-center justify-between gap-2 rounded-lg border border-brass-700/40 bg-brass-300/30 px-3 py-2 text-xs text-brass-700">
                <span>{t('orders.draftDiffers')}</span>
                {onRestoreFromServer && (
                  <button
                    type="button"
                    className="font-semibold underline hover:text-brass-700"
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
              className="min-h-24 w-full resize-y rounded-lg border border-line-strong bg-paper-50 p-3 font-mono text-xs text-ink-900 outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20 disabled:cursor-not-allowed disabled:opacity-50"
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
      {!hasEmittingNoble && (
        <p className="rounded-lg border border-dashed border-line bg-paper-100 p-3 text-sm italic text-ink-500">
          {t('orders.noEmittingNoble')}
        </p>
      )}
      <RulesButton section="action-orders" onOpenRules={onOpenRules} />
    </section>
  )
}

interface SealOrdersProps {
  state: StateData
  player: PlayerId
  winterDraft: string
  specialDraft: string
  submitted: boolean
  submitting: boolean
  error: string | null
  onSubmit: () => void
}

/**
 * Sticky drawer footer that seals the orders (charte-graphique.md
 * § Disposition sur ordinateur): always visible, whatever the bookmark.
 */
export function SealOrders({
  state,
  player,
  winterDraft,
  specialDraft,
  submitted,
  submitting,
  error,
  onSubmit,
}: SealOrdersProps) {
  const { t } = useLanguage()
  const winter = state.season === 'winter'
  const nothingToSeal =
    !winter &&
    !hasEmittingNobleOf(state, player) &&
    specialDraft.trim() === '' &&
    winterDraft.trim() === ''
  const label = submitting
    ? t('orders.sending')
    : winter
      ? t(submitted ? 'orders.editWinter' : 'orders.submitWinter')
      : t(submitted ? 'orders.edit' : 'orders.submit')

  return (
    <>
      <OrderError error={error} />
      {submitted && (
        <p className="text-xs text-success-700">{t('orders.submittedEditable')}</p>
      )}
      <Button
        type="button"
        variant={winter ? 'outline' : 'default'}
        className={
          winter
            ? 'w-full border-winter-700 bg-winter-100 text-winter-700 hover:bg-winter-100 hover:text-winter-700'
            : 'w-full'
        }
        disabled={submitting || nothingToSeal}
        onClick={onSubmit}
      >
        {label}
      </Button>
    </>
  )
}
